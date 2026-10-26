package keystore

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

const host = "10.0.0.5"

func TestStoreAtomicAndCorrupt(t *testing.T) {
	dir := t.TempDir()
	ks := Store{Path: filepath.Join(dir, "keys.json")}
	if ks.Get(host) != "" {
		t.Error("missing file must read empty")
	}
	if err := ks.Put(host, "a"); err != nil {
		t.Fatal(err)
	}
	if err := ks.Put("10.0.0.6", "b"); err != nil {
		t.Fatal(err)
	}
	if ks.Get(host) != "a" || ks.Get("10.0.0.6") != "b" {
		t.Error("keys not kept")
	}
	if _, err := os.Stat(ks.Path + ".tmp"); !os.IsNotExist(err) {
		t.Error("temp file left behind")
	}
	var m map[string]string
	b, _ := os.ReadFile(ks.Path)
	if json.Unmarshal(b, &m) != nil || m[host] != "a" {
		t.Errorf("file = %s", b)
	}
	if err := ks.Delete(host); err != nil {
		t.Fatal(err)
	}
	if ks.Get(host) != "" || ks.Get("10.0.0.6") != "b" {
		t.Error("delete wrong")
	}
	if err := ks.Delete("missing"); err != nil {
		t.Errorf("delete missing: %v", err)
	}

	os.WriteFile(ks.Path, []byte("{not json"), 0o600)
	if ks.Get("10.0.0.6") != "" {
		t.Error("corrupt file must read empty")
	}
	if err := ks.Put(host, "c"); err != nil || ks.Get(host) != "c" {
		t.Errorf("put over corrupt: %v", err)
	}
}

func TestStoreCreatesDir(t *testing.T) {
	ks := Store{Path: filepath.Join(t.TempDir(), "sub", "keys.json")}
	if err := ks.Put(host, "a"); err != nil {
		t.Fatal(err)
	}
	if ks.Get(host) != "a" {
		t.Error("not stored")
	}
}
