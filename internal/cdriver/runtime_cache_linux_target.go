package cdriver

import (
	"bytes"
	"crypto/sha256"
	"debug/elf"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

// BuildCached reuses only the runtime object; validation/instrumentation APIs
// remain uncached. The generated unit and final executable are always rebuilt.
func BuildCached(toolchain Toolchain, profile Profile, unitSource, runtimeSource string) (Artifact, error) {
	return buildLinuxRuntimeCached(toolchain, profile, unitSource, runtimeSource, openRuntimeObjectStore)
}

func buildLinuxRuntimeCached(toolchain Toolchain, profile Profile, unitSource, runtimeSource string, open runtimeStoreOpener) (Artifact, error) {
	uncached := func() (Artifact, error) {
		artifact, err := Build(toolchain, profile, unitSource, runtimeSource)
		artifact.RuntimeCache = RuntimeCacheBypass
		return artifact, err
	}
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" ||
		(profile != LowProfile && profile != OptimizedProfile) || toolchain.Target != "x86_64-linux-gnu" ||
		(toolchain.Name != "gcc" && toolchain.Name != "clang") || unitSource == "" || runtimeSource == "" {
		return uncached()
	}
	compiler, err := readArtifact(toolchain.Path)
	if err != nil {
		return uncached()
	}
	directory, err := os.MkdirTemp("", "fairyc-runtime-object-")
	if err != nil {
		return uncached()
	}
	defer os.RemoveAll(directory)
	sourcePath := filepath.Join(directory, "runtime.c")
	if err := os.WriteFile(sourcePath, []byte(runtimeSource), 0o600); err != nil {
		return uncached()
	}
	flags := compilerArguments(profile, nil)
	environment := append(os.Environ(), "LC_ALL=C", "LANG=C")
	// Retain line/system-header markers so strict C warnings still distinguish
	// libc headers from project-owned input when compiling the snapshot.
	preprocessed, err := preprocessRuntime(toolchain.Path, environment, directory, append(append([]string{}, flags...), "-E", "runtime.c"))
	if err != nil {
		return uncached()
	}
	if err := os.WriteFile(sourcePath, []byte(preprocessed), 0o600); err != nil {
		return uncached()
	}
	key := (runtimeObjectIdentity{CompilerPath: toolchain.Path, CompilerVersion: toolchain.Version,
		CompilerDigest: sha256.Sum256(compiler), Target: string(toolchain.Target), Libc: toolchain.Libc,
		Mode: C11ConsumerMode, Profile: profile, Flags: flags, Environment: environment,
		Source: sha256.Sum256([]byte(runtimeSource)), Preprocessed: sha256.Sum256([]byte(preprocessed))}).key()
	store, _ := open()
	if store != nil {
		defer store.Close()
	}
	unitPath, objectPath, executablePath := filepath.Join(directory, "unit.c"), filepath.Join(directory, "runtime.o"), filepath.Join(directory, "program")
	if err := os.WriteFile(unitPath, []byte(unitSource), 0o600); err != nil {
		return Artifact{}, err
	}
	var contents []byte
	status, err := useRuntimeObject(store, key, func() ([]byte, error) {
		args := append(append([]string{}, flags...), "-c", sourcePath, "-o", objectPath)
		if _, err := runBounded(toolchain.Path, args...); err != nil {
			return nil, fmt.Errorf("compile runtime: %s", strings.ReplaceAll(err.Error(), directory, "<temporary>"))
		}
		return readArtifact(objectPath)
	}, validateLinuxRuntimeObject, func(object []byte) error {
		if err := os.WriteFile(objectPath, object, 0o600); err != nil {
			return err
		}
		args := append(append([]string{}, flags...), unitPath, objectPath, "-o", executablePath)
		if _, err := runBounded(toolchain.Path, args...); err != nil {
			return fmt.Errorf("link runtime: %s", strings.ReplaceAll(err.Error(), directory, "<temporary>"))
		}
		var err error
		contents, err = readArtifact(executablePath)
		return err
	})
	return Artifact{Bytes: contents, Toolchain: toolchain, Profile: profile, ConsumerMode: C11ConsumerMode, RuntimeCache: status}, err
}

func validateLinuxRuntimeObject(object []byte) error {
	file, err := elf.NewFile(bytes.NewReader(object))
	if err != nil {
		return err
	}
	defer file.Close()
	if file.Type != elf.ET_REL || file.Class != elf.ELFCLASS64 || file.Machine != elf.EM_X86_64 || file.Data != elf.ELFDATA2LSB {
		return fmt.Errorf("runtime object must be a little-endian AMD64 ELF relocatable")
	}
	return nil
}
