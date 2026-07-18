package cbackendnative

import (
	"runtime"
	"strings"
	"testing"

	"github.com/0disoft/fairyc/internal/cbackendfixture"
)

func TestExpectedExitCode(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		fixture cbackendfixture.EmittedCase
		want    int
		fails   bool
	}{
		{name: "return", fixture: cbackendfixture.EmittedCase{ID: "return", Outcome: "return", Value: 42}, want: 42},
		{name: "overflow", fixture: cbackendfixture.EmittedCase{ID: "trap", Outcome: "trap", TrapCategory: "integer-overflow"}, want: 194},
		{name: "unknown trap", fixture: cbackendfixture.EmittedCase{ID: "trap", Outcome: "trap", TrapCategory: "unknown"}, fails: true},
		{name: "negative return", fixture: cbackendfixture.EmittedCase{ID: "return", Outcome: "return", Value: -1}, fails: true},
		{name: "reserved return", fixture: cbackendfixture.EmittedCase{ID: "return", Outcome: "return", Value: trapExitBase}, fails: true},
	}
	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			got, err := expectedExitCode(test.fixture)
			if test.fails {
				if err == nil {
					t.Fatalf("expectedExitCode() = %d, nil", got)
				}
				return
			}
			if err != nil || got != test.want {
				t.Fatalf("expectedExitCode() = %d, %v; want %d, nil", got, err, test.want)
			}
		})
	}
}

func TestBoundedBufferRetainsPrefix(t *testing.T) {
	t.Parallel()
	buffer := boundedBuffer{limit: 4}
	if written, err := buffer.Write([]byte("abcdef")); err != nil || written != 6 {
		t.Fatalf("Write() = %d, %v", written, err)
	}
	if buffer.String() != "abcd" || !buffer.truncated {
		t.Fatalf("buffer = %q, truncated=%t", buffer.String(), buffer.truncated)
	}
}

func TestRunRejectsNonLinuxHost(t *testing.T) {
	if runtime.GOOS == "linux" && runtime.GOARCH == "amd64" {
		t.Skip("native host is eligible; actual compiler execution belongs to the configured native command")
	}
	_, err := Run("unused.json")
	if err == nil || !strings.Contains(err.Error(), "requires linux/amd64") {
		t.Fatalf("Run() error = %v", err)
	}
}
