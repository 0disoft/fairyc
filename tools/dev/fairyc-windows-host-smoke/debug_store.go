package main

import (
	"bytes"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/0disoft/fairyc/internal/pecoff"
)

const debugStoreSchema = "fairyc.windows-debug-bundle/v1"

// Private, local developer evidence. Not a public symbol-server layout.
type windowsDebugManifest struct {
	Schema         string            `json:"schema"`
	SourceRevision string            `json:"sourceRevision"`
	Profile        string            `json:"profile"`
	Toolchain      string            `json:"toolchain"`
	GUID           string            `json:"guidRawBytes"`
	Age            uint32            `json:"age"`
	Files          map[string]string `json:"files"`
}

var debugStoreFiles = []string{"program.exe", "program.pdb", "main.fy", "unit.c", "runtime.c", "unit.obj", "runtime.obj", "program.map"}

func debugDigest(data []byte) string { return fmt.Sprintf("%x", sha256.Sum256(data)) }

func validDebugDigest(value string, size int) bool {
	data, err := hex.DecodeString(value)
	return err == nil && len(data) == size && strings.ToLower(value) == value && value != strings.Repeat("0", size*2)
}

func validateDebugBundle(manifest windowsDebugManifest, files map[string][]byte, key string) error {
	if manifest.Schema != debugStoreSchema || !validDebugDigest(manifest.SourceRevision, 20) ||
		(manifest.Profile != "low" && manifest.Profile != "optimized") || manifest.Toolchain == "" ||
		!validDebugDigest(key, 32) || len(files) != len(debugStoreFiles) || len(manifest.Files) != len(debugStoreFiles) {
		return fmt.Errorf("invalid debug-bundle metadata or image key")
	}
	for _, name := range debugStoreFiles {
		data := files[name]
		if len(data) == 0 || len(data) > 64*1024*1024 || !validDebugDigest(manifest.Files[name], 32) || debugDigest(data) != manifest.Files[name] {
			return fmt.Errorf("debug-bundle file %s missing, oversized or hash mismatch", name)
		}
	}
	if manifest.Files["program.exe"] != key {
		return fmt.Errorf("debug-bundle image key mismatch")
	}
	identity, err := pecoff.InspectProgramDatabaseAssociation(files["program.exe"], files["program.pdb"])
	if err != nil {
		return fmt.Errorf("debug-bundle PE/PDB association: %w", err)
	}
	if manifest.GUID != fmt.Sprintf("%x", identity.GUID) || manifest.Age != identity.Age {
		return fmt.Errorf("debug-bundle manifest GUID/age mismatch")
	}
	return nil
}

func retainWindowsDebugBundle(store string, manifest windowsDebugManifest, files map[string][]byte) (string, error) {
	identity, err := pecoff.InspectProgramDatabaseAssociation(files["program.exe"], files["program.pdb"])
	if err != nil {
		return "", err
	}
	manifest.Schema, manifest.GUID, manifest.Age = debugStoreSchema, fmt.Sprintf("%x", identity.GUID), identity.Age
	manifest.Files = make(map[string]string, len(files))
	for name, data := range files {
		manifest.Files[name] = debugDigest(data)
	}
	key := debugDigest(files["program.exe"])
	if err := validateDebugBundle(manifest, files, key); err != nil {
		return "", err
	}
	data, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return "", err
	}
	data = append(data, '\n')
	if err := os.MkdirAll(store, 0700); err != nil {
		return "", err
	}
	root, err := os.OpenRoot(store)
	if err != nil {
		return "", err
	}
	defer root.Close()
	if _, err := root.Lstat(key); err == nil {
		return key, compareRetainedDebugBundle(store, key, manifest)
	} else if !os.IsNotExist(err) {
		return "", err
	}
	var nonce [16]byte
	if _, err := rand.Read(nonce[:]); err != nil {
		return "", err
	}
	stage := fmt.Sprintf(".stage-%x", nonce)
	if err := root.Mkdir(stage, 0700); err != nil {
		return "", err
	}
	defer root.RemoveAll(stage)
	for _, name := range debugStoreFiles {
		if err := root.WriteFile(filepath.Join(stage, name), files[name], 0600); err != nil {
			return "", err
		}
	}
	if err := root.WriteFile(filepath.Join(stage, "manifest.json"), data, 0600); err != nil {
		return "", err
	}
	if err := root.Rename(stage, key); err != nil {
		// A concurrent identical publication is harmless; never overwrite an entry.
		if compareErr := compareRetainedDebugBundle(store, key, manifest); compareErr != nil {
			return "", fmt.Errorf("publish debug bundle: %w; existing entry: %v", err, compareErr)
		}
	}
	return key, nil
}

func compareRetainedDebugBundle(store, key string, expected windowsDebugManifest) error {
	actual, _, err := loadWindowsDebugBundle(store, key)
	if err != nil {
		return err
	}
	a, _ := json.Marshal(actual)
	b, _ := json.Marshal(expected)
	if !bytes.Equal(a, b) {
		return fmt.Errorf("refuse to replace existing debug-bundle metadata")
	}
	return nil
}

func readDebugStoreFile(root *os.Root, name string, limit int64) ([]byte, error) {
	file, err := root.Open(name)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	info, err := file.Stat()
	if err != nil {
		return nil, err
	}
	if !info.Mode().IsRegular() || info.Size() <= 0 || info.Size() > limit {
		return nil, fmt.Errorf("invalid retained debug file %s", name)
	}
	data, err := io.ReadAll(io.LimitReader(file, limit+1))
	if err != nil || int64(len(data)) > limit {
		return nil, fmt.Errorf("read retained debug file %s: %v", name, err)
	}
	return data, nil
}

func loadWindowsDebugBundle(store, key string) (windowsDebugManifest, map[string][]byte, error) {
	var manifest windowsDebugManifest
	if !validDebugDigest(key, 32) {
		return manifest, nil, fmt.Errorf("invalid debug image SHA-256")
	}
	root, err := os.OpenRoot(store)
	if err != nil {
		return manifest, nil, err
	}
	defer root.Close()
	// Retain the store root as the traversal boundary, including for the entry directory.
	data, err := readDebugStoreFile(root, filepath.Join(key, "manifest.json"), 64*1024)
	if err != nil {
		return manifest, nil, err
	}
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&manifest); err != nil {
		return manifest, nil, err
	}
	if err := decoder.Decode(new(any)); err != io.EOF {
		return manifest, nil, fmt.Errorf("trailing debug manifest data")
	}
	files := make(map[string][]byte, len(debugStoreFiles))
	for _, name := range debugStoreFiles {
		contents, err := readDebugStoreFile(root, filepath.Join(key, name), 64*1024*1024)
		if err != nil {
			return manifest, nil, err
		}
		files[name] = contents
	}
	if err := validateDebugBundle(manifest, files, key); err != nil {
		return manifest, nil, err
	}
	return manifest, files, nil
}
