package packagesource

import "testing"

func TestStableRegistryVersionRanges(t *testing.T) {
	for _, tc := range []struct {
		constraint, version string
		want                bool
	}{
		{"1.2.3", "1.2.3", true}, {"1.2.3", "1.2.4", false}, {"^1.2.3", "1.99.0", true}, {"^1.2.3", "2.0.0", false}, {"^1.2.3", "1.2.2", false},
		{"~1.2.3", "1.2.99", true}, {"~1.2.3", "1.3.0", false}, {"^0.2.3", "0.2.99", true}, {"^0.2.3", "0.3.0", false}, {"^0.0.3", "0.0.4", false},
		{"^4294967295.0.0", "4294967295.4294967295.4294967295", true},
	} {
		t.Run(tc.constraint+"/"+tc.version, func(t *testing.T) {
			r, err := parseVersionRange(tc.constraint)
			if err != nil {
				t.Fatal(err)
			}
			v, err := parseStableVersion(tc.version)
			if err != nil {
				t.Fatal(err)
			}
			if r.contains(v) != tc.want {
				t.Fatal("range boundary mismatch")
			}
		})
	}
	for _, bad := range []string{"*", "latest", "1", "1.x", ">=1.0.0", "^01.0.0", "1.0.0-alpha", "1.0.0+build", " 1.0.0", "1.0.0 ", "4294967296.0.0", "+1.0.0"} {
		if _, err := parseVersionRange(bad); err == nil {
			t.Fatalf("admitted %q", bad)
		}
	}
}
