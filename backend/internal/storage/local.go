package storage

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
)

type Local struct {
	BaseDir string
}

func NewLocal(baseDir string) *Local {
	return &Local{BaseDir: baseDir}
}

func (s *Local) Ensure() error {
	return os.MkdirAll(s.BaseDir, 0o755)
}

func (s *Local) Save(filename string, src io.Reader) (string, string, error) {
	if err := s.Ensure(); err != nil {
		return "", "", err
	}

	storedName := filename
	fullPath := filepath.Join(s.BaseDir, storedName)
	file, err := os.Create(fullPath)
	if err != nil {
		return "", "", err
	}
	defer file.Close()

	if _, err := io.Copy(file, src); err != nil {
		return "", "", err
	}

	return storedName, fullPath, nil
}

func (s *Local) Open(filename string) (*os.File, error) {
	return os.Open(filepath.Join(s.BaseDir, filename))
}

func (s *Local) Path(filename string) string {
	return filepath.Join(s.BaseDir, filename)
}

func (s *Local) Remove(filename string) error {
	err := os.Remove(filepath.Join(s.BaseDir, filename))
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("remove file: %w", err)
	}
	return nil
}

