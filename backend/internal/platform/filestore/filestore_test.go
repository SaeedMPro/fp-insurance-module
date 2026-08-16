package filestore

import (
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func newStore(t *testing.T) *Store {
	t.Helper()
	s, err := New(t.TempDir())
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	return s
}

func TestSaveAndOpenRoundTrip(t *testing.T) {
	s := newStore(t)
	key, err := s.Save("claim-1", ".pdf", strings.NewReader("hello"))
	if err != nil {
		t.Fatalf("Save: %v", err)
	}
	if filepath.IsAbs(key) {
		t.Fatalf("key must be relative, got %q", key)
	}
	if !strings.HasSuffix(key, ".pdf") {
		t.Fatalf("key should keep the extension, got %q", key)
	}

	r, size, err := s.Open(key)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer r.Close()
	if size != 5 {
		t.Errorf("size = %d, want 5", size)
	}
	body, _ := io.ReadAll(r)
	if string(body) != "hello" {
		t.Errorf("content = %q, want %q", body, "hello")
	}
}

// The client controls the file name and the claim id in the URL, so neither may
// be able to steer a write or a read outside the root.
func TestHostileInputsStayInsideRoot(t *testing.T) {
	s := newStore(t)

	for _, prefix := range []string{"../..", "/etc", "..\\..\\windows", ""} {
		key, err := s.Save(prefix, ".png", strings.NewReader("x"))
		if err != nil {
			t.Fatalf("Save(%q): %v", prefix, err)
		}
		full := filepath.Join(s.Root(), filepath.FromSlash(key))
		if !strings.HasPrefix(full, s.Root()+string(os.PathSeparator)) {
			t.Errorf("prefix %q escaped the root: %s", prefix, full)
		}
	}

	for _, ext := range []string{"../../evil", ".p/df", ".verylongextension", "", "."} {
		key, err := s.Save("claim", ext, strings.NewReader("x"))
		if err != nil {
			t.Fatalf("Save(ext=%q): %v", ext, err)
		}
		if strings.ContainsAny(key, `/\`[:1]) && strings.Count(key, "/") != 1 {
			t.Errorf("ext %q produced an unexpected key: %s", ext, key)
		}
	}
}

func TestOpenRejectsTraversalAndAbsoluteKeys(t *testing.T) {
	s := newStore(t)
	outside := filepath.Join(filepath.Dir(s.Root()), "secret.txt")
	if err := os.WriteFile(outside, []byte("classified"), 0o600); err != nil {
		t.Fatalf("write bait file: %v", err)
	}

	for _, key := range []string{
		"../secret.txt",
		"../../secret.txt",
		"claim/../../secret.txt",
		outside,                   // absolute
		"/demo/attachments/x.pdf", // legacy demo-style path
		"",                        // empty
	} {
		if _, _, err := s.Open(key); err != ErrNotFound {
			t.Errorf("Open(%q) = %v, want ErrNotFound", key, err)
		}
	}
}

func TestOpenMissingObject(t *testing.T) {
	s := newStore(t)
	if _, _, err := s.Open("claim/does-not-exist.pdf"); err != ErrNotFound {
		t.Errorf("got %v, want ErrNotFound", err)
	}
}

func TestRemove(t *testing.T) {
	s := newStore(t)
	key, err := s.Save("claim", ".jpg", strings.NewReader("data"))
	if err != nil {
		t.Fatalf("Save: %v", err)
	}
	if err := s.Remove(key); err != nil {
		t.Fatalf("Remove: %v", err)
	}
	if _, _, err := s.Open(key); err != ErrNotFound {
		t.Errorf("object should be gone, got %v", err)
	}
	// Removing again is not an error.
	if err := s.Remove(key); err != nil {
		t.Errorf("second Remove: %v", err)
	}
}
