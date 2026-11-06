package cdriver

import (
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"testing"
	"time"

	"golang.org/x/sys/windows"
)

func TestWindowsTargetContextCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := ExecuteWindowsMSVCTargetWithContext(ctx, WindowsHostArtifact{}, strings.NewReader(""), time.Minute); !errors.Is(err, context.Canceled) {
		t.Fatal("pre-cancel", err)
	}
	for _, timeout := range []time.Duration{0, -time.Second, time.Minute + 1} {
		if _, err := ExecuteWindowsMSVCTargetWithContext(context.Background(), WindowsHostArtifact{}, strings.NewReader(""), timeout); err == nil {
			t.Fatal("invalid timeout admitted", timeout)
		}
	}
	toolchain, err := ResolveWindowsHostMSVC()
	if err != nil {
		t.Fatal(err)
	}
	id := sha256.Sum256([]byte(t.TempDir()))
	name := fmt.Sprintf("Local\\fairyc-context-%x", id[:12])
	wide, err := windows.UTF16PtrFromString(name)
	if err != nil {
		t.Fatal(err)
	}
	event, err := windows.CreateEvent(nil, 1, 0, wide)
	if err != nil {
		if event != 0 {
			windows.CloseHandle(event)
		}
		t.Fatal(err)
	}
	defer windows.CloseHandle(event)
	source := fmt.Sprintf(`#include <windows.h>
int main(void){HANDLE ready=OpenEventA(EVENT_MODIFY_STATE,FALSE,%s);if(!ready){return 2;}if(!SetEvent(ready)){CloseHandle(ready);return 3;}CloseHandle(ready);Sleep(30000);return 0;}`, strconv.Quote(name))
	image, err := BuildWindowsHost(toolchain, LowProfile, source, "int fy_context_runtime_unit(void){return 0;}\n")
	if err != nil {
		t.Fatal(err)
	}
	for _, deadline := range []bool{false, true} {
		t.Run(fmt.Sprintf("deadline-%t", deadline), func(t *testing.T) {
			if err := windows.ResetEvent(event); err != nil {
				t.Fatal(err)
			}
			ctx, cancel := context.WithCancel(context.Background())
			want := context.Canceled
			if deadline {
				cancel()
				ctx, cancel = context.WithTimeout(context.Background(), 2*time.Second)
				want = context.DeadlineExceeded
			}
			defer cancel()
			finished := make(chan error, 1)
			go func() {
				_, err := ExecuteWindowsMSVCTargetWithContext(ctx, image, strings.NewReader(""), time.Minute)
				finished <- err
			}()
			if status, err := windows.WaitForSingleObject(event, 5000); err != nil || status != windows.WAIT_OBJECT_0 {
				cancel()
				<-finished
				t.Fatalf("native start: %d %v", status, err)
			}
			if !deadline {
				cancel()
			}
			select {
			case err := <-finished:
				if !errors.Is(err, want) {
					t.Fatal("lost in-flight context error", err)
				}
			case <-time.After(4 * time.Second):
				t.Fatal("native process or owned pipes survived cancellation")
			}
		})
	}
}
