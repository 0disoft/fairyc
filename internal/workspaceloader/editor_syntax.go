package workspaceloader

import (
	"bytes"

	"github.com/0disoft/fairyc/internal/ast"
	"github.com/0disoft/fairyc/internal/frontend"
	"github.com/0disoft/fairyc/internal/source"
	"github.com/0disoft/fairyc/internal/syntax"
)

const maxEditorSyntaxWeight = 8 << 20

type editorSyntaxKey struct{ absolute, relative string }

type editorSyntaxEntry struct {
	tree        *syntax.Tree
	checkpoints *syntax.EditorParseSnapshot
	weight      int64
}

// EditorSyntaxSnapshot retains private, lossless trees for one workspace
// generation. Semantic checking always receives newly lowered ASTs.
type EditorSyntaxSnapshot struct {
	root   string
	trees  map[editorSyntaxKey]editorSyntaxEntry
	weight int64
}

// Weight estimates the retained tree storage for the editor's admission budget.
func (snapshot *EditorSyntaxSnapshot) Weight() int64 { return snapshot.weight }

type editorSyntaxReuse struct {
	previous *EditorSyntaxSnapshot
	next     EditorSyntaxSnapshot
}

func (reuse *editorSyntaxReuse) analyze(absolute, relative string, contents []byte) (frontend.SyntaxResult, error) {
	if reuse == nil {
		return frontend.AnalyzeSyntax(source.NewFile(relative, contents), source.DefaultLimits())
	}
	key := editorSyntaxKey{absolute, relative}
	var entry editorSyntaxEntry
	if reuse.previous != nil && reuse.previous.root == reuse.next.root {
		entry = reuse.previous.trees[key]
		if entry.tree != nil && bytes.Equal(entry.tree.Source, contents) {
			module, err := ast.Lower(entry.tree)
			if err == nil {
				reuse.retain(key, entry)
				return frontend.SyntaxResult{Tree: entry.tree, AST: module}, nil
			}
		}
	}
	result, checkpoints, err := frontend.AnalyzeEditorSyntax(source.NewFile(relative, contents), source.DefaultLimits(), entry.checkpoints)
	if err == nil && result.AST != nil {
		checkpointWeight := checkpoints.CheckpointWeight()
		remaining := int64(maxEditorSyntaxWeight) - reuse.next.weight - checkpointWeight
		weight := editorTreeWeight(key, result.Tree, remaining) + checkpointWeight
		reuse.retain(key, editorSyntaxEntry{tree: result.Tree, checkpoints: checkpoints, weight: weight})
	}
	return result, err
}

func editorTreeWeight(key editorSyntaxKey, tree *syntax.Tree, remaining int64) int64 {
	weight := int64(len(tree.Source) + len(key.absolute) + len(key.relative) + 128)
	weight += int64(cap(tree.Tokens))*64 + int64(cap(tree.Nodes))*160
	if weight > remaining {
		return weight
	}
	for _, token := range tree.Tokens {
		weight += int64(len(token.Cooked))
	}
	for _, node := range tree.Nodes {
		weight += int64(len(node.Text) + len(node.Cooked) + 8*cap(node.Children))
	}
	return weight
}

func (reuse *editorSyntaxReuse) retain(key editorSyntaxKey, entry editorSyntaxEntry) {
	if entry.weight <= maxEditorSyntaxWeight-reuse.next.weight && reuse.next.trees[key].tree == nil {
		reuse.next.trees[key] = entry
		reuse.next.weight += entry.weight
	}
}
