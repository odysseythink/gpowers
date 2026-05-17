package cookieimport

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"fmt"
	"time"

	"github.com/chromedp/cdproto/network"
)

// Chromium epoch offset: microseconds between 1601-01-01 and 1970-01-01.
const chromiumEpochOffset = 11644473600000000

func chromiumNow() int64 {
	return time.Now().Unix()*1e6 + chromiumEpochOffset
}

func chromiumEpochToUnix(epoch int64, hasExpires int) float64 {
	if hasExpires == 0 || epoch == 0 {
		return -1 // session cookie
	}
	return float64(epoch-chromiumEpochOffset) / 1e6
}

func mapSameSite(value int) network.CookieSameSite {
	switch value {
	case 0:
		return network.CookieSameSiteNone
	case 1:
		return network.CookieSameSiteLax
	case 2:
		return network.CookieSameSiteStrict
	default:
		return network.CookieSameSiteLax
	}
}

// decryptCookieValue decrypts a single cookie's encrypted_value.
func decryptCookieValue(row *rawCookieRow, keys map[string][]byte, platform browserPlatform) (string, error) {
	// Prefer unencrypted value if present.
	if row.Value != "" {
		return row.Value, nil
	}

	ev := row.EncryptedValue
	if len(ev) == 0 {
		return "", nil
	}

	prefix := string(ev[:3])

	if prefix == "v20" {
		return "", NewError("v20_encryption", "Cookie uses App-Bound Encryption (v20). Use CDP extraction instead.")
	}

	key, ok := keys[prefix]
	if !ok {
		return "", fmt.Errorf("no decryption key available for %s cookies", prefix)
	}

	if platform == platformWin32 && prefix == "v10" {
		// Windows v10: AES-256-GCM — v10(3) + nonce(12) + ciphertext + tag(16)
		if len(ev) < 3+12+16 {
			return "", fmt.Errorf("encrypted value too short for AES-256-GCM")
		}
		nonce := ev[3:15]
		ciphertext := ev[15 : len(ev)-16]
		tag := ev[len(ev)-16:]
		block, err := aes.NewCipher(key)
		if err != nil {
			return "", err
		}
		gcm, err := cipher.NewGCM(block)
		if err != nil {
			return "", err
		}
		// Append tag to ciphertext for Go's GCM Open
		combined := append(ciphertext, tag...)
		plaintext, err := gcm.Open(nil, nonce, combined, nil)
		if err != nil {
			return "", err
		}
		return string(plaintext), nil
	}

	// macOS / Linux: AES-128-CBC — v10/v11(3) + ciphertext
	ciphertext := ev[3:]
	iv := bytes.Repeat([]byte{0x20}, 16) // 16 space characters
	block, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}
	mode := cipher.NewCBCDecrypter(block, iv)
	plaintext := make([]byte, len(ciphertext))
	mode.CryptBlocks(plaintext, ciphertext)

	// Remove PKCS7 padding.
	plaintext = pkcs7Unpad(plaintext)

	// Chromium prefixes encrypted cookie payloads with 32 bytes of metadata.
	if len(plaintext) <= 32 {
		return "", nil
	}
	return string(plaintext[32:]), nil
}

func pkcs7Unpad(data []byte) []byte {
	if len(data) == 0 {
		return data
	}
	padLen := int(data[len(data)-1])
	if padLen > len(data) || padLen == 0 {
		return data
	}
	for i := 0; i < padLen; i++ {
		if data[len(data)-1-i] != byte(padLen) {
			return data
		}
	}
	return data[:len(data)-padLen]
}

// toCDPCookie converts a raw row + decrypted value to a CDPCookie.
func toCDPCookie(row *rawCookieRow, value string) *CDPCookie {
	return &CDPCookie{
		Name:     row.Name,
		Value:    value,
		Domain:   row.HostKey,
		Path:     row.Path,
		Expires:  chromiumEpochToUnix(row.ExpiresUTC, row.HasExpires),
		Secure:   row.IsSecure == 1,
		HTTPOnly: row.IsHTTPOnly == 1,
		SameSite: mapSameSite(row.SameSite),
	}
}
