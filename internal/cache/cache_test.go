package cache

import (
	"os"
	"path/filepath"
	"testing"
)

func TestNew(t *testing.T) {
	c := New()
	if c == nil {
		t.Fatal("New() returned nil")
	}
}

func TestGetSet(t *testing.T) {
	c := New()
	c.Set("key", 42)

	val, found := c.Get("key")
	if !found {
		t.Fatal("expected key to be found")
	}
	if val.(int) != 42 {
		t.Errorf("got %v, want 42", val)
	}
}

func TestGet_Missing(t *testing.T) {
	c := New()
	_, found := c.Get("missing")
	if found {
		t.Error("expected missing key to not be found")
	}
}

func TestFlush(t *testing.T) {
	c := New()
	c.Set("key", "value")
	c.Flush()

	_, found := c.Get("key")
	if found {
		t.Error("expected key to be gone after Flush")
	}
}

func TestSaveAndLoadFromFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.gob")

	c := New()
	c.Set("stars", 100)
	c.Set("name", "test")

	if err := c.SaveToFile(path); err != nil {
		t.Fatalf("SaveToFile: %v", err)
	}

	loaded, err := LoadFromFile(path)
	if err != nil {
		t.Fatalf("LoadFromFile: %v", err)
	}

	val, found := loaded.Get("stars")
	if !found {
		t.Fatal("expected 'stars' key after load")
	}
	if val.(int) != 100 {
		t.Errorf("got %v, want 100", val)
	}

	val, found = loaded.Get("name")
	if !found {
		t.Fatal("expected 'name' key after load")
	}
	if val.(string) != "test" {
		t.Errorf("got %v, want 'test'", val)
	}
}

func TestLoadFromFile_NonexistentFile(t *testing.T) {
	c, err := LoadFromFile("/nonexistent/path/cache.gob")
	if err != nil {
		t.Fatalf("expected no error for nonexistent file, got %v", err)
	}
	if c == nil {
		t.Fatal("expected fresh cache, got nil")
	}
}

func TestLoadFromFile_CorruptData(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "bad.gob")

	if err := os.WriteFile(path, []byte("not valid gob data"), 0644); err != nil {
		t.Fatal(err)
	}

	c, err := LoadFromFile(path)
	if err != nil {
		t.Fatalf("expected no error for corrupt file, got %v", err)
	}
	if c == nil {
		t.Fatal("expected fresh cache, got nil")
	}

	// Should be empty.
	_, found := c.Get("anything")
	if found {
		t.Error("expected empty cache from corrupt file")
	}
}
