package packagesource

import (
	"bytes"
	"os"
	"path"
	"strings"
	"testing"
)

func TestArchivePublicationVerifiesExistingDigestWithoutReplacing(t *testing.T) {
	root, err := os.OpenRoot(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	defer root.Close()
	data := bytes.Repeat([]byte("archive"), 10000)
	pin := Pin{SHA256: hash(data)}
	if err := storeArchive(root, pin, data); err != nil {
		t.Fatal(err)
	}
	if err := storeArchive(root, pin, data); err != nil {
		t.Fatal(err)
	}
	name := path.Join(CachePath, pin.SHA256+".zip")
	for _, limit := range []int64{int64(len(data)) - 1, int64(len(data)), int64(len(data)) + 1} {
		got, err := readRegularHash(root, name, limit)
		if limit < int64(len(data)) {
			if err == nil {
				t.Fatal("accepted oversized archive")
			}
		} else if err != nil || got != pin.SHA256 {
			t.Fatalf("digest at limit %d = %q, %v", limit, got, err)
		}
	}
	if err := root.WriteFile(name, []byte("corrupt"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := storeArchive(root, pin, data); err == nil || !strings.Contains(err.Error(), "publication conflict") {
		t.Fatalf("corrupt collision: %v", err)
	}
	got, err := root.ReadFile(name)
	if err != nil || string(got) != "corrupt" {
		t.Fatalf("collision replaced existing object: %q, %v", got, err)
	}
	entries, err := root.ReadFile(name)
	if err != nil || len(entries) == 0 {
		t.Fatal(err)
	}
	directory, err := root.Open(CachePath)
	if err != nil {
		t.Fatal(err)
	}
	defer directory.Close()
	names, err := directory.Readdirnames(-1)
	if err != nil || len(names) != 1 {
		t.Fatalf("temporary cleanup = %v, %v", names, err)
	}
}

func BenchmarkExistingArchiveHash(b *testing.B) {
	root, err := os.OpenRoot(b.TempDir())
	if err != nil {
		b.Fatal(err)
	}
	defer root.Close()
	data := bytes.Repeat([]byte{'x'}, MaxArchiveBytes)
	if err := root.WriteFile("archive", data, 0600); err != nil {
		b.Fatal(err)
	}
	want := hash(data)
	for _, streamed := range []bool{false, true} {
		name := "buffered"
		if streamed {
			name = "streamed"
		}
		b.Run(name, func(b *testing.B) {
			b.ReportAllocs()
			for b.Loop() {
				var got string
				var err error
				if streamed {
					got, err = readRegularHash(root, "archive", MaxArchiveBytes)
				} else {
					var encoded []byte
					encoded, err = readRegular(root, "archive", MaxArchiveBytes)
					if err == nil {
						got = hash(encoded)
					}
				}
				if err != nil || got != want {
					b.Fatalf("digest=%s error=%v", got, err)
				}
			}
		})
	}
}
