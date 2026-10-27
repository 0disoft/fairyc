package cdriver

import (
	"crypto/sha256"
	"fmt"
	"strings"
	"testing"
)

func TestSourceDigestPreservesExactBytes(t *testing.T) {
	for _, size := range []int{0, 1, 31, 32, 33, 4095, 4096, 4097, 65536, maxRuntimePreprocessedBytes} {
		t.Run(fmt.Sprint(size), func(t *testing.T) {
			data := make([]byte, size)
			for i := range data {
				data[i] = byte(i)
			}
			if got, want := sourceDigest(string(data)), sha256.Sum256(data); got != want {
				t.Fatalf("digest = %x, want %x", got, want)
			}
		})
	}
}

func TestSourceDigestsPreserveNativeCacheKeys(t *testing.T) {
	source := strings.Repeat("/* Café 🧚 */\r\n", 400)
	snapshot := "#line 1 \"runtime.c\"\n" + source
	previous := runtimeObjectIdentity{
		CompilerPath: "/compiler", CompilerVersion: "1", Target: "target",
		Mode: C11ConsumerMode, Profile: OptimizedProfile, Flags: []string{"-O2"},
		Source: sha256.Sum256([]byte(source)), Preprocessed: sha256.Sum256([]byte(snapshot)),
	}
	current := previous
	current.Source, current.Preprocessed = sourceDigest(source), sourceDigest(snapshot)
	if current.key() != previous.key() || current.unitKey() != previous.unitKey() {
		t.Fatal("streaming source hashing invalidated an existing native cache key")
	}
}

func BenchmarkNativeCacheSourceDigests(b *testing.B) {
	for _, size := range []int{16, 4096, 1 << 20, maxRuntimePreprocessedBytes} {
		b.Run(fmt.Sprint(size), func(b *testing.B) {
			source := strings.Repeat("x", size)
			snapshot := strings.Repeat("y", size)
			wantSource, wantSnapshot := sha256.Sum256([]byte(source)), sha256.Sum256([]byte(snapshot))
			b.SetBytes(int64(size) * 2)
			b.ReportAllocs()
			for b.Loop() {
				if sourceDigest(source) != wantSource || sourceDigest(snapshot) != wantSnapshot {
					b.Fatal("cache source digest changed")
				}
			}
		})
	}
}
