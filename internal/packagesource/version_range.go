package packagesource

import (
	"cmp"
	"fmt"
	"slices"
	"strconv"
	"strings"
)

// Registry resolution initially admits stable uint32 SemVer components only.
// Existing exact manifest prerelease/build spellings are unaffected.
type stableVersion [3]uint64

func parseStableVersion(value string) (stableVersion, error) {
	var result stableVersion
	parts := strings.Split(value, ".")
	if len(parts) != 3 || len(value) > 32 {
		return result, fmt.Errorf("registry version needs three stable uint32 components")
	}
	for i, part := range parts {
		n, err := strconv.ParseUint(part, 10, 32)
		if err != nil || strconv.FormatUint(n, 10) != part {
			return stableVersion{}, fmt.Errorf("noncanonical registry stable version")
		}
		result[i] = n
	}
	return result, nil
}

type versionRange struct {
	lower, upper stableVersion
	exact        bool
}

func parseVersionRange(value string) (versionRange, error) {
	op := ""
	base := value
	if strings.HasPrefix(value, "^") || strings.HasPrefix(value, "~") {
		op, base = value[:1], value[1:]
	}
	v, err := parseStableVersion(base)
	if err != nil {
		return versionRange{}, err
	}
	r := versionRange{lower: v, upper: v, exact: op == ""}
	if op == "~" {
		r.upper[1]++
		r.upper[2] = 0
	}
	if op == "^" {
		index := 0
		if v[0] == 0 {
			index = 1
			if v[1] == 0 {
				index = 2
			}
		}
		r.upper[index]++
		for i := index + 1; i < 3; i++ {
			r.upper[i] = 0
		}
	}
	return r, nil
}

func compareStable(a, b stableVersion) int {
	return slices.CompareFunc(a[:], b[:], cmp.Compare[uint64])
}
func (r versionRange) contains(v stableVersion) bool {
	if r.exact {
		return v == r.lower
	}
	return compareStable(v, r.lower) >= 0 && compareStable(v, r.upper) < 0
}
