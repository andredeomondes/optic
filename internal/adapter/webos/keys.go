package webos

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
)

// keysMu serializes read-modify-write of the keys file.
var keysMu sync.Mutex

// keyStore keeps the client key each LG TV issued, by host, in one JSON file.
type keyStore struct{ path string }

// Get returns the stored key for host, or "" when none (or the file is unreadable).
func (k keyStore) Get(host string) string {
	keysMu.Lock()
	defer keysMu.Unlock()
	return k.read()[host]
}

// Put stores the key for host.
func (k keyStore) Put(host, key string) error {
	keysMu.Lock()
	defer keysMu.Unlock()
	m := k.read()
	m[host] = key
	return k.write(m)
}

// Delete forgets the key for host.
func (k keyStore) Delete(host string) error {
	keysMu.Lock()
	defer keysMu.Unlock()
	m := k.read()
	if _, ok := m[host]; !ok {
		return nil
	}
	delete(m, host)
	return k.write(m)
}

// read returns the stored keys; a missing or corrupt file reads as empty.
func (k keyStore) read() map[string]string {
	m := map[string]string{}
	b, err := os.ReadFile(k.path)
	if err != nil {
		return m
	}
	if json.Unmarshal(b, &m) != nil || m == nil {
		return map[string]string{}
	}
	return m
}

func (k keyStore) write(m map[string]string) error {
	b, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(k.path), 0o700); err != nil {
		return err
	}
	return writeAtomic(k.path, b)
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
