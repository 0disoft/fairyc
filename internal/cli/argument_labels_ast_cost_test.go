package cli

import (
	"reflect"
	"runtime"
	"testing"

	"github.com/0disoft/fairyc/internal/ast"
	"github.com/0disoft/fairyc/internal/workspaceloader"
)

// Count retained AST expressions in the same scanner workspace used by the
// earlier frontend-cost experiment. This does not measure TotalAlloc or speed.
func TestArgumentLabelsUnusedASTCost(t *testing.T) {
	root, _ := tokenIndexWorkspace(t, false)
	workspace, err := workspaceloader.Load(root)
	if err != nil {
		t.Fatal(err)
	}
	expressionType := reflect.TypeOf((*ast.Expression)(nil))
	var local, sdk, scanner, labeled, localFiles, sdkFiles int
	for _, pkg := range workspace.Packages {
		for _, file := range pkg.Sources {
			seen := map[uintptr]bool{}
			count, labels := countASTExpressions(reflect.ValueOf(file.Module), expressionType, seen)
			labeled += labels
			switch pkg.SourceKind {
			case workspaceloader.SourceKindLocal:
				localFiles++
				local += count
				if file.Path == "scanner.fy" {
					scanner = count
				}
			case workspaceloader.SourceKindSDK:
				sdkFiles++
				sdk += count
			default:
				t.Fatalf("unexpected source kind %q", pkg.SourceKind)
			}
		}
	}
	if scanner == 0 || local == 0 || sdk == 0 || labeled != 0 {
		t.Fatalf("incomplete unlabeled scanner AST count: scanner=%d local=%d sdk=%d labeled=%d", scanner, local, sdk, labeled)
	}
	current := reflect.TypeOf(ast.Expression{})
	fields := make([]reflect.StructField, 0, current.NumField()-1)
	for index := 0; index < current.NumField(); index++ {
		field := current.Field(index)
		if field.Name != "CallLabels" {
			fields = append(fields, field)
		}
	}
	if len(fields) != current.NumField()-1 {
		t.Fatal("CallLabels field missing")
	}
	withoutLabels := reflect.StructOf(fields)
	var stats runtime.MemStats
	runtime.ReadMemStats(&stats)
	allocationClass := func(size uintptr) uint32 {
		for _, class := range stats.BySize {
			if class.Size >= uint32(size) && class.Size != 0 {
				return class.Size
			}
		}
		return 0
	}
	currentClass := allocationClass(current.Size())
	withoutClass := allocationClass(withoutLabels.Size())
	if currentClass == 0 || withoutClass == 0 || currentClass < withoutClass {
		t.Fatalf("unexpected allocator classes: current=%d without=%d", currentClass, withoutClass)
	}
	unlabeled := local + sdk - labeled
	t.Logf("ARGUMENT_LABEL_UNUSED_AST local_files=%d sdk_files=%d scanner=%d local=%d sdk=%d labeled=%d unlabeled=%d struct_current=%d struct_without=%d class_current=%d class_without=%d reachable_unlabeled_class_delta=%d",
		localFiles, sdkFiles, scanner, local, sdk, labeled, unlabeled, current.Size(), withoutLabels.Size(), currentClass, withoutClass, unlabeled*int(currentClass-withoutClass))
}

func countASTExpressions(value reflect.Value, expressionType reflect.Type, seen map[uintptr]bool) (count, labeled int) {
	if !value.IsValid() {
		return 0, 0
	}
	switch value.Kind() {
	case reflect.Interface:
		if !value.IsNil() {
			return countASTExpressions(value.Elem(), expressionType, seen)
		}
	case reflect.Pointer:
		if value.IsNil() {
			return 0, 0
		}
		if value.Type() == expressionType {
			if seen[value.Pointer()] {
				return 0, 0
			}
			seen[value.Pointer()] = true
			count = 1
			if !value.Elem().FieldByName("CallLabels").IsNil() {
				labeled = 1
			}
		}
		more, extraLabels := countASTExpressions(value.Elem(), expressionType, seen)
		return count + more, labeled + extraLabels
	case reflect.Struct:
		for index := 0; index < value.NumField(); index++ {
			more, extraLabels := countASTExpressions(value.Field(index), expressionType, seen)
			count += more
			labeled += extraLabels
		}
	case reflect.Slice, reflect.Array:
		for index := 0; index < value.Len(); index++ {
			more, extraLabels := countASTExpressions(value.Index(index), expressionType, seen)
			count += more
			labeled += extraLabels
		}
	case reflect.Map:
		for iter := value.MapRange(); iter.Next(); {
			more, extraLabels := countASTExpressions(iter.Value(), expressionType, seen)
			count += more
			labeled += extraLabels
		}
	}
	return count, labeled
}
