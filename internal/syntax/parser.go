package syntax

import (
	"strings"

	"github.com/0disoft/fairyc/internal/diagnostic"
	"github.com/0disoft/fairyc/internal/lexer"
	"github.com/0disoft/fairyc/internal/source"
)

type parseMode uint8

const (
	parseCurrent parseMode = iota
	parsePreviousRevision
)

// Parse performs UTF-8-aware lexing, lossless CST construction, recovery, and module validation
// for the current product grammar.
func Parse(file source.File, limits source.Limits) *Tree {
	return parse(file, limits, parseCurrent)
}

// ParsePreviousRevision accepts the immediately preceding source grammar for the standalone
// migration package. Product compilation must call Parse instead.
func ParsePreviousRevision(file source.File, limits source.Limits) *Tree {
	return parse(file, limits, parsePreviousRevision)
}

func parse(file source.File, limits source.Limits, mode parseMode) *Tree {
	budget := source.NewBudget(limits, len(file.Bytes))
	var lexed lexer.Result
	if mode == parsePreviousRevision {
		lexed = lexer.LexPreviousRevision(file, budget)
	} else {
		lexed = lexer.Lex(file, budget)
	}
	tree := &Tree{
		Source:      lexed.Source,
		Tokens:      lexed.Tokens,
		Root:        NoNode,
		Diagnostics: append([]diagnostic.Diagnostic(nil), lexed.Diagnostics...),
	}
	parser := newParser(file, tree, budget, mode)
	if !budget.Exhausted() {
		tree.Root = parser.parseSource()
	}
	if budget.Exhausted() {
		parser.emitExhausted()
	}
	tree.Diagnostics = sortAndDedupeDiagnostics(tree.Diagnostics)
	return tree
}

type parser struct {
	file          source.File
	tree          *Tree
	budget        *source.Budget
	mode          parseMode
	significant   []int
	position      int
	lastFullIndex int
	lastEndByte   int
	splitShift    bool
	hasExhausted  bool
}

func newParser(file source.File, tree *Tree, budget *source.Budget, mode parseMode) *parser {
	significant := make([]int, 0, len(tree.Tokens))
	for index, token := range tree.Tokens {
		if token.Kind != lexer.Trivia {
			significant = append(significant, index)
		}
	}
	return &parser{
		file:          file,
		tree:          tree,
		budget:        budget,
		mode:          mode,
		significant:   significant,
		lastFullIndex: -1,
		lastEndByte:   -1,
	}
}

func (p *parser) acceptsPreviousRevision() bool {
	return p.mode == parsePreviousRevision
}

func (p *parser) current() lexer.Token {
	if p.position >= len(p.significant) {
		position := len(p.tree.Source)
		return lexer.Token{Kind: lexer.EOF, Span: diagnostic.Span{StartByte: position, EndByte: position}}
	}
	token := p.tree.Tokens[p.significant[p.position]]
	if p.splitShift {
		token.Span.StartByte++
	}
	return token
}

func (p *parser) currentFullIndex() int {
	if p.position >= len(p.significant) {
		return len(p.tree.Tokens) - 1
	}
	return p.significant[p.position]
}

func (p *parser) currentText() string {
	return p.current().Text(p.tree.Source)
}

func (p *parser) peek(offset int) lexer.Token {
	position := p.position + offset
	if position < 0 || position >= len(p.significant) {
		end := len(p.tree.Source)
		return lexer.Token{Kind: lexer.EOF, Span: diagnostic.Span{StartByte: end, EndByte: end}}
	}
	return p.tree.Tokens[p.significant[position]]
}

func (p *parser) peekText(offset int) string {
	return p.peek(offset).Text(p.tree.Source)
}

func (p *parser) at(text string) bool {
	return p.currentText() == text
}

func (p *parser) atKind(kind lexer.Kind) bool {
	return p.current().Kind == kind
}

func (p *parser) advance() lexer.Token {
	token := p.current()
	if token.Kind != lexer.EOF {
		if !p.budget.ChargeWork(1) {
			return token
		}
		p.lastFullIndex = p.currentFullIndex()
		p.lastEndByte = token.Span.EndByte
		p.position++
		p.splitShift = false
	}
	return token
}

func (p *parser) match(text string) bool {
	if !p.at(text) {
		return false
	}
	p.advance()
	return true
}

func (p *parser) expect(text string) bool {
	if p.match(text) {
		return true
	}
	position := p.current().Span.StartByte
	p.emit("E0007", diagnostic.Span{StartByte: position, EndByte: position}, diagnostic.RepairInsert, text)
	return false
}

func (p *parser) expectTypeClose() bool {
	if p.match(">") {
		return true
	}
	if p.at(">>") {
		token := p.current()
		if token.Span.EndByte-token.Span.StartByte == 2 && p.budget.ChargeToken() && p.budget.ChargeWork(1) {
			p.lastFullIndex = p.currentFullIndex()
			p.lastEndByte = token.Span.StartByte + 1
			p.splitShift = true
			return true
		}
		p.emitExhausted()
		return false
	}
	position := p.current().Span.StartByte
	p.emit("E0007", diagnostic.Span{StartByte: position, EndByte: position}, diagnostic.RepairInsert, ">")
	return false
}

func (p *parser) expectStatementTerminator() bool {
	if p.match(";") {
		return true
	}
	position := p.current().Span.StartByte
	if p.lastEndByte >= 0 {
		position = p.lastEndByte
	}
	p.emit("E0007", diagnostic.Span{StartByte: position, EndByte: position}, diagnostic.RepairInsert, ";")
	return false
}

func (p *parser) emit(code string, span diagnostic.Span, repair diagnostic.Repair, token string) {
	if repair != diagnostic.RepairNone && !p.budget.ChargeRepair() {
		p.emitExhausted()
		return
	}
	if !p.budget.ChargeDiagnostic() {
		p.emitExhausted()
		return
	}
	p.tree.Diagnostics = append(p.tree.Diagnostics, diagnostic.Diagnostic{
		Code:   code,
		Span:   span,
		Repair: repair,
		Token:  token,
	})
}

func (p *parser) emitExhausted() {
	if p.hasExhausted {
		return
	}
	p.hasExhausted = true
	for _, existing := range p.tree.Diagnostics {
		if existing.Code == "E0010" {
			return
		}
	}
	p.tree.Diagnostics = append(p.tree.Diagnostics, source.ExhaustionDiagnostic(p.current().Span.StartByte))
}

func (p *parser) unexpected() NodeID {
	token := p.current()
	text := token.Text(p.tree.Source)
	start := p.currentFullIndex()
	p.emit("E0008", token.Span, diagnostic.RepairSkip, text)
	if token.Kind != lexer.EOF {
		p.advance()
	}
	return p.addNode(KindError, start, nil, text, "")
}

func (p *parser) addNode(kind Kind, startFull int, children []NodeID, text, cooked string) NodeID {
	if !p.budget.ChargeNode() {
		p.emitExhausted()
		return NoNode
	}
	startByte := p.current().Span.StartByte
	if startFull >= 0 && startFull < len(p.tree.Tokens) {
		startByte = p.tree.Tokens[startFull].Span.StartByte
	}
	endByte := startByte
	endFull := startFull
	if p.lastFullIndex >= startFull && p.lastFullIndex >= 0 && p.lastFullIndex < len(p.tree.Tokens) {
		endFull = p.lastFullIndex + 1
		endByte = p.lastEndByte
	}
	node := Node{
		Kind:       kind,
		Span:       diagnostic.Span{StartByte: startByte, EndByte: endByte},
		TokenStart: startFull,
		TokenEnd:   endFull,
		Text:       text,
		Cooked:     cooked,
		Children:   append([]NodeID(nil), children...),
	}
	p.tree.Nodes = append(p.tree.Nodes, node)
	return NodeID(len(p.tree.Nodes) - 1)
}

func (p *parser) addNamedNode(kind Kind, startFull int, children []NodeID, text, cooked string, nameSpan diagnostic.Span) NodeID {
	id := p.addNode(kind, startFull, children, text, cooked)
	if id != NoNode && int(id) < len(p.tree.Nodes) {
		p.tree.Nodes[id].NameSpan = nameSpan
		if supportsDeclarationDocumentation(kind) {
			if span, found := lexer.AttachedDocumentationSpan(p.tree.Tokens, p.tree.Source, startFull); found {
				p.tree.Nodes[id].DocumentationSpan = span
			}
		}
	}
	return id
}

func supportsDeclarationDocumentation(kind Kind) bool {
	switch kind {
	case KindStruct, KindResource, KindEnum, KindFunction, KindForeign:
		return true
	default:
		return false
	}
}

func (p *parser) enter() bool {
	if p.budget.Enter() {
		return true
	}
	p.emitExhausted()
	return false
}

func (p *parser) leave() {
	p.budget.Leave()
}

func (p *parser) pathNode() (NodeID, string, bool) {
	start := p.currentFullIndex()
	if !p.atKind(lexer.Identifier) {
		return p.unexpected(), "", false
	}
	parts := []string{p.currentText()}
	p.advance()
	for !p.budget.Exhausted() && p.match("::") {
		if !p.atKind(lexer.Identifier) {
			p.expect("identifier")
			return p.addNode(KindPath, start, nil, strings.Join(parts, "::"), ""), strings.Join(parts, "::"), false
		}
		parts = append(parts, p.currentText())
		p.advance()
	}
	value := strings.Join(parts, "::")
	return p.addNode(KindPath, start, nil, value, ""), value, true
}

func (p *parser) synchronize(boundaries ...string) {
	boundarySet := make(map[string]struct{}, len(boundaries))
	for _, boundary := range boundaries {
		boundarySet[boundary] = struct{}{}
	}
	for p.current().Kind != lexer.EOF && !p.budget.Exhausted() {
		if _, found := boundarySet[p.currentText()]; found {
			return
		}
		p.advance()
	}
}
