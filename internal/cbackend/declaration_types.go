package cbackend

import (
	"github.com/0disoft/fairyc/internal/layout"
	"github.com/0disoft/fairyc/internal/mir"
)

type structCType struct {
	definition mir.StructType
	layout     layout.Result
	layoutType layout.Type
	fieldIndex map[string]int
}

type resourceCType struct {
	definition mir.ResourceType
	layout     layout.Result
	layoutType layout.Type
	fieldIndex map[string]int
	feature    string
}

type enumCType struct {
	definition   mir.EnumType
	layout       layout.Result
	layoutType   layout.Type
	variantIndex map[string]int
	fieldIndex   []map[string]int
}
