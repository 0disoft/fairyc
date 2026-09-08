package packageartifact

import (
	"testing"

	"github.com/0disoft/fairyc/internal/hir"
)

func TestByteViewSourceEntitiesRebindAcrossPackages(t *testing.T) {
	for _, kind := range []string{"byte-slice-from-raw-parts", "mut-byte-slice-from-raw-parts", "storage-borrow"} {
		t.Run(kind, func(t *testing.T) {
			original := kind + ":" + hir.QualifiedSourcePath(17, "std/bytes.fy") + ":10:42"
			stable, err := stabilizeSourceEntity(original, 17)
			if err != nil || stable != kind+":@/std/bytes.fy:10:42" {
				t.Fatalf("stable byte-view entity: %q %v", stable, err)
			}
			rebound, err := rebindSourceEntity(stable, 91)
			if err != nil || rebound != kind+":"+hir.QualifiedSourcePath(91, "std/bytes.fy")+":10:42" {
				t.Fatalf("rebound byte-view entity: %q %v", rebound, err)
			}
			if _, err := stabilizeSourceEntity(original, 18); err == nil {
				t.Fatal("entity from another owner admitted")
			}
			if _, err := rebindSourceEntity(original, 91); err == nil {
				t.Fatal("build-local entity admitted as stable")
			}
		})
	}
}
