// Package scipindex projects verified Fairy C workspace facts into SCIP.
package scipindex

import (
	"cmp"
	"fmt"
	"net/url"
	"path/filepath"
	"slices"
	"sort"
	"strconv"
	"strings"

	"github.com/0disoft/fairyc/internal/compiler"
	"github.com/0disoft/fairyc/internal/diagnostic"
	"github.com/0disoft/fairyc/internal/hir"
	"github.com/0disoft/fairyc/internal/semantic"
	"github.com/0disoft/fairyc/internal/workspaceloader"
	"github.com/scip-code/scip/bindings/go/scip"
	"google.golang.org/protobuf/proto"
)

const maxOccurrences = 131_072
const maxIndexBytes = 16 << 20

type fileKey struct {
	pkg  hir.PackageID
	path string
}

type document struct {
	wire    *scip.Document
	source  workspaceloader.SourceFile
	symbols map[diagnostic.Span]string
}

type useFact struct {
	symbol    string
	ambiguous bool
}

// Marshal uses the exact source snapshot used by the verified compiler.
func Marshal(workspace workspaceloader.Workspace, checked compiler.Result) ([]byte, error) {
	if !checked.Verified {
		return nil, fmt.Errorf("SCIP requires verified workspace source")
	}
	root, err := filepath.Abs(workspace.Root)
	if err != nil {
		return nil, err
	}
	projectPath := filepath.ToSlash(root)
	projectURI := (&url.URL{Scheme: "file", Path: "/" + strings.TrimPrefix(projectPath, "/")}).String()
	if host, path, unc := strings.Cut(strings.TrimPrefix(projectPath, "//"), "/"); strings.HasPrefix(projectPath, "//") && unc {
		projectURI = (&url.URL{Scheme: "file", Host: host, Path: "/" + path}).String()
	}
	index := &scip.Index{Metadata: &scip.Metadata{
		ToolInfo:    &scip.ToolInfo{Name: "fairyc", Version: "scip-preview-1"},
		ProjectRoot: projectURI, TextDocumentEncoding: scip.TextEncoding_UTF8,
	}}
	documents, err := collectDocuments(workspace)
	if err != nil {
		return nil, err
	}
	uses := make(map[hir.Span]useFact)
	for _, records := range [][]semantic.Resolution{checked.Semantic.Resolutions, checked.Semantic.SelectorResolutions} {
		for _, record := range records {
			if err := collectUse(documents, uses, record); err != nil {
				return nil, err
			}
		}
	}
	count := len(uses)
	for _, doc := range documents {
		count += len(doc.symbols)
		if count > maxOccurrences {
			return nil, fmt.Errorf("SCIP occurrence limit exceeded")
		}
		for span, symbol := range doc.symbols {
			doc.add(span, symbol, int32(scip.SymbolRole_Definition))
			if scip.IsGlobalSymbol(symbol) {
				doc.wire.Symbols = append(doc.wire.Symbols, &scip.SymbolInformation{Symbol: symbol})
			}
		}
	}
	for span, use := range uses {
		if use.symbol != "" && !use.ambiguous {
			doc := documents[fileKey{span.Package, span.Source}]
			sourceSpan := diagnostic.Span{StartByte: span.StartByte, EndByte: span.EndByte}
			if doc.symbols[sourceSpan] == "" {
				doc.add(sourceSpan, use.symbol, 0)
			}
		}
	}
	for _, doc := range documents {
		slices.SortFunc(doc.wire.Symbols, func(a, b *scip.SymbolInformation) int { return cmp.Compare(a.Symbol, b.Symbol) })
		slices.SortFunc(doc.wire.Occurrences, compareOccurrences)
		index.Documents = append(index.Documents, doc.wire)
	}
	slices.SortFunc(index.Documents, func(a, b *scip.Document) int { return cmp.Compare(a.RelativePath, b.RelativePath) })
	if proto.Size(index) > maxIndexBytes {
		return nil, fmt.Errorf("SCIP index exceeds 16 MiB")
	}
	return proto.Marshal(index)
}

func collectDocuments(workspace workspaceloader.Workspace) (map[fileKey]*document, error) {
	packages, err := workspace.SemanticPackages()
	if err != nil {
		return nil, err
	}
	documents := make(map[fileKey]*document)
	paths := make(map[string]bool)
	for _, pkg := range packages {
		if pkg.SourceKind != workspaceloader.SourceKindLocal {
			continue
		}
		loaded, found := workspace.Package(pkg.Path)
		if !found {
			return nil, fmt.Errorf("SCIP package is unavailable")
		}
		for _, source := range loaded.Sources {
			path, err := filepath.Rel(workspace.Root, filepath.Join(loaded.SourceRoot, filepath.FromSlash(source.Path)))
			path = filepath.ToSlash(path)
			if err != nil || path == "." || !filepath.IsLocal(filepath.FromSlash(path)) || paths[path] {
				return nil, fmt.Errorf("SCIP document path escapes the workspace or has multiple package instances")
			}
			paths[path] = true
			doc := &document{
				wire:   &scip.Document{RelativePath: path, Language: "fairyc", PositionEncoding: scip.PositionEncoding_UTF8CodeUnitOffsetFromLineStart},
				source: source, symbols: make(map[diagnostic.Span]string),
			}
			doc.addDeclarations(pkg)
			documents[fileKey{pkg.ID, source.Path}] = doc
		}
	}
	return documents, nil
}

func collectUse(documents map[fileKey]*document, uses map[hir.Span]useFact, record semantic.Resolution) error {
	use, target := record.Use, record.Target.Declaration
	owner := documents[fileKey{use.Package, use.Source}]
	definition := documents[fileKey{target.Package, target.Source}]
	if owner == nil || !owner.validSpan(use.StartByte, use.EndByte) {
		return nil
	}
	var symbol string
	if definition != nil && definition.validSpan(target.StartByte, target.EndByte) {
		span := diagnostic.Span{StartByte: target.StartByte, EndByte: target.EndByte}
		symbol = definition.symbols[span]
		if symbol == "" && owner == definition && (record.Target.Kind == "local" || record.Target.Kind == "parameter") &&
			string(definition.source.Bytes[span.StartByte:span.EndByte]) == record.Target.Name {
			symbol = "local " + strconv.Itoa(span.StartByte)
			definition.symbols[span] = symbol
		}
	}
	previous, exists := uses[use]
	if exists {
		if previous.symbol != symbol {
			uses[use] = useFact{ambiguous: true}
		}
		return nil
	}
	if len(uses) == maxOccurrences {
		return fmt.Errorf("SCIP occurrence limit exceeded")
	}
	uses[use] = useFact{symbol: symbol}
	return nil
}

func (doc *document) validSpan(start, end int) bool {
	return start >= 0 && end > start && end <= len(doc.source.Bytes)
}

func (doc *document) position(offset int) (int32, int32) {
	starts := doc.source.Module.SourceLineStarts
	line := sort.Search(len(starts), func(i int) bool { return starts[i] > offset }) - 1
	return int32(line), int32(offset - starts[line])
}

func (doc *document) add(span diagnostic.Span, symbol string, roles int32) {
	line, column := doc.position(span.StartByte)
	endLine, endColumn := doc.position(span.EndByte)
	occurrence := &scip.Occurrence{Symbol: symbol, SymbolRoles: roles}
	if line == endLine {
		occurrence.TypedRange = &scip.Occurrence_SingleLineRange{SingleLineRange: &scip.SingleLineRange{Line: line, StartCharacter: column, EndCharacter: endColumn}}
	} else {
		occurrence.TypedRange = &scip.Occurrence_MultiLineRange{MultiLineRange: &scip.MultiLineRange{
			StartLine: line, StartCharacter: column, EndLine: endLine, EndCharacter: endColumn,
		}}
	}
	doc.wire.Occurrences = append(doc.wire.Occurrences, occurrence)
}

func compareOccurrences(a, b *scip.Occurrence) int {
	pos := func(o *scip.Occurrence) [4]int32 {
		if r := o.GetSingleLineRange(); r != nil {
			return [4]int32{r.Line, r.StartCharacter, r.Line, r.EndCharacter}
		}
		r := o.GetMultiLineRange()
		return [4]int32{r.StartLine, r.StartCharacter, r.EndLine, r.EndCharacter}
	}
	aPos, bPos := pos(a), pos(b)
	if order := slices.Compare(aPos[:], bPos[:]); order != 0 {
		return order
	}
	return cmp.Compare(a.Symbol, b.Symbol)
}
