package mir

import "github.com/0disoft/fairyc/internal/hir"

func signedIntegerType(typ hir.Type) bool {
	return typ == hir.TypeI32 || typ == hir.TypeI64
}
