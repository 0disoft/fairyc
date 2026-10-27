package attributedtext

import (
	"fmt"
	"strings"
	"testing"
)

// Compile and index outside the timer; measure both lookup and public extraction.
func BenchmarkWorkspaceDocumentation(b *testing.B) {
	for _, count := range []int{64, 512} {
		b.Run(fmt.Sprintf("extra-declarations=%d", count), func(b *testing.B) {
			var contents strings.Builder
			contents.WriteString(attributedDocumentationSource)
			for i := 0; i < count; i++ {
				fmt.Fprintf(&contents, "\n/// Extra documentation %d.\nfn i32 extra_%04d(i32 value) { return value; }\n", i, i)
			}
			fixture := loadAttributedTextFixture(b, []byte(contents.String()))
			subject, err := fixture.index.ResolveAttributionSubject(fixture.visible)
			if err != nil {
				b.Fatal(err)
			}
			b.Run("span", func(b *testing.B) {
				b.ReportAllocs()
				for b.Loop() {
					_, found, err := exactDocumentationSpan(fixture.snapshot, subject)
					if err != nil || !found {
						b.Fatalf("documentation span: found=%t, error=%v", found, err)
					}
				}
			})
			b.Run("extract", func(b *testing.B) {
				request := documentationRequest(fixture.visible, Budget{
					MaxFragments: 1, MaxFragmentBytes: 1 << 20, MaxTextBytes: 1 << 20,
				})
				b.ReportAllocs()
				for b.Loop() {
					result, err := ExtractWorkspaceDocumentation(fixture.workspace, fixture.index, fixture.snapshot, request)
					if err != nil || len(result.Records) != 1 {
						b.Fatalf("documentation extraction: records=%d, error=%v", len(result.Records), err)
					}
				}
			})
		})
	}
}
