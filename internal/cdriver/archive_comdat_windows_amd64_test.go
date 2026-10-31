//go:build windows && amd64

package cdriver

import (
	"crypto/sha256"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/0disoft/fairyc/internal/pecoff"
)

func TestWindowsCRTArchiveCOMDAT(t *testing.T) {
	tc, err := ResolveWindowsHostMSVC()
	if err != nil {
		t.Fatal(err)
	}
	source := `#include <stdlib.h>
#include <string.h>
__declspec(noinline) int observe(unsigned char *p,size_t n){memset(p,42,n);return p[n-1];}
int main(void){unsigned char *p=malloc(37);if(!p)return 1;int result=observe(p,37);free(p);return result;}
`
	t.Logf("MSVC %s source=%x", tc.Version, sha256.Sum256([]byte(source)))
	for _, profile := range []Profile{LowProfile, OptimizedProfile} {
		t.Run(profile.Name, func(t *testing.T) {
			a, err := BuildWindowsMSVCProbeWithLibraryLog(tc, profile, source)
			if err != nil {
				t.Fatal(err)
			}
			report, err := pecoff.InspectMSVCDefaultLibraries([][]byte{a.Objects[0].Bytes}, a.Bytes, a.LinkMap, a.LinkerVerbose)
			if err != nil {
				t.Fatal(err)
			}
			libraries := map[string][]byte{}
			for _, lib := range report.Libraries {
				if lib.Name == "libcmt.lib" || lib.Name == "libucrt.lib" || lib.Name == "libvcruntime.lib" {
					raw, err := os.ReadFile(lib.Path)
					if err != nil {
						t.Fatal(err)
					}
					libraries[lib.Name] = raw
					t.Logf("archive %s sha256=%x", lib.Name, sha256.Sum256(raw))
				}
			}
			for _, demand := range []struct{ library, symbol string }{{"libcmt.lib", "mainCRTStartup"}, {"libucrt.lib", "__acrt_initialize"}, {"libvcruntime.lib", "__vcrt_initialize"}, {"libucrt.lib", "malloc"}, {"libvcruntime.lib", "memset"}} {
				archive := libraries[demand.library]
				member, err := pecoff.InspectIndexedArchiveObject(archive, demand.symbol)
				if err != nil {
					t.Fatal(err)
				}
				raw := archive[member.DataOffset : uint64(member.DataOffset)+uint64(member.Size)]
				modes := map[uint8]int{}
				for _, s := range member.Object.Sections {
					if s.COMDATSelection != 0 && s.COMDATSelection != pecoff.ObjectCOMDATNoDuplicates && s.COMDATSelection != pecoff.ObjectCOMDATAny && s.COMDATSelection != pecoff.ObjectCOMDATAssociative {
						t.Fatal("new CRT COMDAT mode needs review")
					}
					modes[s.COMDATSelection]++
				}
				var root pecoff.ObjectSymbol
				matches := 0
				for _, s := range member.Object.SymbolRecords {
					if s.Name == demand.symbol && s.SectionNumber > 0 {
						root = s
						matches++
					}
				}
				if matches != 1 {
					t.Fatal("ambiguous CRT root")
				}
				owner := strings.TrimSuffix(demand.library, ".lib") + ":" + filepath.Base(member.Name)
				owned := 0
				for _, line := range strings.Split(string(a.LinkMap), "\n") {
					fields := strings.Fields(line)
					if len(fields) >= 4 && fields[1] == demand.symbol {
						if !strings.EqualFold(fields[len(fields)-1], owner) {
							t.Fatal("CRT indexed member differs from map owner")
						}
						owned++
					}
				}
				if owned != 1 {
					t.Fatal("missing selected CRT owner")
				}
				loaded := false
				for _, lib := range report.Libraries {
					if lib.Name == demand.library {
						for _, name := range lib.Members {
							if strings.EqualFold(name, filepath.Base(member.Name)) {
								loaded = true
							}
						}
					}
				}
				if !loaded {
					t.Fatal("indexed CRT member was not loaded")
				}
				t.Logf("indexed %s(%s) symbol=%s offset=%d size=%d sha256=%x modes=%v root-selection=%d", demand.library, member.Name, demand.symbol, member.HeaderOffset, member.Size, sha256.Sum256(raw), modes, member.Object.Sections[root.SectionNumber-1].COMDATSelection)
				hasChildren := false
				for _, s := range member.Object.Sections {
					if s.Name == ".pdata" && s.COMDATSelection == pecoff.ObjectCOMDATAssociative && s.AssociativeSection == int(root.SectionNumber) {
						hasChildren = true
					}
				}
				if hasChildren {
					p, err := pecoff.InspectAssociativeCOMDATUnwindLink(raw, a.Bytes, a.LinkMap, pecoff.AssociativeCOMDATUnwindSpec{ArchiveName: demand.library, SourceMember: filepath.Base(member.Name), SourceSymbol: demand.symbol})
					if err != nil {
						t.Fatal(err)
					}
					t.Logf("CRT %s parent=%d pdata=%d xdata=%d final=%#x..%#x unwind=%#x", demand.symbol, p.ParentSectionIndex, p.PDataSectionIndex, p.XDataSectionIndex, p.FinalBeginRVA, p.FinalEndRVA, p.FinalUnwindRVA)
				} else {
					t.Logf("CRT %s code-section=%s selection=%d bytes=%d has no direct associative pdata; no associative triple claimed", demand.symbol, member.Object.Sections[root.SectionNumber-1].Name, member.Object.Sections[root.SectionNumber-1].COMDATSelection, member.Object.Sections[root.SectionNumber-1].RawSize)
				}
				if demand.symbol == "mainCRTStartup" {
					if member.Object.Sections[root.SectionNumber-1].COMDATSelection != pecoff.ObjectCOMDATNoDuplicates {
						t.Fatal("startup no longer exercises NODUPLICATES")
					}
					objects := append([]WindowsHostObject{}, a.Objects...)
					objects = append(objects, WindowsHostObject{Name: "startup-one.obj", Bytes: raw}, WindowsHostObject{Name: "startup-two.obj", Bytes: raw})
					_, output, _, err := linkCRTInputs(t, tc, objects, nil, false, "")
					requireCRTLinkFailure(t, err, output, "LNK2005", "mainCRTStartup")
				}
			}
			run, err := ExecuteWindowsMSVCTarget(a)
			if err != nil || run.ExitCode != 42 || run.Stdout != "" || run.Stderr != "" {
				t.Fatalf("CRT archive execution %+v %v", run, err)
			}
			t.Logf("CRT control object=%x image=%x", sha256.Sum256(a.Objects[0].Bytes), sha256.Sum256(a.Bytes))
		})
	}
}
