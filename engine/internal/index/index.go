// Package index maintains a disposable SQLite cache over the .devrites files.
//
// The files are always the source of truth. The index is gitignored, carries
// its own schema version, is staleness-checked on read (mtime + content hash),
// and is rebuilt from scratch on demand (Reindex) or whenever its schema
// mismatches or it is unreadable. A hand-edit to a file always wins: a stale
// feature is re-indexed before its status is returned, and if the index cannot
// be used at all the caller falls back to reading the files directly.
package index

import (
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	_ "modernc.org/sqlite" // pure-Go SQLite driver, registered as "sqlite" (no CGo)

	"github.com/devrites/devrites/internal/state"
)

// schemaVersion is the index DB's own schema version, independent of the state
// schema. A DB whose user_version differs is dropped and rebuilt, never trusted.
const schemaVersion = 1

// DBName is the index file created under the .devrites root. It is disposable
// and gitignored.
const DBName = "state.db"

// Index is an open handle to the SQLite cache for one .devrites root.
type Index struct {
	root string
	db   *sql.DB
}

// Open opens — creating if needed — the index under root. If the existing DB
// has a mismatched schema version or is unreadable, it is dropped and recreated
// with a fresh schema. The caller must Close it.
func Open(root string) (*Index, error) {
	path := filepath.Join(root, DBName)

	// Try to open and trust an existing DB with a current schema. A failure to
	// open at all (e.g. the file is not a database) is not fatal — it just means
	// the cache must be rebuilt, same as a stale schema.
	if db, err := openDB(path); err == nil {
		ix := &Index{root: root, db: db}
		if ix.schemaCurrent() {
			return ix, nil
		}
		db.Close()
	}

	// Drop and rebuild: nothing here is the source of truth.
	if err := removeDBFiles(path); err != nil {
		return nil, err
	}
	db, err := openDB(path)
	if err != nil {
		return nil, err
	}
	ix := &Index{root: root, db: db}
	if err := ix.install(); err != nil {
		ix.db.Close()
		return nil, err
	}
	return ix, nil
}

// Close releases the underlying database handle.
func (ix *Index) Close() error { return ix.db.Close() }

func openDB(path string) (*sql.DB, error) {
	dsn := "file:" + path + "?_pragma=busy_timeout(5000)&_pragma=journal_mode(WAL)&_pragma=foreign_keys(1)"
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("open index at %s: %w", path, err)
	}
	// Short-lived CLI process: a single connection keeps writes serialized and
	// avoids WAL writer contention within the process.
	db.SetMaxOpenConns(1)
	if err := db.Ping(); err != nil {
		db.Close()
		return nil, fmt.Errorf("open index at %s: %w", path, err)
	}
	return db, nil
}

// schemaCurrent reports whether the open DB has this engine's index schema. Any
// doubt (read error, missing table, version mismatch) returns false so the DB is
// rebuilt rather than trusted.
func (ix *Index) schemaCurrent() bool {
	var v int
	if err := ix.db.QueryRow("PRAGMA user_version").Scan(&v); err != nil || v != schemaVersion {
		return false
	}
	var name string
	err := ix.db.QueryRow(`SELECT name FROM sqlite_master WHERE type='table' AND name='features'`).Scan(&name)
	return err == nil
}

func (ix *Index) install() error {
	stmts := []string{
		`CREATE TABLE IF NOT EXISTS features (
			slug        TEXT PRIMARY KEY,
			phase       TEXT NOT NULL,
			fingerprint TEXT NOT NULL
		)`,
		`CREATE TABLE IF NOT EXISTS sections (
			slug    TEXT NOT NULL,
			name    TEXT NOT NULL,
			present INTEGER NOT NULL,
			PRIMARY KEY (slug, name),
			FOREIGN KEY (slug) REFERENCES features(slug) ON DELETE CASCADE
		)`,
	}
	for _, s := range stmts {
		if _, err := ix.db.Exec(s); err != nil {
			return fmt.Errorf("install index schema: %w", err)
		}
	}
	// user_version takes no bound parameters; schemaVersion is a trusted const.
	if _, err := ix.db.Exec(fmt.Sprintf("PRAGMA user_version = %d", schemaVersion)); err != nil {
		return fmt.Errorf("install index schema: %w", err)
	}
	return nil
}

// Reindex rebuilds the whole index from the files and returns the number of
// features indexed. It clears and repopulates the tables inside a SINGLE
// transaction rather than deleting the database file, so a concurrent reader (or
// another reindexer) is never left staring at a half-deleted DB — SQLite's WAL +
// busy timeout serialize the write instead. Deleting the DB and calling Reindex
// still reproduces byte-identical status: Open recreates a fresh schema, then
// this repopulates it.
func Reindex(root string) (int, error) {
	ix, err := Open(root) // heals a missing/corrupt/stale-schema DB to a fresh one
	if err != nil {
		return 0, err
	}
	defer ix.Close()

	slugs, err := state.ListFeatures(root)
	if err != nil {
		return 0, err
	}

	tx, err := ix.db.Begin()
	if err != nil {
		return 0, fmt.Errorf("reindex: %w", err)
	}
	defer tx.Rollback()

	if _, err := tx.Exec(`DELETE FROM sections`); err != nil {
		return 0, fmt.Errorf("reindex: %w", err)
	}
	if _, err := tx.Exec(`DELETE FROM features`); err != nil {
		return 0, fmt.Errorf("reindex: %w", err)
	}
	for _, slug := range slugs {
		f, err := state.LoadFeature(root, slug)
		if err != nil {
			return 0, err
		}
		fp, err := fingerprint(root, slug)
		if err != nil {
			return 0, err
		}
		if err := putFeatureExec(tx, f, fp); err != nil {
			return 0, err
		}
	}
	if err := tx.Commit(); err != nil {
		return 0, fmt.Errorf("reindex: %w", err)
	}
	return len(slugs), nil
}

// Status returns the status report for feature <slug> under root, served from
// the index. The feature is re-indexed first if it is missing from the index or
// its files have changed since (staleness heal), so a hand-edit always wins. If
// the index cannot be used at all, it falls back to the already-loaded files, so
// status never fails merely because the cache is unavailable.
func Status(root, slug string) (*state.Report, error) {
	// Load once from the files: this establishes existence (surfacing the
	// not-found error identically to files-only status) and is reused for the
	// write-through and for the fail-open path, so no file is read twice.
	f, err := state.LoadFeature(root, slug)
	if err != nil {
		return nil, err
	}

	ix, err := Open(root)
	if err != nil {
		return fallback(f, slug, err)
	}
	defer ix.Close()

	fp, err := fingerprint(root, slug)
	if err != nil {
		return fallback(f, slug, err)
	}
	stored, indexed, err := ix.storedFingerprint(slug)
	if err != nil {
		return fallback(f, slug, err)
	}
	if !indexed || stored != fp {
		if err := ix.putFeature(f, fp); err != nil {
			return fallback(f, slug, err)
		}
	}
	fromIndex, err := ix.readFeature(slug)
	if err != nil {
		return fallback(f, slug, err)
	}
	return state.NewReport(fromIndex), nil
}

// fallback serves a files-only report when the index is unusable. It reuses the
// already-loaded feature so no file is read twice, and its output is identical
// to index-served output. Setting DEVRITES_DEBUG surfaces the cause on stderr so
// a broken cache is distinguishable from a merely absent one.
func fallback(f *state.Feature, slug string, cause error) (*state.Report, error) {
	if os.Getenv("DEVRITES_DEBUG") != "" {
		fmt.Fprintf(os.Stderr, "devrites: index unavailable for %q, serving from files: %v\n", slug, cause)
	}
	return state.NewReport(f), nil
}

// execer is the write surface shared by *sql.DB and *sql.Tx, so a feature can be
// written either standalone (its own transaction, via putFeature) or as one step
// of a larger transaction (a full Reindex).
type execer interface {
	Exec(query string, args ...any) (sql.Result, error)
}

// putFeature writes a loaded feature through the index in its own transaction.
func (ix *Index) putFeature(f *state.Feature, fp string) error {
	tx, err := ix.db.Begin()
	if err != nil {
		return fmt.Errorf("index feature %q: %w", f.Slug, err)
	}
	defer tx.Rollback()
	if err := putFeatureExec(tx, f, fp); err != nil {
		return err
	}
	return tx.Commit()
}

// putFeatureExec writes a loaded feature — its phase, section presence, and
// freshness fingerprint — using the given execer.
func putFeatureExec(x execer, f *state.Feature, fp string) error {
	if _, err := x.Exec(
		`INSERT INTO features(slug, phase, fingerprint) VALUES(?,?,?)
		 ON CONFLICT(slug) DO UPDATE SET phase=excluded.phase, fingerprint=excluded.fingerprint`,
		f.Slug, string(f.Phase), fp,
	); err != nil {
		return fmt.Errorf("index feature %q: %w", f.Slug, err)
	}
	if _, err := x.Exec(`DELETE FROM sections WHERE slug=?`, f.Slug); err != nil {
		return fmt.Errorf("index feature %q: %w", f.Slug, err)
	}
	for _, s := range state.Sections {
		present := 0
		if f.Present[s] {
			present = 1
		}
		if _, err := x.Exec(`INSERT INTO sections(slug, name, present) VALUES(?,?,?)`, f.Slug, string(s), present); err != nil {
			return fmt.Errorf("index feature %q: %w", f.Slug, err)
		}
	}
	return nil
}

// readFeature reconstructs a state.Feature from the indexed rows.
func (ix *Index) readFeature(slug string) (*state.Feature, error) {
	var phase string
	err := ix.db.QueryRow(`SELECT phase FROM features WHERE slug=?`, slug).Scan(&phase)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, fmt.Errorf("feature %q not indexed", slug)
	}
	if err != nil {
		return nil, fmt.Errorf("read feature %q from index: %w", slug, err)
	}

	present := make(map[state.Section]bool)
	rows, err := ix.db.Query(`SELECT name, present FROM sections WHERE slug=?`, slug)
	if err != nil {
		return nil, fmt.Errorf("read feature %q from index: %w", slug, err)
	}
	defer rows.Close()
	for rows.Next() {
		var name string
		var p int
		if err := rows.Scan(&name, &p); err != nil {
			return nil, fmt.Errorf("read feature %q from index: %w", slug, err)
		}
		present[state.Section(name)] = p != 0
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("read feature %q from index: %w", slug, err)
	}
	return &state.Feature{Slug: slug, Phase: state.Phase(phase), Present: present}, nil
}

// storedFingerprint returns the fingerprint recorded for slug and whether a row
// exists.
func (ix *Index) storedFingerprint(slug string) (fp string, exists bool, err error) {
	err = ix.db.QueryRow(`SELECT fingerprint FROM features WHERE slug=?`, slug).Scan(&fp)
	if errors.Is(err, sql.ErrNoRows) {
		return "", false, nil
	}
	if err != nil {
		return "", false, err
	}
	return fp, true, nil
}

// fingerprint hashes each of the feature's files by name, size, mtime, and
// content. Content is included so a restored mtime cannot hide an edit; the file
// set is fixed so an added or removed section also changes the fingerprint.
func fingerprint(root, slug string) (string, error) {
	h := sha256.New()
	for _, path := range state.CandidateFiles(root, slug) {
		name := filepath.Base(path)
		info, err := os.Stat(path)
		if errors.Is(err, os.ErrNotExist) {
			fmt.Fprintf(h, "%s\x00absent\x00", name)
			continue
		}
		if err != nil {
			return "", err
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return "", err
		}
		sum := sha256.Sum256(data)
		fmt.Fprintf(h, "%s\x00%d\x00%d\x00%x\x00", name, info.Size(), info.ModTime().UnixNano(), sum)
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

// removeDBFiles deletes the DB and its WAL sidecars. Absent files are fine.
func removeDBFiles(path string) error {
	for _, p := range []string{path, path + "-wal", path + "-shm", path + "-journal"} {
		if err := os.Remove(p); err != nil && !errors.Is(err, os.ErrNotExist) {
			return err
		}
	}
	return nil
}
