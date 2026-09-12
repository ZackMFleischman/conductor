package store

import (
	"database/sql"
	_ "embed"
	"errors"
	"fmt"
	_ "modernc.org/sqlite"
	"net/url"
	"os"
	"path/filepath"
)

//go:embed schema.sql
var schema string

type Store struct{ DB *sql.DB }

func Open(path string, create bool) (*Store, error) {
	if !create {
		if _, err := os.Stat(path); err != nil {
			return nil, err
		}
	} else {
		if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
			return nil, err
		}
	}
	u := databaseURL(path)
	q := u.Query()
	if !create {
		q.Set("mode", "rw")
	}
	u.RawQuery = q.Encode()
	db, err := sql.Open("sqlite", u.String())
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)
	s := &Store{db}
	fail := func(e error) (*Store, error) { db.Close(); return nil, e }
	for _, p := range []string{"busy_timeout=5000", "foreign_keys=ON", "synchronous=FULL"} {
		if _, err = db.Exec("PRAGMA " + p); err != nil {
			return fail(err)
		}
	}
	var version int
	if err = db.QueryRow("PRAGMA user_version").Scan(&version); err != nil {
		return fail(err)
	}
	if version > 1 || version < 0 {
		return fail(fmt.Errorf("unsupported schema version %d", version))
	}
	if version == 0 {
		if !create {
			return fail(errors.New("uninitialized schema"))
		}
		if _, err = db.Exec("BEGIN IMMEDIATE"); err != nil {
			return fail(err)
		}
		// A concurrent initializer may have completed while we waited for the lock.
		if err = db.QueryRow("PRAGMA user_version").Scan(&version); err == nil && version == 0 {
			_, err = db.Exec(schema + " PRAGMA user_version=1;")
		} else if err == nil && version != 1 {
			err = fmt.Errorf("unsupported schema version %d", version)
		}
		if err != nil {
			db.Exec("ROLLBACK")
			return fail(err)
		}
		if _, err = db.Exec("COMMIT"); err != nil {
			return fail(err)
		}
	}
	var mode string
	if err = db.QueryRow("PRAGMA journal_mode=WAL").Scan(&mode); err != nil {
		return fail(err)
	}
	if mode != "wal" {
		return fail(errors.New("WAL unavailable"))
	}
	for p, want := range map[string]int{"foreign_keys": 1, "busy_timeout": 5000, "synchronous": 2} {
		var got int
		if err = db.QueryRow("PRAGMA " + p).Scan(&got); err != nil {
			return fail(err)
		}
		if got != want {
			return fail(fmt.Errorf("pragma %s=%d", p, got))
		}
	}
	return s, nil
}

func databaseURL(path string) url.URL {
	p := filepath.ToSlash(path)
	if filepath.VolumeName(path) != "" && len(p) > 0 && p[0] != '/' {
		p = "/" + p
	}
	return url.URL{Scheme: "file", Path: p}
}

// OpenReadOnly never creates a database, migrates its schema, or changes journal mode.
func OpenReadOnly(path string) (*Store, error) {
	if _, e := os.Stat(path); e != nil {
		return nil, e
	}
	u := databaseURL(path)
	q := u.Query()
	q.Set("mode", "ro")
	u.RawQuery = q.Encode()
	db, e := sql.Open("sqlite", u.String())
	if e != nil {
		return nil, e
	}
	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)
	fail := func(err error) (*Store, error) { db.Close(); return nil, err }
	for _, p := range []string{"busy_timeout=5000", "foreign_keys=ON", "synchronous=FULL", "query_only=ON"} {
		if _, e = db.Exec("PRAGMA " + p); e != nil {
			return fail(e)
		}
	}
	var version int
	if e = db.QueryRow("PRAGMA user_version").Scan(&version); e != nil {
		return fail(e)
	}
	if version != 1 {
		return fail(fmt.Errorf("unsupported schema version %d", version))
	}
	var check string
	if e = db.QueryRow("PRAGMA quick_check").Scan(&check); e != nil {
		return fail(e)
	}
	if check != "ok" {
		return fail(errors.New(check))
	}
	var n int
	if e = db.QueryRow("SELECT count(*) FROM projects").Scan(&n); e != nil {
		return fail(e)
	}
	return &Store{DB: db}, nil
}
