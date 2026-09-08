package ast

import "github.com/0disoft/fairyc/internal/syntax"

// PrivateStorageLayout is produced only by private generic expansion, never by
// the parser. Type is the unevaluated query operand; Text selects size/alignment.
const PrivateStorageLayout syntax.Kind = "private-storage-layout"
