// Package durablefile owns opened-root-contained publication of immutable derived files.
// It provides crash-safe observation boundaries, not universal physical-media retention.
package durablefile

import (
	"bytes"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// CreateTemporary creates one permission-restricted exclusive temporary inside directory.
// The caller owns file closure and best-effort path removal on every outcome.
func CreateTemporary(root *os.Root, directory string) (*os.File, string, error) {
	if root == nil {
		return nil, "", fmt.Errorf("durable file root must be initialized")
	}
	if !validRelativePath(directory) {
		return nil, "", fmt.Errorf("durable file directory must be canonical and nonempty")
	}
	for range 100 {
		var nonce [16]byte
		if _, err := rand.Read(nonce[:]); err != nil {
			return nil, "", err
		}
		name := filepath.Join(directory, ".tmp-"+hex.EncodeToString(nonce[:]))
		file, err := root.OpenFile(name, os.O_RDWR|os.O_CREATE|os.O_EXCL, 0o600)
		if errors.Is(err, os.ErrExist) {
			continue
		}
		return file, name, err
	}
	return nil, "", fmt.Errorf("exhausted durable file temporary names")
}

// PublishImmutable publishes fully written, synced, and closed expected bytes through a
// same-directory opened-root rename. An existing identical destination converges. Any different
// bytes under the same content-derived name are replaced and independently reverified by readers.
func PublishImmutable(root *os.Root, temporaryPath, finalPath string, expected []byte) error {
	if root == nil {
		return fmt.Errorf("durable file root must be initialized")
	}
	if temporaryPath == "" || finalPath == "" || len(expected) == 0 {
		return fmt.Errorf("durable file publication requires paths and expected bytes")
	}
	if !validRelativePath(temporaryPath) || !validRelativePath(finalPath) ||
		filepath.Dir(temporaryPath) != filepath.Dir(finalPath) {
		return fmt.Errorf("durable file publication paths must be canonical and share one directory")
	}
	directory := filepath.Dir(finalPath)
	if err := rename(root, temporaryPath, finalPath); err == nil {
		if syncErr := syncPublished(root, finalPath, directory); syncErr != nil {
			return fmt.Errorf("sync published durable file: %w", syncErr)
		}
		return nil
	}
	existing, readErr := root.ReadFile(finalPath)
	if readErr == nil && bytes.Equal(existing, expected) {
		if syncErr := syncPublished(root, finalPath, directory); syncErr != nil {
			return fmt.Errorf("sync converged durable file: %w", syncErr)
		}
		return nil
	}
	if removeErr := root.Remove(finalPath); removeErr == nil || errors.Is(removeErr, os.ErrNotExist) {
		if renameErr := rename(root, temporaryPath, finalPath); renameErr == nil {
			if syncErr := syncPublished(root, finalPath, directory); syncErr != nil {
				return fmt.Errorf("sync replaced durable file: %w", syncErr)
			}
			return nil
		}
		existing, readErr = root.ReadFile(finalPath)
		if readErr == nil && bytes.Equal(existing, expected) {
			return nil
		}
	}
	return fmt.Errorf("publish immutable durable file: destination conflict")
}

// SyncDirectory persists a newly created directory entry where the host exposes such a contract.
func SyncDirectory(root *os.Root, path string) error {
	if root == nil || !validRelativePath(path) {
		return fmt.Errorf("durable directory sync requires a root and canonical path")
	}
	return syncDirectory(root, path)
}

func validRelativePath(value string) bool {
	if value == "" || filepath.IsAbs(value) || filepath.VolumeName(value) != "" || filepath.Clean(value) != value {
		return false
	}
	return value != ".." && !strings.HasPrefix(value, ".."+string(filepath.Separator))
}
