package cookieimport

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	_ "modernc.org/sqlite"
)

// openDB opens the cookie database. On Windows it always copies first
// because Chrome holds exclusive WAL locks.
func openDB(dbPath string, browserName string) (*sql.DB, string, error) {
	if runtime.GOOS == "windows" {
		return openDBFromCopy(dbPath, browserName)
	}
	db, err := sql.Open("sqlite", dbPath+"?_pragma=query_only(true)")
	if err != nil {
		if strings.Contains(err.Error(), "busy") || strings.Contains(err.Error(), "locked") {
			return openDBFromCopy(dbPath, browserName)
		}
		if strings.Contains(err.Error(), "corrupt") || strings.Contains(err.Error(), "malformed") {
			return nil, "", NewError("db_corrupt", fmt.Sprintf("Cookie database for %s is corrupt", browserName))
		}
		return nil, "", err
	}
	if err := db.Ping(); err != nil {
		db.Close()
		if strings.Contains(err.Error(), "busy") || strings.Contains(err.Error(), "locked") {
			return openDBFromCopy(dbPath, browserName)
		}
		return nil, "", err
	}
	return db, "", nil
}

// openDBFromCopy copies the DB (and WAL/SHM) to a temp location and opens there.
func openDBFromCopy(dbPath string, browserName string) (*sql.DB, string, error) {
	tmpPath := filepath.Join(os.TempDir(), fmt.Sprintf("browse-cookies-%s-%d.db", strings.ToLower(browserName), os.Getpid()))
	if err := copyFile(dbPath, tmpPath); err != nil {
		return nil, "", NewRetryError("db_locked",
			fmt.Sprintf("Cookie database is locked (%s may be running). Try closing %s first.", browserName, browserName))
	}
	// Copy WAL and SHM if they exist
	for _, suffix := range []string{"-wal", "-shm"} {
		src := dbPath + suffix
		if _, err := os.Stat(src); err == nil {
			_ = copyFile(src, tmpPath+suffix)
		}
	}
	db, err := sql.Open("sqlite", tmpPath+"?_pragma=query_only(true)")
	if err != nil {
		_ = os.Remove(tmpPath)
		return nil, "", err
	}
	if err := db.Ping(); err != nil {
		db.Close()
		_ = os.Remove(tmpPath)
		return nil, "", err
	}
	return db, tmpPath, nil
}

func copyFile(src, dst string) error {
	data, err := os.ReadFile(src)
	if err != nil {
		return err
	}
	return os.WriteFile(dst, data, 0600)
}

// closeDB closes the DB and cleans up temp files.
func closeDB(db *sql.DB, tmpPath string) {
	db.Close()
	if tmpPath != "" {
		_ = os.Remove(tmpPath)
		_ = os.Remove(tmpPath + "-wal")
		_ = os.Remove(tmpPath + "-shm")
	}
}
