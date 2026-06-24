package graph

import (
	"context"
	"os"
	"path/filepath"
	"sync"

	"github.com/AzureAD/microsoft-authentication-library-for-go/apps/cache"
)

// fileCache persists the MSAL token cache to a 0600 file on disk, implementing
// cache.ExportReplace.
type fileCache struct {
	path string
	mu   sync.Mutex
}

func newFileCache(path string) *fileCache { return &fileCache{path: path} }

// Replace loads the on-disk cache into MSAL before a token operation.
func (f *fileCache) Replace(_ context.Context, u cache.Unmarshaler, _ cache.ReplaceHints) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	data, err := os.ReadFile(f.path)
	if err != nil {
		// A missing cache is not an error: MSAL starts empty.
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	return u.Unmarshal(data)
}

// Export writes MSAL's updated cache back to disk after a token operation.
func (f *fileCache) Export(_ context.Context, m cache.Marshaler, _ cache.ExportHints) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	data, err := m.Marshal()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(f.path), 0o700); err != nil {
		return err
	}
	tmp, err := os.CreateTemp(filepath.Dir(f.path), ".tok-*")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	defer func() { _ = os.Remove(tmpName) }()
	if _, err := tmp.Write(data); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Chmod(0o600); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	return os.Rename(tmpName, f.path)
}
