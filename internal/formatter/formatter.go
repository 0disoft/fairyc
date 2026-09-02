// Package formatter owns the first deterministic full-file editor formatting slice.
package formatter

import (
	"crypto/sha256"
	"encoding/binary"
	"errors"
	"fmt"
	"slices"

	"github.com/0disoft/fairyc/internal/frontend"
	"github.com/0disoft/fairyc/internal/lexer"
	"github.com/0disoft/fairyc/internal/source"
	"github.com/0disoft/fairyc/internal/syntax"
)

var ErrMalformedSource = errors.New("formatter input contains source diagnostics")

// GreenToken is an immutable token/trivia value detached from mutable parser arenas.
type GreenToken struct {
	Kind   lexer.Kind
	Trivia lexer.TriviaKind
	Text   string
	Cooked string
}

// GreenNode is an immutable syntax node view. Children returns a defensive copy.
type GreenNode struct {
	kind       syntax.Kind
	text       string
	cooked     string
	tokenStart int
	tokenEnd   int
	children   []uint32
}

func (node GreenNode) Kind() syntax.Kind { return node.kind }
func (node GreenNode) Text() string      { return node.text }
func (node GreenNode) Cooked() string    { return node.cooked }
func (node GreenNode) TokenRange() (int, int) {
	return node.tokenStart, node.tokenEnd
}
func (node GreenNode) Children() []uint32 { return slices.Clone(node.children) }

// GreenTree is a detached immutable snapshot suitable for editor-side identity.
// It deliberately owns no source.File, syntax.Tree, AST, or compiler authority.
type GreenTree struct {
	identity   [sha256.Size]byte
	shape      [sha256.Size]byte
	sourceHash [sha256.Size]byte
	root       uint32
	tokens     []GreenToken
	nodes      []GreenNode
}

func (tree GreenTree) Identity() [sha256.Size]byte       { return tree.identity }
func (tree GreenTree) ShapeIdentity() [sha256.Size]byte  { return tree.shape }
func (tree GreenTree) SourceIdentity() [sha256.Size]byte { return tree.sourceHash }
func (tree GreenTree) Root() uint32                      { return tree.root }
func (tree GreenTree) TokenCount() int                   { return len(tree.tokens) }
func (tree GreenTree) NodeCount() int                    { return len(tree.nodes) }
func (tree GreenTree) Tokens() []GreenToken              { return slices.Clone(tree.tokens) }
func (tree GreenTree) Node(index uint32) (GreenNode, bool) {
	if index >= uint32(len(tree.nodes)) {
		return GreenNode{}, false
	}
	node := tree.nodes[index]
	node.children = slices.Clone(node.children)
	return node, true
}

// Result contains a complete formatting decision. Malformed input is returned
// byte-for-byte unchanged rather than partially rewritten.
type Result struct {
	Bytes     []byte
	Tree      GreenTree
	Malformed bool
}

// Format performs a bounded full-file parse, formats only lexer-owned whitespace
// trivia, and verifies that the formatted file has the identical parser shape and
// non-trivia token sequence before returning rewritten bytes.
func Format(relativePath string, input []byte, limits source.Limits) (Result, error) {
	before, err := frontend.AnalyzeSyntax(source.NewFile(relativePath, input), limits)
	if err != nil {
		return Result{}, err
	}
	green, err := BuildGreen(before.Tree)
	if err != nil {
		return Result{}, err
	}
	if !before.Tree.ErrorFree() {
		return Result{Bytes: slices.Clone(input), Tree: green, Malformed: true}, ErrMalformedSource
	}

	formatted := formatTrivia(before.Tree)
	after, err := frontend.AnalyzeSyntax(source.NewFile(relativePath, formatted), limits)
	if err != nil {
		return Result{}, err
	}
	if !after.Tree.ErrorFree() {
		return Result{}, errors.New("formatter produced source diagnostics")
	}
	if syntaxShape(before.Tree) != syntaxShape(after.Tree) {
		return Result{}, errors.New("formatter changed parser shape")
	}
	if !equivalentSignificantTokens(before.Tree, after.Tree) {
		return Result{}, errors.New("formatter changed significant tokens")
	}
	formattedGreen, err := BuildGreen(after.Tree)
	if err != nil {
		return Result{}, err
	}
	return Result{Bytes: formatted, Tree: formattedGreen}, nil
}

// BuildGreen detaches one lossless syntax tree into immutable token/node values.
func BuildGreen(tree *syntax.Tree) (GreenTree, error) {
	if tree == nil {
		return GreenTree{}, errors.New("nil syntax tree")
	}
	if tree.Root < 0 || int(tree.Root) >= len(tree.Nodes) {
		return GreenTree{}, errors.New("syntax tree has invalid root")
	}
	green := GreenTree{
		sourceHash: sha256.Sum256(tree.Source),
		root:       uint32(tree.Root),
		tokens:     make([]GreenToken, 0, len(tree.Tokens)),
		nodes:      make([]GreenNode, len(tree.Nodes)),
	}
	for _, token := range tree.Tokens {
		if token.Kind == lexer.EOF {
			continue
		}
		green.tokens = append(green.tokens, GreenToken{
			Kind: token.Kind, Trivia: token.Trivia, Text: token.Text(tree.Source), Cooked: token.Cooked,
		})
	}
	for index, node := range tree.Nodes {
		children := make([]uint32, len(node.Children))
		for childIndex, child := range node.Children {
			if child < 0 || int(child) >= len(tree.Nodes) {
				return GreenTree{}, fmt.Errorf("syntax node %d has invalid child %d", index, child)
			}
			children[childIndex] = uint32(child)
		}
		green.nodes[index] = GreenNode{
			kind: node.Kind, text: node.Text, cooked: node.Cooked,
			tokenStart: node.TokenStart, tokenEnd: node.TokenEnd, children: children,
		}
	}
	green.shape = syntaxShape(tree)
	green.identity = greenIdentity(green)
	return green, nil
}

func formatTrivia(tree *syntax.Tree) []byte {
	result := make([]byte, 0, len(tree.Source)+1)
	last := len(tree.Tokens) - 1
	for last >= 0 && tree.Tokens[last].Kind == lexer.EOF {
		last--
	}
	for index, token := range tree.Tokens {
		if token.Kind == lexer.EOF {
			continue
		}
		text := []byte(token.Text(tree.Source))
		if token.Kind == lexer.Trivia && token.Trivia == lexer.TriviaWhitespace {
			formatted := canonicalWhitespace(text)
			if index == last {
				trimHorizontalSuffix(&formatted)
			}
			result = append(result, formatted...)
			continue
		}
		result = append(result, text...)
	}
	if len(result) > 0 && result[len(result)-1] != '\n' {
		result = append(result, '\n')
	}
	return result
}

func canonicalWhitespace(input []byte) []byte {
	output := make([]byte, 0, len(input))
	for index := 0; index < len(input); {
		if input[index] == '\r' && index+1 < len(input) && input[index+1] == '\n' {
			trimHorizontalSuffix(&output)
			output = append(output, '\n')
			index += 2
			continue
		}
		if input[index] == '\n' {
			trimHorizontalSuffix(&output)
			output = append(output, '\n')
			index++
			continue
		}
		output = append(output, input[index])
		index++
	}
	return output
}

func trimHorizontalSuffix(output *[]byte) {
	for len(*output) > 0 {
		last := (*output)[len(*output)-1]
		if last != ' ' && last != '\t' {
			return
		}
		*output = (*output)[:len(*output)-1]
	}
}

func equivalentSignificantTokens(left, right *syntax.Tree) bool {
	leftTokens := significantTokens(left)
	rightTokens := significantTokens(right)
	return slices.Equal(leftTokens, rightTokens)
}

func significantTokens(tree *syntax.Tree) []GreenToken {
	result := make([]GreenToken, 0, len(tree.Tokens))
	for _, token := range tree.Tokens {
		if token.Kind == lexer.EOF || token.Kind == lexer.Trivia {
			continue
		}
		result = append(result, GreenToken{Kind: token.Kind, Text: token.Text(tree.Source), Cooked: token.Cooked})
	}
	return result
}

func syntaxShape(tree *syntax.Tree) [sha256.Size]byte {
	hash := sha256.New()
	var scratch [8]byte
	binary.LittleEndian.PutUint64(scratch[:], uint64(tree.Root))
	_, _ = hash.Write(scratch[:])
	for _, node := range tree.Nodes {
		writeString(hash, string(node.Kind))
		writeString(hash, node.Text)
		writeString(hash, node.Cooked)
		binary.LittleEndian.PutUint64(scratch[:], uint64(len(node.Children)))
		_, _ = hash.Write(scratch[:])
		for _, child := range node.Children {
			binary.LittleEndian.PutUint64(scratch[:], uint64(child))
			_, _ = hash.Write(scratch[:])
		}
	}
	var result [sha256.Size]byte
	copy(result[:], hash.Sum(nil))
	return result
}

func greenIdentity(tree GreenTree) [sha256.Size]byte {
	hash := sha256.New()
	_, _ = hash.Write([]byte("fairyc-green-tree/v0\x00"))
	_, _ = hash.Write(tree.sourceHash[:])
	_, _ = hash.Write(tree.shape[:])
	for _, token := range tree.tokens {
		_, _ = hash.Write([]byte{byte(token.Kind), byte(token.Trivia)})
		writeString(hash, token.Text)
		writeString(hash, token.Cooked)
	}
	var result [sha256.Size]byte
	copy(result[:], hash.Sum(nil))
	return result
}

type byteWriter interface{ Write([]byte) (int, error) }

func writeString(writer byteWriter, value string) {
	var length [8]byte
	binary.LittleEndian.PutUint64(length[:], uint64(len(value)))
	_, _ = writer.Write(length[:])
	_, _ = writer.Write([]byte(value))
}
