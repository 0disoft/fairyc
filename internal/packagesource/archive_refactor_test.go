package packagesource

import (
	"bytes"
	"context"
	"os"
	"strings"
	"testing"
)

func TestArchiveValidationOrder(t *testing.T) {
	canceled, cancel := context.WithCancel(context.Background())
	cancel()
	badPin := testPin()
	badPin.PackageUID = "example.other"
	for _, test := range []struct {
		name  string
		ctx   context.Context
		data  []byte
		pin   Pin
		files int
		want  string
	}{
		{"format before cancellation", canceled, []byte("not ZIP"), testPin(), 0, "invalid source ZIP: zip: not a valid zip file"},
		{"cancellation before paths", canceled, archiveBytes(t, zipEntry{"../escape", "x", 0644}), testPin(), 2, "context canceled"},
		{"path before mode", context.Background(), archiveBytes(t, zipEntry{"../escape", "x", os.ModeSymlink | 0777}), testPin(), 2, "nonportable ZIP path"},
		{"mode before duplicate", context.Background(), archiveBytes(t, zipEntry{"src/math.fy", "x", os.ModeSymlink | 0777}), testPin(), 2, "ZIP special entry rejected"},
		{"duplicate before budget", context.Background(), archiveBytes(t, zipEntry{"src/math.fy", "x", 0644}), testPin(), 2, "duplicate ZIP entry"},
		{"conflict before budget", context.Background(), archiveBytes(t, zipEntry{"src", "x", 0644}), testPin(), 2, "ZIP case or file/directory conflict"},
		{"budget before identity", context.Background(), archiveBytes(t), badPin, 1, "ZIP file limit exceeded"},
	} {
		t.Run(test.name, func(t *testing.T) {
			_, err := unpackWithin(test.ctx, test.data, test.pin, "fairyc-language/20", MaxTotalBytes, test.files)
			if err == nil || err.Error() != test.want {
				t.Fatalf("error=%v, want %q", err, test.want)
			}
		})
	}
}

func TestArchiveInputIsolation(t *testing.T) {
	for _, invalid := range []bool{false, true} {
		data := archiveBytes(t)
		if invalid {
			data = archiveBytes(t, zipEntry{"../escape", "x", 0644})
		}
		original := bytes.Clone(data)
		pin := testPin()
		result, err := unpack(context.Background(), data, pin, "fairyc-language/20")
		if (err != nil) != invalid {
			t.Fatalf("invalid=%v error=%v", invalid, err)
		}
		if !invalid {
			result.files["src/math.fy"][0] = 'X'
		}
		if !bytes.Equal(data, original) || pin != testPin() {
			t.Fatal("archive input or pin changed")
		}
	}
}

func TestArchiveRemainingBudgetBoundary(t *testing.T) {
	data := archiveBytes(t)
	baseline, err := unpack(context.Background(), data, testPin(), "fairyc-language/20")
	if err != nil {
		t.Fatal(err)
	}
	for _, delta := range []int{-1, 0, 1} {
		_, err := unpackWithin(context.Background(), data, testPin(), "fairyc-language/20", baseline.bytes+delta, MaxFiles)
		if delta < 0 {
			if err == nil || err.Error() != "ZIP file limit exceeded" {
				t.Fatalf("byte delta=%d error=%v", delta, err)
			}
		} else if err != nil {
			t.Fatalf("byte delta=%d error=%v", delta, err)
		}
		_, err = unpackWithin(context.Background(), data, testPin(), "fairyc-language/20", MaxTotalBytes, len(baseline.files)+delta)
		if (err != nil) != (delta < 0) {
			t.Fatalf("file delta=%d error=%v", delta, err)
		}
	}
}

func TestArchivePathPolicyBoundary(t *testing.T) {
	prefix := strings.Repeat(strings.Repeat("a", 63)+"/", 15)
	for _, size := range []int{1023, 1024, 1025} {
		name := prefix + strings.Repeat("b", size-len(prefix))
		if got := portableEntry(name); got != (size <= 1024) {
			t.Fatalf("path bytes=%d accepted=%v", size, got)
		}
	}
	for _, depth := range []int{31, 32, 33} {
		name := strings.Repeat("a/", depth-1) + "a"
		if got := portableEntry(name); got != (depth <= 32) {
			t.Fatalf("path depth=%d accepted=%v", depth, got)
		}
	}
}
