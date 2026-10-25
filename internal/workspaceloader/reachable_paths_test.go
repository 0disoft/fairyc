package workspaceloader

import (
	"fmt"
	"slices"
	"strings"
	"testing"
)

func TestReachablePathsPreserveFirstMatchesAndFreshUIDOrder(t *testing.T) {
	workspace := metadataWorkspace(t, 5)
	workspace.Packages[0].Manifest.Package.UID = "z"
	workspace.Packages[1].Manifest.Package.UID = "a"
	workspace.Packages[2].Manifest.Package.UID = "a"
	workspace.Packages[3].SourceKind = SourceKindSDK
	workspace.Resolution.InstanceClasses = workspace.Resolution.InstanceClasses[:4]
	slices.Reverse(workspace.Packages)
	slices.Reverse(workspace.Instances)
	// Duplicate catalog entries must not override the first matching records.
	duplicate := workspace.Packages[4]
	duplicate.SourceKind = SourceKindSDK
	workspace.Packages = append(workspace.Packages, duplicate)
	duplicateInstance := workspace.Instances[4]
	duplicateInstance.PackagePath = "missing"
	workspace.Instances = append(workspace.Instances, duplicateInstance)
	check := func(want []string) {
		t.Helper()
		got, err := workspace.RootReachableLocalPackagePaths()
		if err != nil || !slices.Equal(got, want) {
			t.Fatalf("reachable paths = %v, %v; want %v", got, err, want)
		}
	}
	check([]string{"package-0001", "package-0002", "package-0000"})
	workspace.Packages[4].Manifest.Package.UID = "0"
	check([]string{"package-0000", "package-0001", "package-0002"})
	workspace.Packages[4].SourceKind = SourceKindSDK
	check([]string{"package-0001", "package-0002"})
	// Mapping edits between calls must also be visible, without a retained index.
	workspace.Instances[4].PackagePath = "package-0004"
	check([]string{"package-0001", "package-0002", "package-0004"})
}

func TestReachablePathsKeepInvalidMappingErrors(t *testing.T) {
	for _, test := range []struct {
		name, want string
		mutate     func(*Workspace)
	}{
		{"empty class", "workspace has an empty package instance class", func(w *Workspace) {
			w.Resolution.InstanceClasses[0].Members = nil
		}},
		{"missing instance", "has no package mapping", func(w *Workspace) {
			w.Instances = w.Instances[1:]
		}},
		{"missing package", `workspace package "package-0000" is missing`, func(w *Workspace) {
			w.Packages = w.Packages[1:]
		}},
	} {
		t.Run(test.name, func(t *testing.T) {
			workspace := metadataWorkspace(t, 4)
			test.mutate(&workspace)
			if _, err := workspace.RootReachableLocalPackagePaths(); err == nil || !strings.Contains(err.Error(), test.want) {
				t.Fatalf("error = %v, want %q", err, test.want)
			}
		})
	}
}

func BenchmarkRootReachableLocalPackagePaths(b *testing.B) {
	for _, count := range []int{16, 1024} {
		b.Run(fmt.Sprintf("packages-%d", count), func(b *testing.B) {
			workspace := metadataWorkspace(b, count)
			last := fmt.Sprintf("package-%04d", count-1)
			b.ReportAllocs()
			for b.Loop() {
				paths, err := workspace.RootReachableLocalPackagePaths()
				if err != nil || len(paths) != count || paths[0] != "package-0000" || paths[count-1] != last {
					b.Fatalf("wrong reachable paths: count=%d, error=%v", len(paths), err)
				}
			}
		})
	}
}
