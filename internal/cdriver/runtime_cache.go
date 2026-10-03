package cdriver

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"slices"
	"strings"
	"time"

	"github.com/0disoft/fairyc/internal/moduleinterface"
	"github.com/0disoft/fairyc/internal/packagecache"
)

const maxRuntimePreprocessedBytes = 16 * 1024 * 1024

type runtimeStoreOpener func() (*packagecache.DiskStore, error)

// RuntimeObjectCacheStatus describes work avoided, not elapsed-time savings.
type RuntimeObjectCacheStatus string

const (
	RuntimeCacheBypass    RuntimeObjectCacheStatus = "bypass"
	RuntimeCacheMiss      RuntimeObjectCacheStatus = "miss"
	RuntimeCacheHit       RuntimeObjectCacheStatus = "hit"
	RuntimeCacheRecovered RuntimeObjectCacheStatus = "recovered"
)

// A runtime key describes the compiler's exact input, never the generated unit.
// The preprocessed snapshot is also the input to object compilation, closing the
// header-change race between preprocessing and compiling the original source.
type runtimeObjectIdentity struct {
	Version         string
	Host            string
	CompilerPath    string
	CompilerVersion string
	CompilerDigest  [32]byte
	Target          string
	Libc            string
	Mode            ConsumerMode
	Profile         Profile
	Flags           []string
	Environment     []string
	Source          [32]byte
	Preprocessed    [32]byte
}

func (identity runtimeObjectIdentity) key() packagecache.Key {
	identity.Version = "fairyc.native-runtime-object/1"
	identity.Host = runtime.GOOS + "/" + runtime.GOARCH
	identity.Environment = canonicalRuntimeEnvironment(identity.Environment)
	encoded, _ := json.Marshal(identity)
	return packagecache.Key(sha256.Sum256(encoded))
}

// exec.Cmd uses the last value of a duplicated environment name. Hash that
// effective environment rather than sorting conflicting duplicates together.
func canonicalRuntimeEnvironment(environment []string) []string {
	values := make(map[string]string, len(environment))
	for _, entry := range environment {
		name, value, found := strings.Cut(entry, "=")
		if !found || name == "" {
			continue
		}
		if runtime.GOOS == "windows" {
			name = strings.ToUpper(name)
		}
		values[name] = value
	}
	result := make([]string, 0, len(values))
	for name, value := range values {
		result = append(result, name+"="+value)
	}
	slices.Sort(result)
	return result
}

func openRuntimeObjectStore() (*packagecache.DiskStore, error) {
	base, err := os.UserCacheDir()
	if err != nil {
		return nil, err
	}
	if err := os.MkdirAll(base, 0o700); err != nil {
		return nil, err
	}
	boundary, err := os.OpenRoot(base)
	if err != nil {
		return nil, err
	}
	defer boundary.Close()
	return packagecache.NewDiskStoreInRoot(boundary, filepath.Join("fairyc", "native-runtime-v1"), runtimeObjectLimits())
}

func runtimeObjectLimits() packagecache.DiskLimits {
	return packagecache.DiskLimits{MaxEntries: 64, MaxBytes: 64 * 1024 * 1024,
		MaxRecordBytes: 16 * 1024 * 1024, TempMaxAge: 24 * time.Hour}
}

// Cache errors are deliberately disposable. Only compilation/link errors fail
// a build. A rejected hit gets exactly one fresh compile and link attempt.
func useRuntimeObject(store *packagecache.DiskStore, key packagecache.Key,
	compile func() ([]byte, error), validate func([]byte) error, link func([]byte) error,
) (RuntimeObjectCacheStatus, error) {
	status := RuntimeCacheBypass
	if store != nil {
		status = RuntimeCacheMiss
		if record, found := store.Get(key); found {
			if record.PublicInterface == moduleinterface.Hash(key) && len(record.Artifacts) == 1 &&
				record.Artifacts[0].Kind == packagecache.ArtifactObject && validate(record.Artifacts[0].Data) == nil {
				if err := link(record.Artifacts[0].Data); err == nil {
					return RuntimeCacheHit, nil
				}
				status = RuntimeCacheRecovered
			}
			store.DeleteIfMatch(record)
		}
	}
	object, err := compile()
	if err != nil {
		return status, err
	}
	if err := validate(object); err != nil {
		return status, err
	}
	if err := link(object); err != nil {
		return status, err
	}
	if store != nil {
		_ = store.Put(packagecache.Record{Key: key, PublicInterface: moduleinterface.Hash(key),
			Artifacts: []packagecache.Artifact{{Kind: packagecache.ArtifactObject, Data: object}}})
	}
	return status, nil
}

// Preprocessor stdout is kept separate from diagnostics: warnings must never
// become C input. An oversized/failed snapshot causes the uncached build path.
func preprocessRuntime(path string, environment []string, directory string, arguments []string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), commandTimeout)
	defer cancel()
	command := exec.CommandContext(ctx, path, arguments...)
	command.Dir = directory
	command.Env = environment
	output := newBoundedProcessOutput(maxRuntimePreprocessedBytes)
	command.Stdout, command.Stderr = output.stdoutWriter(), output.stderrWriter()
	err := command.Run()
	stdout, stderr, truncated := output.values()
	if errors.Is(ctx.Err(), context.DeadlineExceeded) || truncated || err != nil || stdout == "" {
		return "", fmt.Errorf("runtime preprocessing failed: %v: %s", err, stderr)
	}
	return stdout, nil
}
