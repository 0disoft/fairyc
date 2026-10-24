package durablefile

import (
	"errors"
	"os"
	"testing"
)

func TestPublicationSyncsEverySuccessPath(t *testing.T) {
	for _, path := range []string{"published", "converged", "replaced", "late-converged"} {
		for _, syncFails := range []bool{false, true} {
			name := path + "/sync-ok"
			if syncFails {
				name = path + "/sync-fails"
			}
			t.Run(name, func(t *testing.T) {
				root, err := os.OpenRoot(t.TempDir())
				if err != nil {
					t.Fatal(err)
				}
				defer root.Close()
				payload := []byte("canonical-payload")
				if err := root.WriteFile("temporary", payload, 0o600); err != nil {
					t.Fatal(err)
				}
				if path != "published" {
					initial := []byte("corrupt")
					if path == "converged" {
						initial = payload
					}
					if err := root.WriteFile("final", initial, 0o600); err != nil {
						t.Fatal(err)
					}
				}
				moves, syncs := 0, 0
				move := func(root *os.Root, from, to string) error {
					moves++
					if path == "published" || path == "replaced" && moves == 2 {
						return root.Rename(from, to)
					}
					if path == "late-converged" && moves == 2 {
						// Another publisher wins after the initial mismatch/removal.
						if err := root.WriteFile(to, payload, 0o600); err != nil {
							t.Fatal(err)
						}
					}
					return os.ErrPermission
				}
				failure := errors.New("test sync failure")
				flush := func(opened *os.Root, final, directory string) error {
					syncs++
					if opened != root || final != "final" || directory != "." {
						t.Fatal("sync escaped the publication boundary")
					}
					if syncFails {
						return failure
					}
					return nil
				}
				err = publishImmutable(root, "temporary", "final", payload, move, flush)
				if syncs != 1 {
					t.Errorf("sync calls: got %d, want 1", syncs)
				}
				if syncFails && !errors.Is(err, failure) || !syncFails && err != nil {
					t.Errorf("sync failure propagation: got %v, failure expected=%t", err, syncFails)
				}
				wantMoves := 1
				if path == "replaced" || path == "late-converged" {
					wantMoves = 2
				}
				if moves != wantMoves || !matchesFile(root, "final", payload) {
					t.Errorf("wrong publication path: moves=%d, want %d", moves, wantMoves)
				}
			})
		}
	}
}

func TestPublicationConflictDoesNotSync(t *testing.T) {
	for _, directory := range []bool{false, true} {
		name := "mismatched-destination"
		if directory {
			name = "nonempty-directory"
		}
		t.Run(name, func(t *testing.T) {
			root, err := os.OpenRoot(t.TempDir())
			if err != nil {
				t.Fatal(err)
			}
			defer root.Close()
			if directory {
				if err := root.Mkdir("final", 0o700); err != nil {
					t.Fatal(err)
				}
				if err := root.WriteFile("final/keep", []byte("unrelated"), 0o600); err != nil {
					t.Fatal(err)
				}
			} else if err := root.WriteFile("final", []byte("corrupt"), 0o600); err != nil {
				t.Fatal(err)
			}
			moves, syncs := 0, 0
			move := func(*os.Root, string, string) error { moves++; return os.ErrPermission }
			flush := func(*os.Root, string, string) error { syncs++; return nil }
			if err := publishImmutable(root, "temporary", "final", []byte("canonical-payload"), move, flush); err == nil {
				t.Fatal("publication conflict was accepted")
			}
			wantMoves := 2
			if directory {
				wantMoves = 1
				if data, err := root.ReadFile("final/keep"); err != nil || string(data) != "unrelated" {
					t.Fatalf("unrelated file changed: %q, %v", data, err)
				}
			}
			if moves != wantMoves || syncs != 0 {
				t.Fatalf("failed publication: moves=%d, want %d; syncs=%d, want 0", moves, wantMoves, syncs)
			}
		})
	}
}
