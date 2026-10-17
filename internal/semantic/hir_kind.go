package semantic

import (
	"github.com/0disoft/fairyc/internal/ast"
	"github.com/0disoft/fairyc/internal/hir"
	"github.com/0disoft/fairyc/internal/syntax"
)

func expressionKind(kind syntax.Kind) hir.ExpressionKind {
	switch kind {
	case syntax.KindBlock:
		return hir.ExpressionBlock
	case syntax.KindWhile:
		return hir.ExpressionWhile
	case syntax.KindRegion:
		return hir.ExpressionRegion
	case syntax.KindPath:
		return hir.ExpressionPath
	case syntax.KindLiteral:
		return hir.ExpressionLiteral
	case syntax.KindGroup:
		return hir.ExpressionGroup
	case syntax.KindPrefix:
		return hir.ExpressionPrefix
	case syntax.KindBinary:
		return hir.ExpressionBinary
	case syntax.KindCast:
		return hir.ExpressionCast
	case syntax.KindCall:
		return hir.ExpressionCall
	case syntax.KindOmitBound:
		return hir.ExpressionOmittedRangeBound
	case syntax.KindFieldExpr:
		return hir.ExpressionFieldExpression
	case syntax.KindPropagate:
		return hir.ExpressionPropagate
	case syntax.KindArray:
		return hir.ExpressionArray
	case syntax.KindRecord:
		return hir.ExpressionRecord
	case syntax.KindIf:
		return hir.ExpressionIf
	case syntax.KindMatch:
		return hir.ExpressionMatch
	case syntax.KindUnsafe:
		return hir.ExpressionUnsafe
	case ast.PrivateSharedReborrow:
		return hir.ExpressionSharedReborrow
	case ast.PrivateStorageLayout:
		return hir.ExpressionStorageLayout
	case ast.PrivateStorageTake:
		return hir.ExpressionStorageTake
	case ast.PrivateStoragePut:
		return hir.ExpressionStoragePut
	case ast.PrivateStorageBorrow:
		return hir.ExpressionStorageBorrow
	case ast.PrivateStorageElementAddress:
		return hir.ExpressionStorageElementAddress
	default:
		return hir.ExpressionInvalid
	}
}
func statementKind(kind syntax.Kind) hir.StatementKind {
	switch kind {
	case syntax.KindBinding:
		return hir.StatementBinding
	case syntax.KindAssignment:
		return hir.StatementAssignment
	case syntax.KindAssert:
		return hir.StatementAssert
	case syntax.KindDefer:
		return hir.StatementDefer
	case syntax.KindReturn:
		return hir.StatementReturn
	case syntax.KindDiscard:
		return hir.StatementDiscard
	case syntax.KindBurn:
		return hir.StatementBurn
	case syntax.KindWhile:
		return hir.StatementWhile
	case syntax.KindFor:
		return hir.StatementFor
	case syntax.KindBreak:
		return hir.StatementBreak
	case syntax.KindContinue:
		return hir.StatementContinue
	case syntax.KindRegion:
		return hir.StatementRegion
	case syntax.KindExprStmt:
		return hir.StatementExpressionStatement
	default:
		return hir.StatementInvalid
	}
}
