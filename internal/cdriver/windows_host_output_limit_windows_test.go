package cdriver

import (
	"bytes"
	"context"
	"strings"
	"testing"
	"time"
)

func TestWindowsTargetExplicitOutputLimit(t *testing.T) {
	for _, limit := range []int{0, -1, maxWindowsExecutionOutputBytes + 1} {
		if _, err := ExecuteWindowsMSVCTargetWithContextAndOutputLimit(context.Background(), WindowsHostArtifact{}, strings.NewReader(""), time.Minute, limit); err == nil || !strings.Contains(err.Error(), "output limit") {
			t.Fatal("invalid explicit limit", limit, err)
		}
	}
	toolchain, err := ResolveWindowsHostMSVC()
	if err != nil {
		t.Fatal(err)
	}
	source := `#include <stdio.h>
#include <fcntl.h>
#include <io.h>
int main(void){int i;if(_setmode(_fileno(stdout),_O_BINARY)==-1){return 2;}for(i=0;i<65537;++i){if(putchar(i&255)==EOF){return 3;}}return fflush(stdout)!=0?4:0;}`
	image, err := BuildWindowsHost(toolchain, LowProfile, source, "int fy_output_limit_runtime(void){return 0;}\n")
	if err != nil {
		t.Fatal(err)
	}
	for _, limit := range []int{65536, 65537, 65538} {
		out, err := ExecuteWindowsMSVCTargetWithContextAndOutputLimit(context.Background(), image, strings.NewReader(""), time.Minute, limit)
		if limit == 65536 {
			if err == nil || !strings.Contains(err.Error(), "exceeded 65536 output bytes") {
				t.Fatal("one-byte overflow admitted", err)
			}
			continue
		}
		want := make([]byte, 65537)
		for i := range want {
			want[i] = byte(i)
		}
		if err != nil || out.ExitCode != 0 || out.Stderr != "" || !bytes.Equal([]byte(out.Stdout), want) {
			t.Fatal("explicit binary capture changed bytes", len(out.Stdout), err)
		}
	}
	if _, err := ExecuteWindowsMSVCTargetWithContext(context.Background(), image, strings.NewReader(""), time.Minute); err == nil || !strings.Contains(err.Error(), "65536 output bytes") {
		t.Fatal("ordinary default changed", err)
	}
}
