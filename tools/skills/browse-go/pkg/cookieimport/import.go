package cookieimport

import (
	"database/sql"
	"fmt"
	"strings"
)

// ListDomains returns unique cookie domains + counts from a browser's DB.
func ListDomains(browserName string, profile string) (*struct {
	Domains  []DomainEntry
	Browser  string
}, error) {
	browser, err := resolveBrowser(browserName)
	if err != nil {
		return nil, err
	}
	match, err := getBrowserMatch(browser, profile)
	if err != nil {
		return nil, err
	}
	db, tmpPath, err := openDB(match.dbPath, browser.Name)
	if err != nil {
		return nil, err
	}
	defer closeDB(db, tmpPath)

	now := chromiumNow()
	rows, err := db.Query(
		`SELECT host_key AS domain, COUNT(*) AS count
		 FROM cookies
		 WHERE has_expires = 0 OR expires_utc > ?
		 GROUP BY host_key
		 ORDER BY count DESC`, now)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var domains []DomainEntry
	for rows.Next() {
		var d DomainEntry
		if err := rows.Scan(&d.Domain, &d.Count); err != nil {
			continue
		}
		domains = append(domains, d)
	}
	return &struct {
		Domains []DomainEntry
		Browser string
	}{Domains: domains, Browser: browser.Name}, nil
}

// ImportCookies decrypts and returns CDP-compatible cookies for specific domains.
func ImportCookies(browserName string, domains []string, profile string) (*ImportResult, error) {
	if len(domains) == 0 {
		return &ImportResult{Cookies: nil, Count: 0, Failed: 0, DomainCounts: map[string]int{}}, nil
	}

	browser, err := resolveBrowser(browserName)
	if err != nil {
		return nil, err
	}
	match, err := getBrowserMatch(browser, profile)
	if err != nil {
		return nil, err
	}

	derivedKeys, err := getDerivedKeys(match)
	if err != nil {
		return nil, err
	}

	db, tmpPath, err := openDB(match.dbPath, browser.Name)
	if err != nil {
		return nil, err
	}
	defer closeDB(db, tmpPath)

	now := chromiumNow()
	placeholders := make([]string, len(domains))
	args := make([]interface{}, len(domains)+1)
	for i, d := range domains {
		placeholders[i] = "?"
		args[i] = d
	}
	args[len(domains)] = now

	query := fmt.Sprintf(
		`SELECT host_key, name, value, encrypted_value, path, expires_utc,
		        is_secure, is_httponly, has_expires, samesite
		 FROM cookies
		 WHERE host_key IN (%s)
		   AND (has_expires = 0 OR expires_utc > ?)
		 ORDER BY host_key, name`, strings.Join(placeholders, ","))

	rows, err := db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var cookies []*CDPCookie
	failed := 0
	domainCounts := make(map[string]int)

	for rows.Next() {
		var row rawCookieRow
		var ev sql.RawBytes
		if err := rows.Scan(
			&row.HostKey, &row.Name, &row.Value, &ev, &row.Path,
			&row.ExpiresUTC, &row.IsSecure, &row.IsHTTPOnly, &row.HasExpires, &row.SameSite,
		); err != nil {
			failed++
			continue
		}
		row.EncryptedValue = []byte(ev)

		value, err := decryptCookieValue(&row, derivedKeys, match.platform)
		if err != nil {
			failed++
			continue
		}
		cookie := toCDPCookie(&row, value)
		cookies = append(cookies, cookie)
		domainCounts[row.HostKey]++
	}

	return &ImportResult{
		Cookies:      cookies,
		Count:        len(cookies),
		Failed:       failed,
		DomainCounts: domainCounts,
	}, nil
}
