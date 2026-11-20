package cli

import (
	"crypto/sha256"
	"encoding/binary"

	"github.com/0disoft/fairyc/internal/compiler"
	"github.com/0disoft/fairyc/internal/hir"
	"github.com/0disoft/fairyc/internal/lexer"
	"github.com/0disoft/fairyc/internal/source"
	"github.com/0disoft/fairyc/internal/workspaceloader"
)

// Use the same immutable source snapshot from which compiler provenance was built.
func atlasTemplateSyntax(workspace workspaceloader.Workspace, owner hir.PackageID, provenance compiler.GenericProvenance) map[string][sha256.Size]byte {
	files := make(map[string][]byte)
	result := make(map[string][sha256.Size]byte)
	for _, declaration := range provenance.Declarations() {
		body := declaration.BodySpan
		if declaration.Package != owner || body.EndByte <= body.StartByte {
			continue
		}
		data, known := files[body.Source]
		if !known {
			_, data, _ = workspace.DiagnosticSource(owner, body.Source)
			files[body.Source] = data
		}
		if body.StartByte < 0 || body.EndByte > len(data) {
			continue
		}
		if digest, ok := atlasTemplateBodyDigest(data[body.StartByte:body.EndByte]); ok {
			result[declaration.Module+"::"+declaration.Name] = digest
		}
	}
	return result
}

func atlasTemplateBodyDigest(body []byte) ([sha256.Size]byte, bool) {
	budget := source.NewBudget(source.DefaultLimits(), len(body))
	lexed := lexer.Lex(source.File{Bytes: body}, budget)
	if budget.Exhausted() || len(lexed.Diagnostics) != 0 {
		return [sha256.Size]byte{}, false
	}
	digest := sha256.New()
	var header [5]byte
	for _, token := range lexed.Tokens {
		if token.Kind == lexer.Trivia || token.Kind == lexer.EOF {
			continue
		}
		spelling := lexed.Source[token.Span.StartByte:token.Span.EndByte]
		header[0] = byte(token.Kind)
		binary.LittleEndian.PutUint32(header[1:], uint32(len(spelling)))
		digest.Write(header[:])
		digest.Write(spelling)
	}
	var hash [sha256.Size]byte
	digest.Sum(hash[:0])
	return hash, true
}
