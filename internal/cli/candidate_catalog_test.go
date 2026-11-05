package cli

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"slices"
	"strconv"
	"strings"
	"testing"

	"github.com/0disoft/fairyc/internal/atlasmap"
	"github.com/0disoft/fairyc/internal/semanticsnapshot"
	"github.com/0disoft/fairyc/internal/symbolcatalog"
)

const candidateCatalogMain = `module main;import project_catalog as catalog;import project_catalog_options as options;import project_atlas_current as current;` + atlasReadInput + `
@effects(console) fn i32 main(){
 let request=switch(options::arguments()){case .ok(p):{p}case .err(_):{return 2;}};
 let data=input();let raw=bytes::view(ref data);var usize at=0;while(at<raw.len && raw[at]!=10){at+=1;}if(at==raw.len){return 7;}
 var usize end=at+1;while(end<raw.len && raw[end]!=10){end+=1;}if(end==raw.len){return 7;}
 let source=switch(bytes::copy(raw[..at])){case .ok(p):{p}case .err(_):{return 7;}};
 let expected=switch(bytes::copy(raw[at+1..end])){case .ok(p):{p}case .err(_):{return 7;}};let mode=raw[end+1..];var usize meter=0;
 var u32 version=options::version(ref request);if(bytes::equal(mode,b"1")){version=1;}else if(bytes::equal(mode,b"2")){version=2;}else if(bytes::equal(mode,b"3")){version=3;}
 let ready=switch(if(version==2){current::read_v2(source,expected,ref mut meter)}else if(version==3){current::read_generics(source,expected,ref mut meter)}else{current::read(source,expected,ref mut meter)}){case .ok(p):{p}case .err(_):{return 7;}};
 if(bytes::equal(mode,b"work")){meter=16777216;}
 let out=switch(catalog::encode(ref ready,ref request,ref mut meter)){case .ok(p):{p}case .err(_):{return 7;}};
 return if(io::write_bytes(bytes::view(ref out)).is_ok){0}else{4};
}`

type candidateCatalogFixture struct {
	name, uid      string
	version        int
	data, expected []byte
	index          *atlasmap.QueryIndex
}

func candidateCatalogFixtures(t *testing.T) []candidateCatalogFixture {
	t.Helper()
	common := `module main;import answer as a;
pub struct Cell{i32 value;}struct Other{i32 value;}
fn i32 Cell::get(ref Cell this){return this.value;}fn i32 Other::get(ref Other this){return this.value;}
fn i32 get(){return 1;}pub enum State{ready;done;}pub resource Empty{cleanup{}}
pub unsafe extern C fn i32 external_value();`
	sources := []string{
		common + `fn i32 main(){return a::value();}`,
		common + `pub fn T id<T>(T value){return value;}pub fn T unused<T>(T value){return value;}fn i32 main(){discard id<u32>(7);return id<i32>(a::value());}`,
		common + `pub struct Box<T>{T value;}pub resource Token<T>{Box<T> payload;cleanup{}}pub struct Unused<T>{T value;}pub resource UnusedToken<T>{T value;cleanup{}}
fn Token<T> make<T>(T value){return Token<T>{.payload=Box<T>{.value=value}};}fn i32 main(){let token=make<i32>(a::value());discard make<u32>(7);return token.payload.value;}`,
	}
	var fixtures []candidateCatalogFixture
	for v, schema := range []string{atlasmap.LogicalSchema, atlasmap.LogicalSchemaV2, atlasmap.LogicalSchemaV3} {
		root := writeWorkspaceSessionFixture(t, sources[v])
		writeWorkspaceFile(t, root, "src/extra.fy", `module extra;pub fn i32 get(){return 5;}`)
		var diagnostic bytes.Buffer
		prepared, code := prepareAtlasWorkspaceSchema(root, false, &diagnostic, schema)
		if code != 0 {
			t.Fatalf("catalog v%d source oracle: %d %s", v+1, code, &diagnostic)
		}
		for _, p := range prepared.partitions {
			encode := []func(atlasmap.Partition, uint64) ([]byte, error){atlasmap.Encode, atlasmap.EncodeV2, atlasmap.EncodeV3}[v]
			query := []func(atlasmap.Partition, atlasmap.LoadExpectation) (*atlasmap.QueryIndex, error){atlasmap.NewQueryIndex, atlasmap.NewQueryIndexV2, atlasmap.NewQueryIndexV3}[v]
			data, err := encode(p, 4<<20)
			if err != nil {
				t.Fatal(err)
			}
			e := prepared.expectations[p.Header().PackageInstance]
			index, err := query(p, e)
			if err != nil {
				t.Fatal(err)
			}
			fixtures = append(fixtures, candidateCatalogFixture{fmt.Sprintf("v%d-%s", v+1, p.Header().PackageUID), p.Header().PackageUID, v + 1, data, candidateAtlasCurrent(t, e), index})
		}
	}
	return fixtures
}

func candidateCatalogArgs(f candidateCatalogFixture, r atlasmap.SymbolCatalogRequest, output uint64) []string {
	args := []string{"experimental", "atlas-symbols", "list", "--catalog-version", strconv.Itoa(f.version), "--package-uid", f.uid, "--max-symbols", strconv.Itoa(r.Budget.MaxSymbols), "--max-selection-bytes", strconv.FormatUint(r.Budget.MaxSelectionBytes, 10), "--max-output-bytes", strconv.FormatUint(output, 10)}
	for _, k := range r.Kinds {
		args = append(args, "--kind", string(k))
	}
	for _, field := range [][2]string{{"--module", r.Module}, {"--name", r.Name}, {"--after", r.After}} {
		if field[1] != "" {
			args = append(args, field[:]...)
		}
	}
	return args
}

func candidateCatalogOracle(t *testing.T, f candidateCatalogFixture, r atlasmap.SymbolCatalogRequest, output uint64) ([]byte, error) {
	t.Helper()
	encode := []func(*atlasmap.QueryIndex, atlasmap.SymbolCatalogRequest, uint64) ([]byte, error){symbolcatalog.Encode, symbolcatalog.EncodeV2, symbolcatalog.EncodeV3}[f.version-1]
	data, err := encode(f.index, r, output)
	if err != nil {
		return data, err
	}
	// Independently recompute framing; do not trust the Go writer's checksum.
	var envelope struct {
		Version  string          `json:"catalog_version"`
		Encoding string          `json:"encoding"`
		Checksum string          `json:"checksum"`
		Catalog  json.RawMessage `json:"catalog"`
	}
	if err := json.Unmarshal(data, &envelope); err != nil {
		t.Fatal(err)
	}
	h := sha256.New()
	h.Write([]byte("fairyc.symbol-catalog-checksum"))
	h.Write([]byte{0, byte(f.version - 1)})
	for _, value := range [][]byte{[]byte(envelope.Version), []byte(envelope.Encoding), envelope.Catalog} {
		var length [8]byte
		binary.BigEndian.PutUint64(length[:], uint64(len(value)))
		h.Write(length[:])
		h.Write(value)
	}
	if want := fmt.Sprintf("fyscat%d-sha256-", f.version) + hex.EncodeToString(h.Sum(nil)); envelope.Checksum != want {
		t.Fatal("independent catalog checksum differs")
	}
	return data, nil
}

func TestCandidateCatalog(t *testing.T) {
	fixtures := candidateCatalogFixtures(t)
	candidateAtlasProbe(t, "project_catalog", candidateCatalogMain, func(t *testing.T, image string) {
		calls := 0
		for _, f := range fixtures {
			t.Run(f.name, func(t *testing.T) {
				input := append(append(append(append([]byte{}, f.data...), '\n'), f.expected...), '\n')
				run := func(name string, r atlasmap.SymbolCatalogRequest, output uint64) {
					t.Helper()
					calls++
					t.Run(name, func(t *testing.T) {
						want, err := candidateCatalogOracle(t, f, r, output)
						code, got := candidateAtlasProcess(t, image, input, candidateCatalogArgs(f, r, output)...)
						if err != nil {
							if code != 7 || got != "" {
								t.Fatalf("atomic refusal: %d %q, Go=%v", code, got, err)
							}
						} else if code != 0 || got != string(want) {
							t.Fatalf("catalog bytes differ: code=%d\n got=%s\nwant=%s", code, got, want)
						}
					})
				}
				r := atlasmap.SymbolCatalogRequest{Kinds: atlasmap.AllDeclarationKinds(), Budget: atlasmap.SymbolCatalogBudget{MaxSymbols: 65536, MaxSelectionBytes: 1 << 20}}
				all, err := f.index.CatalogSymbols(r)
				if err != nil || !all.Complete || all.Total == 0 || len(all.Symbols) != all.Total {
					t.Fatalf("full source catalog: %+v %v", all, err)
				}
				for i, s := range all.Symbols {
					if i > 0 && all.Symbols[i-1].Symbol.Full >= s.Symbol.Full {
						t.Fatal("source catalog is not strictly full-ID ordered")
					}
				}
				run("all", r, 1<<20)
				want, err := candidateCatalogOracle(t, f, r, 1<<20)
				if err != nil {
					t.Fatal(err)
				}
				for _, n := range []int{len(want) - 1, len(want), len(want) + 1} {
					run(fmt.Sprintf("output-%d", n), r, uint64(n))
				}
				for _, kind := range atlasmap.AllDeclarationKinds() {
					filtered := r
					filtered.Kinds = []semanticsnapshot.DeclarationKind{kind}
					run("kind-"+string(kind), filtered, 1<<20)
				}
				for _, name := range []string{"get", "Cell::get", "Other::get", "Get", "missing", "unused", "Unused", "UnusedToken"} {
					filtered := r
					filtered.Name = name
					run("name-"+name, filtered, 1<<20)
				}
				for _, module := range []string{"main", "extra", "answer", "absent"} {
					filtered := r
					filtered.Module, filtered.Name = module, "get"
					run("module-"+module, filtered, 1<<20)
				}
				page := r
				page.Budget.MaxSymbols = 1
				for i := 0; i <= all.Total; i++ {
					run(fmt.Sprintf("page-%d", i), page, 1<<20)
					if i < all.Total {
						page.After = all.Symbols[i].Symbol.Full
					}
				}
				page.After = "fysym1-sha256-256-" + strings.Repeat("0", 64)
				run("absent-cursor", page, 1<<20)
				page.After, page.Name = all.Symbols[0].Symbol.Full, "absent"
				run("cursor-outside-filter", page, 1<<20)
				// Mandatory context and each entry cost include one separator byte.
				one := r
				one.Budget.MaxSymbols = 1
				first, err := f.index.CatalogSymbols(one)
				if err != nil || len(first.Symbols) != 1 {
					t.Fatalf("first page: %v", err)
				}
				entry, _ := json.Marshal(first.Symbols[0])
				mandatory := first.Usage.SelectionBytes - uint64(len(entry)+1)
				for _, limit := range []uint64{1, mandatory - 1, mandatory, mandatory + 1, first.Usage.SelectionBytes - 1, first.Usage.SelectionBytes, first.Usage.SelectionBytes + 1, all.Usage.SelectionBytes - 1, all.Usage.SelectionBytes, all.Usage.SelectionBytes + 1} {
					limited := r
					limited.Budget.MaxSelectionBytes = limit
					run(fmt.Sprintf("selection-%d", limit), limited, 1<<20)
				}
				empty := r
				empty.Name = "absent"
				e, err := f.index.CatalogSymbols(empty)
				if err != nil || e.Total != 0 {
					t.Fatalf("empty catalog: %v", err)
				}
				for _, delta := range []int{-1, 0, 1} {
					empty.Budget.MaxSelectionBytes = uint64(int(e.Usage.SelectionBytes) + delta)
					run(fmt.Sprintf("empty-selection-%d", delta), empty, 1<<20)
				}
				for _, mode := range []string{"stale", "corrupt", "work", "uid", "version"} {
					calls++
					data := slices.Clone(input)
					args := candidateCatalogArgs(f, r, 1<<20)
					switch mode {
					case "stale":
						changed := bytes.Replace(f.expected, []byte(`"compiler_build":"`), []byte(`"compiler_build":"stale-`), 1)
						data = append(append(append(append([]byte{}, f.data...), '\n'), changed...), '\n')
					case "corrupt":
						data[20] ^= 1
					case "work":
						data = append(data, "work"...)
					case "uid":
						args[6] = "wrong.package"
					case "version":
						args[4] = strconv.Itoa(f.version%3 + 1)
						// Admit the actual map first, then test the encoder's own
						// cross-version refusal rather than just the reader gate.
						data = append(data, strconv.Itoa(f.version)...)
					}
					code, got := candidateAtlasProcess(t, image, data, args...)
					if code != 7 || got != "" {
						t.Fatalf("%s must reject atomically: %d %q", mode, code, got)
					}
				}
			})
		}
		t.Logf("catalog native processes: %d", calls)
	})
}
