// Package projectmeta provides strict storage primitives for reserved project metadata.
package projectmeta

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

// ReadRegularFile reads a size-bounded regular metadata file.
func ReadRegularFile(filePath string, maximumBytes int64) ([]byte, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	info, err := file.Stat()
	if err != nil {
		return nil, err
	}
	if !info.Mode().IsRegular() {
		return nil, errors.New("metadata path is not a regular file")
	}
	if info.Size() > maximumBytes {
		return nil, fmt.Errorf("metadata exceeds %d bytes", maximumBytes)
	}
	content, err := io.ReadAll(io.LimitReader(file, maximumBytes+1))
	if err != nil {
		return nil, err
	}
	if int64(len(content)) > maximumBytes {
		return nil, fmt.Errorf("metadata exceeds %d bytes", maximumBytes)
	}
	return content, nil
}

// WriteImmutable atomically publishes content or accepts an identical existing file.
func WriteImmutable(target string, content []byte) error {
	if existing, err := os.ReadFile(target); err == nil {
		if bytes.Equal(existing, content) {
			return nil
		}
		return errors.New("content address already exists with different content")
	} else if !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("inspect existing metadata: %w", err)
	}
	if err := os.MkdirAll(filepath.Dir(target), 0o700); err != nil {
		return fmt.Errorf("create metadata directory: %w", err)
	}
	temporary, err := os.CreateTemp(filepath.Dir(target), ".metadata-*.tmp")
	if err != nil {
		return fmt.Errorf("create temporary metadata: %w", err)
	}
	temporaryPath := temporary.Name()
	defer os.Remove(temporaryPath)
	if err := temporary.Chmod(0o600); err != nil {
		temporary.Close()
		return fmt.Errorf("secure temporary metadata: %w", err)
	}
	if _, err := temporary.Write(content); err != nil {
		temporary.Close()
		return fmt.Errorf("write temporary metadata: %w", err)
	}
	if err := temporary.Sync(); err != nil {
		temporary.Close()
		return fmt.Errorf("sync temporary metadata: %w", err)
	}
	if err := temporary.Close(); err != nil {
		return fmt.Errorf("close temporary metadata: %w", err)
	}
	if err := os.Rename(temporaryPath, target); err != nil {
		return fmt.Errorf("publish immutable metadata: %w", err)
	}
	return nil
}

// DecodeStrict decodes exactly one JSON value and rejects unknown struct fields.
func DecodeStrict(content []byte, target any) error {
	decoder := json.NewDecoder(bytes.NewReader(content))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		return err
	}
	if err := decoder.Decode(new(any)); !errors.Is(err, io.EOF) {
		if err == nil {
			return errors.New("multiple JSON documents are not allowed")
		}
		return err
	}
	return nil
}
