package cbackend

import (
	"github.com/0disoft/fairyc/internal/hir"
	"github.com/0disoft/fairyc/internal/layout"
	"github.com/0disoft/fairyc/internal/mir"
	"github.com/0disoft/fairyc/internal/target"
)

// emissionAnalysis is frozen after analyzeEmission returns. Output methods may
// query its metadata but cannot invoke analysisBuilder capability checks.
type emissionAnalysis struct {
	profile            Profile
	reachableBlocks    [][]bool
	storageLayouts     map[hir.Type]ValueStorageLayout
	targetID           target.ID
	functions          []mir.Function
	bySymbol           map[hir.SymbolID]int
	nonReturning       map[hir.SymbolID]bool
	checkedFailures    checkedFailureProofs
	foreignFunctions   []mir.ForeignFunction
	foreignBySymbol    map[hir.SymbolID]int
	entryIndex         int
	arrayTypes         []arrayCType
	arrayByType        map[hir.Type]int
	structTypes        []structCType
	structByType       map[hir.Type]int
	enumTypes          []enumCType
	enumByType         map[hir.Type]int
	resourceTypes      []resourceCType
	resourceByType     map[hir.Type]int
	taggedTypes        []taggedCType
	taggedByType       map[hir.Type]int
	resourceLocals     []map[hir.SymbolID]mir.ResourceLocal
	features           map[string]struct{}
	usedForeignSymbols map[string]struct{}
	storageTransfers   bool
	trapCategories     map[string]struct{}
	dynamicTrapTargets map[int]map[mir.BlockID]struct{}
	localTypes         []map[mir.LocalID]hir.Type
	needsByteSlice     bool
	needsStructSlice   bool
	integerViews       map[hir.Type]bool
	needsI64Slice      bool
	needsMutI64Slice   bool
	needsMutByteSlice  bool
	needsStringView    bool
	needsBitcast       bool
	needsShift         bool
	i64Helpers         i64Helpers
	cScalars           map[hir.Type]target.Scalar
	instrumentation    string
	dropSites          []DropSite
	dropSiteByKey      map[dropSiteKey]uint32
	byteStringStorage  map[string]string
	stringStorage      map[string]string
	cStringStorage     map[string]string

	needsMutStructSlice bool
}

// analysisBuilder alone owns mutable requirement collection and layout caching.
type analysisBuilder struct {
	emissionAnalysis
	layoutEngine *layout.Engine
}
