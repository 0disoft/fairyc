package cdriver

import (
	"errors"
	"runtime"
	"sync"
	"sync/atomic"
	"testing"
)

func TestMemoizedWindowsHostResolverConcurrentIsolation(t *testing.T) {
	var calls atomic.Int32
	resolve := memoizeWindowsHostResolver(func() (WindowsHostToolchain, error) {
		calls.Add(1)
		runtime.Gosched()
		return WindowsHostToolchain{Name: "msvc", Path: "cl.exe", Version: "test", Environment: []string{"PATH=original", "LIB=original"}}, nil
	})
	var workers sync.WaitGroup
	failures := make(chan string, 32)
	for i := 0; i < 32; i++ {
		workers.Add(1)
		go func() {
			defer workers.Done()
			toolchain, err := resolve()
			if err != nil || len(toolchain.Environment) != 2 {
				failures <- "resolution failed"
				return
			}
			if toolchain.Environment[0] != "PATH=original" {
				failures <- "another caller mutated the cached environment"
				return
			}
			toolchain.Environment[0] = "PATH=caller"
		}()
	}
	workers.Wait()
	close(failures)
	for failure := range failures {
		t.Error(failure)
	}
	fresh, err := resolve()
	if err != nil || fresh.Environment[0] != "PATH=original" {
		t.Fatal("cached environment was exposed")
	}
	if calls.Load() != 1 {
		t.Fatalf("discovery calls=%d, want 1", calls.Load())
	}
}

func TestMemoizedWindowsHostResolverRetainsFirstFailure(t *testing.T) {
	sentinel := errors.New("toolchain unavailable")
	calls := 0
	resolve := memoizeWindowsHostResolver(func() (WindowsHostToolchain, error) {
		calls++
		if calls == 1 {
			return WindowsHostToolchain{}, sentinel
		}
		return WindowsHostToolchain{Name: "unexpected recovery"}, nil
	})
	for i := 0; i < 3; i++ {
		_, err := resolve()
		if !errors.Is(err, sentinel) {
			t.Fatal("first failure was not retained")
		}
	}
	if calls != 1 {
		t.Fatalf("discovery calls=%d, want 1", calls)
	}
}

func TestResolvedWindowsHostEnvironmentIsDetached(t *testing.T) {
	if runtime.GOOS != "windows" || runtime.GOARCH != "amd64" {
		t.Skip("requires Windows AMD64 MSVC")
	}
	first, err := ResolveWindowsHostMSVC()
	if err != nil {
		t.Fatal(err)
	}
	second, err := ResolveWindowsHostMSVC()
	if err != nil {
		t.Fatal(err)
	}
	if len(first.Environment) == 0 || len(first.Environment) != len(second.Environment) {
		t.Fatal("invalid cached environment")
	}
	if &first.Environment[0] == &second.Environment[0] {
		t.Fatal("resolution shares a mutable environment slice")
	}
	expected := second.Environment[0]
	first.Environment[0] = "FAIRYC_CALLER_ONLY=1"
	third, err := ResolveWindowsHostMSVC()
	if err != nil || third.Environment[0] != expected {
		t.Fatal("caller modified process-cached discovery")
	}
	if first.Path != second.Path || second.Path != third.Path || first.Version != third.Version {
		t.Fatal("process toolchain snapshot changed")
	}
}
