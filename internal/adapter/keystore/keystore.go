// Package keystore keeps per-host secrets (client keys, tokens) that TVs
// issue, in one JSON file.
package keystore

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
)

// mu serializes read-modify-write of the key files.
var mu sync.Mutex

// Store keeps the value each TV issued, by host, in the JSON file at Path.
type Store struct{ Path string }

// Get returns the stored value for host, or "" when none (or the file is unreadable).
func (s Store) Get(host string) string {
	mu.Lock()
	defer mu.Unlock()
	return s.read()[host]
}

// Put stores the value for host.
func (s Store) Put(host, v string) error {
	mu.Lock()
	defer mu.Unlock()
	m := s.read()
	m[host] = v
	return s.write(m)
}

// Delete forgets the value for host.
func (s Store) Delete(host string) error {
	mu.Lock()
	defer mu.Unlock()
	m := s.read()
	if _, ok := m[host]; !ok {
		return nil
	}
	delete(m, host)
	return s.write(m)
}

// read returns the stored values; a missing or corrupt file reads as empty.
func (s Store) read() map[string]string {
	m := map[string]string{}
	b, err := os.ReadFile(s.Path)
	if err != nil {
		return m
	}
	if json.Unmarshal(b, &m) != nil || m == nil {
		return map[string]string{}
	}
	return m
}

func (s Store) write(m map[string]string) error {
	b, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(s.Path), 0o700); err != nil {
		return err
	}
	return writeAtomic(s.Path, b)
}

// writeAtomic writes via a temp file and rename so a crash never leaves a
// half-written file behind.
func writeAtomic(path string, b []byte) error {
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, b, 0o600); err != nil {
		return err
	}
	if err := os.Rename(tmp, path); err != nil {
		_ = os.Remove(tmp)
		return err
	}
	return nil
}
