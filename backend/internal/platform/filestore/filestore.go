// Package filestore keeps uploaded claim documents on local disk.
//
// Design notes:
//   - The client's file name is never used as a path. Every stored object gets
//     a server-generated UUID name, so a hostile name ("../../etc/passwd",
//     "C:\\x", a 4KB unicode string) cannot escape the root or collide.
//   - Keys handed back to callers are relative to the root, so the root can be
//     moved (or mounted elsewhere in a container) without rewriting the database.
//   - Open re-validates that the resolved path is still inside the root, which
//     protects reads even if a row's path was tampered with directly in the DB.
package filestore

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/google/uuid"
)

// ErrNotFound is returned when a key has no file behind it — e.g. a database
// row that outlived its blob, or demo data pointing at files never created.
var ErrNotFound = errors.New("filestore: object not found")

type Store struct {
	root string
}

// New creates the root directory if needed and returns a store rooted there.
func New(root string) (*Store, error) {
	abs, err := filepath.Abs(root)
	if err != nil {
		return nil, fmt.Errorf("resolve attachments dir: %w", err)
	}
	if err := os.MkdirAll(abs, 0o750); err != nil {
		return nil, fmt.Errorf("create attachments dir %s: %w", abs, err)
	}
	return &Store{root: abs}, nil
}

// Root is the absolute directory this store writes into (used in logs).
func (s *Store) Root() string { return s.root }

// Save streams r into a new object under prefix and returns its key. ext should
// include the leading dot (".pdf"); it is sanitised, not trusted.
func (s *Store) Save(prefix, ext string, r io.Reader) (string, error) {
	dir := filepath.Join(s.root, sanitiseSegment(prefix))
	if err := os.MkdirAll(dir, 0o750); err != nil {
		return "", fmt.Errorf("create object dir: %w", err)
	}

	name := uuid.NewString() + sanitiseExt(ext)
	full := filepath.Join(dir, name)

	// #nosec G304 -- full is root + sanitised segment + a server-generated UUID
	// name; no part of it comes from the client.
	f, err := os.OpenFile(full, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if err != nil {
		return "", fmt.Errorf("create object: %w", err)
	}
	if _, err := io.Copy(f, r); err != nil {
		_ = f.Close()
		_ = os.Remove(full) // don't leave a half-written file behind
		return "", fmt.Errorf("write object: %w", err)
	}
	if err := f.Close(); err != nil {
		_ = os.Remove(full)
		return "", fmt.Errorf("close object: %w", err)
	}

	key, err := filepath.Rel(s.root, full)
	if err != nil {
		return "", fmt.Errorf("derive object key: %w", err)
	}
	return filepath.ToSlash(key), nil
}

// Open returns a reader for a stored object plus its size.
func (s *Store) Open(key string) (io.ReadSeekCloser, int64, error) {
	full, err := s.resolve(key)
	if err != nil {
		return nil, 0, err
	}
	f, err := os.Open(full) // #nosec G304 -- resolve() confines the path to the root
	if errors.Is(err, os.ErrNotExist) {
		return nil, 0, ErrNotFound
	}
	if err != nil {
		return nil, 0, fmt.Errorf("open object: %w", err)
	}
	info, err := f.Stat()
	if err != nil {
		_ = f.Close()
		return nil, 0, fmt.Errorf("stat object: %w", err)
	}
	if info.IsDir() {
		_ = f.Close()
		return nil, 0, ErrNotFound
	}
	return f, info.Size(), nil
}

// Remove deletes an object; a missing object is not an error.
func (s *Store) Remove(key string) error {
	full, err := s.resolve(key)
	if err != nil {
		return err
	}
	if err := os.Remove(full); err != nil && !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("remove object: %w", err)
	}
	return nil
}

// resolve turns a stored key into an absolute path, refusing anything that
// would land outside the root.
func (s *Store) resolve(key string) (string, error) {
	if key == "" {
		return "", ErrNotFound
	}
	// Reject absolute keys outright; legacy/demo rows may hold "/demo/...".
	if filepath.IsAbs(key) || strings.HasPrefix(key, "/") {
		return "", ErrNotFound
	}
	full := filepath.Join(s.root, filepath.FromSlash(key))
	// filepath.Join cleans "..", so compare the result against the root.
	if full != s.root && !strings.HasPrefix(full, s.root+string(os.PathSeparator)) {
		return "", ErrNotFound
	}
	return full, nil
}

// sanitiseSegment reduces a path prefix to one safe directory name.
func sanitiseSegment(seg string) string {
	seg = strings.Map(func(r rune) rune {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9', r == '-', r == '_':
			return r
		default:
			return -1
		}
	}, seg)
	if seg == "" {
		return "misc"
	}
	return seg
}

// sanitiseExt keeps a short alphanumeric extension or drops it entirely.
func sanitiseExt(ext string) string {
	if ext == "" {
		return ""
	}
	ext = strings.ToLower(strings.TrimPrefix(ext, "."))
	if len(ext) == 0 || len(ext) > 8 {
		return ""
	}
	for _, r := range ext {
		isLower := r >= 'a' && r <= 'z'
		isDigit := r >= '0' && r <= '9'
		if !isLower && !isDigit {
			return ""
		}
	}
	return "." + ext
}
