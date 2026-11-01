package packagecache

import (
	"crypto/sha256"
	"fmt"
	"slices"
	"testing"
)

func TestInlineCacheClassAndV4Compatibility(t *testing.T) {
	inputs := testInputs(t)
	legacy, err := EncodeLegacyV4(inputs)
	if err != nil {
		t.Fatal(err)
	}
	if fmt.Sprintf("%x", sha256.Sum256(legacy)) != "51508c63893427d1cdc799f8d478c25102ea28d3ce93e0bd6b918d817ccef96f" {
		t.Fatal("v4 tuple changed")
	}
	if _, err := ParseKey("fycache4-sha256-51508c63893427d1cdc799f8d478c25102ea28d3ce93e0bd6b918d817ccef96f"); err == nil {
		t.Fatal("v4 key admitted by v5")
	}
	baseline, err := Compute(inputs)
	if err != nil {
		t.Fatal(err)
	}
	changed := inputs
	changed.InlineBodies[0] = 1
	owner, err := Compute(changed)
	if err != nil || owner == baseline {
		t.Fatal("inline owner class omitted", err)
	}
	changed = inputs
	changed.Dependencies = slices.Clone(inputs.Dependencies)
	changed.Dependencies[0].InlineBodies[0] = 2
	consumer, err := Compute(changed)
	if err != nil || consumer == baseline || consumer == owner {
		t.Fatal("inline dependency class omitted", err)
	}
	if _, err := EncodeLegacyV4(changed); err == nil {
		t.Fatal("v4 silently discarded inline dependency")
	}
}
