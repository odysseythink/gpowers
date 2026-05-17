package cookieimport

import (
	"crypto/pbkdf2"
	"crypto/sha1"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// keyCache caches derived AES keys per browser.
var keyCache sync.Map

type derivedKeyCache struct {
	key []byte
}

func deriveKey(password string, iterations int) ([]byte, error) {
	return pbkdf2.Key(sha1.New, password, []byte("saltysalt"), iterations, 16)
}

func getCachedDerivedKey(cacheKey, password string, iterations int) ([]byte, error) {
	if v, ok := keyCache.Load(cacheKey); ok {
		return v.([]byte), nil
	}
	key, err := deriveKey(password, iterations)
	if err != nil {
		return nil, err
	}
	keyCache.Store(cacheKey, key)
	return key, nil
}

// getDerivedKeys returns a map of prefix -> AES key for the given browser match.
func getDerivedKeys(match *browserMatch) (map[string][]byte, error) {
	switch match.platform {
	case platformDarwin:
		password, err := getMacKeychainPassword(match.browser.KeychainService)
		if err != nil {
			return nil, err
		}
		key, err := getCachedDerivedKey(
			fmt.Sprintf("darwin:%s:v10", match.browser.KeychainService),
			password, 1003)
		if err != nil {
			return nil, err
		}
		return map[string][]byte{"v10": key}, nil

	case platformWin32:
		key, err := getWindowsAESKey(match.browser)
		if err != nil {
			return nil, err
		}
		return map[string][]byte{"v10": key}, nil

	default: // Linux
		keys := make(map[string][]byte)
		v10Key, err := getCachedDerivedKey("linux:v10", "peanuts", 1)
		if err != nil {
			return nil, err
		}
		keys["v10"] = v10Key

		linuxPassword, err := getLinuxSecretPassword(match.browser)
		if err == nil && linuxPassword != "" {
			v11Key, err := getCachedDerivedKey(
				fmt.Sprintf("linux:%s:v11", match.browser.KeychainService),
				linuxPassword, 1)
			if err != nil {
				return nil, err
			}
			keys["v11"] = v11Key
		}
		return keys, nil
	}
}

// getMacKeychainPassword reads the browser safe-storage password from macOS Keychain.
func getMacKeychainPassword(service string) (string, error) {
	ctx, cancel := contextWithTimeout(10 * time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, "security", "find-generic-password", "-s", service, "-w")
	out, err := cmd.Output()
	if err != nil {
		if ctx.Err() != nil {
			return "", NewRetryError("keychain_timeout",
				fmt.Sprintf("macOS is waiting for Keychain permission. Look for a dialog asking to allow access to %q.", service))
		}
		exitErr, ok := err.(*exec.ExitError)
		if ok {
			stderr := strings.ToLower(string(exitErr.Stderr))
			if strings.Contains(stderr, "user canceled") || strings.Contains(stderr, "denied") || strings.Contains(stderr, "interaction not allowed") {
				return "", NewRetryError("keychain_denied",
					fmt.Sprintf("Keychain access denied. Click \"Allow\" in the macOS dialog for %q.", service))
			}
			if strings.Contains(stderr, "could not be found") || strings.Contains(stderr, "not found") {
				return "", NewError("keychain_not_found",
					fmt.Sprintf("No Keychain entry for %q. Is this a Chromium-based browser?", service))
			}
			return "", NewRetryError("keychain_error",
				fmt.Sprintf("Could not read Keychain: %s", strings.TrimSpace(string(exitErr.Stderr))))
		}
		return "", NewRetryError("keychain_error", fmt.Sprintf("Could not read Keychain: %v", err))
	}
	return strings.TrimSpace(string(out)), nil
}

// getLinuxSecretPassword attempts to retrieve the browser password from libsecret.
func getLinuxSecretPassword(browser *BrowserInfo) (string, error) {
	attempts := [][]string{
		{"secret-tool", "lookup", "Title", browser.KeychainService},
	}
	if browser.LinuxApp != "" {
		attempts = append(attempts,
			[]string{"secret-tool", "lookup", "xdg:schema", "chrome_libsecret_os_crypt_password_v2", "application", browser.LinuxApp},
			[]string{"secret-tool", "lookup", "xdg:schema", "chrome_libsecret_os_crypt_password", "application", browser.LinuxApp},
		)
	}
	for _, cmd := range attempts {
		password, err := runPasswordLookup(cmd, 3*time.Second)
		if err == nil && password != "" {
			return password, nil
		}
	}
	return "", NewError("keychain_not_found", "No secret-tool password found")
}

func runPasswordLookup(cmdArgs []string, timeout time.Duration) (string, error) {
	ctx, cancel := contextWithTimeout(timeout)
	defer cancel()
	cmd := exec.CommandContext(ctx, cmdArgs[0], cmdArgs[1:]...)
	cmd.Env = minimalEnv()
	out, err := cmd.Output()
	if err != nil {
		return "", err
	}
	password := strings.TrimSpace(string(out))
	if password == "" {
		return "", fmt.Errorf("empty password")
	}
	return password, nil
}

// getWindowsAESKey reads and decrypts the AES key from Chrome's Local State file.
func getWindowsAESKey(browser *BrowserInfo) ([]byte, error) {
	cacheKey := fmt.Sprintf("win32:%s", browser.KeychainService)
	if v, ok := keyCache.Load(cacheKey); ok {
		return v.([]byte), nil
	}

	dataDir := dataDirForPlatform(browser, platformWin32)
	if dataDir == "" {
		return nil, NewError("not_installed", fmt.Sprintf("No Windows data dir for %s", browser.Name))
	}
	localStatePath := filepath.Join(baseDir(platformWin32), dataDir, "Local State")
	data, err := os.ReadFile(localStatePath)
	if err != nil {
		return nil, NewError("keychain_error", fmt.Sprintf("Cannot read Local State for %s: %v", browser.Name, err))
	}

	// Extract os_crypt.encrypted_key from JSON.
	encryptedKeyB64 := jsonExtractString(data, `"os_crypt"`, `"encrypted_key"`)
	if encryptedKeyB64 == "" {
		return nil, NewError("keychain_not_found", fmt.Sprintf("No encrypted key in Local State for %s", browser.Name))
	}

	encryptedKey, err := base64Decode(encryptedKeyB64)
	if err != nil {
		return nil, NewError("keychain_error", fmt.Sprintf("Invalid base64 in Local State: %v", err))
	}
	// Strip the 5-byte "DPAPI" prefix.
	if len(encryptedKey) < 5 {
		return nil, NewError("keychain_error", "Encrypted key too short")
	}
	encryptedKey = encryptedKey[5:]

	key, err := dpapiDecrypt(encryptedKey)
	if err != nil {
		return nil, err
	}
	keyCache.Store(cacheKey, key)
	return key, nil
}

// dpapiDecrypt decrypts data using Windows DPAPI via PowerShell.
func dpapiDecrypt(encryptedBytes []byte) ([]byte, error) {
	script := `Add-Type -AssemblyName System.Security; $stdin = [Console]::In.ReadToEnd().Trim(); $bytes = [System.Convert]::FromBase64String($stdin); $dec = [System.Security.Cryptography.ProtectedData]::Unprotect($bytes, $null, [System.Security.Cryptography.DataProtectionScope]::CurrentUser); Write-Output ([System.Convert]::ToBase64String($dec))`

	ctx, cancel := contextWithTimeout(10 * time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, "powershell", "-NoProfile", "-Command", script)
	cmd.Stdin = strings.NewReader(base64Encode(encryptedBytes))
	out, err := cmd.Output()
	if err != nil {
		if ctx.Err() != nil {
			return nil, NewRetryError("keychain_timeout", "DPAPI decryption timed out")
		}
		exitErr, ok := err.(*exec.ExitError)
		if ok {
			return nil, NewError("keychain_error", fmt.Sprintf("DPAPI decryption failed: %s", strings.TrimSpace(string(exitErr.Stderr))))
		}
		return nil, NewError("keychain_error", fmt.Sprintf("DPAPI decryption failed: %v", err))
	}
	return base64Decode(strings.TrimSpace(string(out)))
}
