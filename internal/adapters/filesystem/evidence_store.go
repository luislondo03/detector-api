package filesystem

import (
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
)

const maxEvidenceBytes = 5 << 20

type EvidenceStore struct {
	directory string
}

func NewEvidenceStore(directory string) (*EvidenceStore, error) {
	if err := os.MkdirAll(directory, 0o750); err != nil {
		return nil, err
	}
	return &EvidenceStore{directory: directory}, nil
}

func (store *EvidenceStore) Save(_ context.Context, evidenceID, mediaType string, content io.Reader) (string, error) {
	extension := map[string]string{"image/jpeg": ".jpg", "image/png": ".png"}[mediaType]
	if extension == "" {
		return "", errors.New("only JPEG and PNG evidence is allowed")
	}
	fileName := evidenceID + extension
	file, err := os.OpenFile(filepath.Join(store.directory, fileName), os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o640)
	if err != nil {
		return "", err
	}
	defer file.Close()
	if _, err := io.Copy(file, io.LimitReader(content, maxEvidenceBytes+1)); err != nil {
		return "", err
	}
	info, err := file.Stat()
	if err != nil {
		return "", err
	}
	if info.Size() > maxEvidenceBytes {
		_ = os.Remove(file.Name())
		return "", errors.New("evidence exceeds the 5 MiB limit")
	}
	return fileName, nil
}
