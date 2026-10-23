package cdriver

import (
	"crypto/sha256"
	"time"

	"github.com/0disoft/fairyc/internal/packagecache"
)

func unitObjectLimits() packagecache.DiskLimits {
	return packagecache.DiskLimits{MaxEntries: 2, MaxBytes: 4 << 20,
		MaxRecordBytes: 2 << 20, TempMaxAge: 24 * time.Hour}
}

func openUnitObjectStore() (*packagecache.DiskStore, error) {
	return openNativeObjectStore("native-unit-v1", unitObjectLimits())
}

// Use a separate key domain even when the source bytes happen to match runtime.c.
func (identity runtimeObjectIdentity) unitKey() packagecache.Key {
	key := identity.key()
	return packagecache.Key(sha256.Sum256(append([]byte("fairyc.native-unit-object/1\x00"), key[:]...)))
}
