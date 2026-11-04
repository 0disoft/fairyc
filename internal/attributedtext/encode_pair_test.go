package attributedtext

import (
	"bytes"
	"errors"
	"strings"
	"testing"

	"github.com/0disoft/fairyc/internal/workspaceloader"
)

func TestWorkspacePairSharesOneResolutionAndPreservesDocuments(t *testing.T) {
	for _, source := range []string{attributedDocumentationSource, attributedTextSource} {
		fixture := loadAttributedTextFixture(t, []byte(source))
		for _, textLimit := range []uint64{1, 1 << 20} {
			budget := Budget{MaxFragments: 1, MaxFragmentBytes: textLimit, MaxTextBytes: textLimit}
			wantSource, err := EncodeWorkspaceSource(fixture.workspace, fixture.index, sourceRequest(fixture.visible, budget), 2<<20)
			if err != nil {
				t.Fatal(err)
			}
			wantDoc, err := EncodeWorkspaceDocumentation(fixture.workspace, fixture.index, fixture.snapshot, documentationRequest(fixture.visible, budget), 2<<20)
			if err != nil {
				t.Fatal(err)
			}
			calls := 0
			resolve := func(instance, path string) (workspaceloader.ExactSource, error) {
				calls++
				return fixture.workspace.ResolveExactSource(instance, path)
			}
			gotSource, gotDoc, err := encodeWorkspacePair(resolve, fixture.index, fixture.snapshot, fixture.visible, budget, 2<<20)
			if err != nil || calls != 1 || !bytes.Equal(gotSource, wantSource) || !bytes.Equal(gotDoc, wantDoc) {
				t.Fatalf("pair calls=%d, error=%v, documents match=%t/%t", calls, err, bytes.Equal(gotSource, wantSource), bytes.Equal(gotDoc, wantDoc))
			}
			for _, limit := range []uint64{uint64(len(wantSource)) - 1, uint64(len(wantSource))} {
				_, _, err := encodeWorkspacePair(resolve, fixture.index, fixture.snapshot, fixture.visible, budget, limit)
				wantTooLarge := limit < uint64(len(wantSource)) || limit < uint64(len(wantDoc))
				if errors.Is(err, ErrDocumentTooLarge) != wantTooLarge {
					t.Fatalf("boundary %d: %v", limit, err)
				}
			}
		}
	}
}

func BenchmarkWorkspaceSourceAndDocumentation(b *testing.B) {
	fixture := loadAttributedTextFixture(b, []byte(attributedDocumentationSource+strings.Repeat(" ", 1<<20)))
	budget := Budget{MaxFragments: 1, MaxFragmentBytes: 1 << 20, MaxTextBytes: 1 << 20}
	for _, paired := range []bool{false, true} {
		name := "separate"
		if paired {
			name = "shared"
		}
		b.Run(name, func(b *testing.B) {
			b.ReportAllocs()
			for b.Loop() {
				if paired {
					if _, _, err := EncodeWorkspaceSourceAndDocumentation(fixture.workspace, fixture.index, fixture.snapshot, fixture.visible, budget, 2<<20); err != nil {
						b.Fatal(err)
					}
				} else {
					if _, err := EncodeWorkspaceSource(fixture.workspace, fixture.index, sourceRequest(fixture.visible, budget), 2<<20); err != nil {
						b.Fatal(err)
					}
					if _, err := EncodeWorkspaceDocumentation(fixture.workspace, fixture.index, fixture.snapshot, documentationRequest(fixture.visible, budget), 2<<20); err != nil {
						b.Fatal(err)
					}
				}
			}
		})
	}
}

func TestWorkspacePairRevalidatesEachRequestAndPropagatesErrors(t *testing.T) {
	fixture := loadAttributedTextFixture(t, []byte(attributedDocumentationSource))
	budget := Budget{MaxFragments: 1, MaxFragmentBytes: 1 << 20, MaxTextBytes: 1 << 20}
	if _, _, err := EncodeWorkspaceSourceAndDocumentation(fixture.workspace, fixture.index, fixture.snapshot, fixture.visible, budget, 2<<20); err != nil {
		t.Fatal(err)
	}
	fixture.workspace.Packages[0].Sources[0].Bytes[0] ^= 1
	if _, _, err := EncodeWorkspaceSourceAndDocumentation(fixture.workspace, fixture.index, fixture.snapshot, fixture.visible, budget, 2<<20); err == nil || !strings.Contains(err.Error(), "stale") {
		t.Fatalf("mutated request: %v", err)
	}
	want := errors.New("resolve failed")
	resolve := func(string, string) (workspaceloader.ExactSource, error) { return workspaceloader.ExactSource{}, want }
	if _, _, err := encodeWorkspacePair(resolve, fixture.index, fixture.snapshot, fixture.visible, budget, 2<<20); !errors.Is(err, want) {
		t.Fatalf("resolver error: %v", err)
	}
}
