package save

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
)

// WriteVerified publishes a verified temporary file in the destination directory.
// Replacement is explicit, and retains the original when backup is requested.
func WriteVerified(path string, b []byte, replace, backup bool, verify func([]byte) error) (string, error) {
	path, err := filepath.Abs(path)
	if err != nil {
		return "", err
	}
	f, err := os.CreateTemp(filepath.Dir(path), ".feth-save-*")
	if err != nil {
		return "", err
	}
	temp := f.Name()
	defer os.Remove(temp)
	if _, err = f.Write(b); err == nil {
		err = f.Sync()
	}
	closeErr := f.Close()
	if err == nil {
		err = closeErr
	}
	if err != nil {
		return "", err
	}
	stored, err := os.ReadFile(temp)
	if err != nil {
		return "", err
	}
	if !bytes.Equal(stored, b) {
		return "", fmt.Errorf("written save failed verification")
	}
	if err = verify(stored); err != nil {
		return "", err
	}
	backupPath := ""
	if replace {
		if backup {
			original, err := os.ReadFile(path)
			if err != nil {
				return "", err
			}
			bf, err := os.CreateTemp(filepath.Dir(path), filepath.Base(path)+".backup-*")
			if err != nil {
				return "", err
			}
			backupPath = bf.Name()
			_, err = bf.Write(original)
			if err == nil {
				err = bf.Sync()
			}
			closeErr = bf.Close()
			if err == nil {
				err = closeErr
			}
			if err != nil {
				return backupPath, err
			}
		}
		err = os.Rename(temp, path)
	} else {
		err = os.Link(temp, path)
	}
	return backupPath, err
}
