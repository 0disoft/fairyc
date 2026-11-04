package cdriver

import (
	"bytes"
	"debug/pe"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/0disoft/fairyc/internal/cruntime"
)

func TestProjectRetainedObjectBudget(t *testing.T) {
	first := WindowsHostObject{Name: "fy_literals.obj", Bytes: []byte{1, 2}}
	objects, retained, err := retainProjectObject(nil, 0, first)
	if err != nil || len(objects) != 1 || retained != len(first.Bytes) || objects[0].Name != first.Name {
		t.Fatalf("first retained object: %d objects, %d bytes, %v", len(objects), retained, err)
	}
	// The documented budget is inclusive, so exactly the limit still retains.
	objects, retained, err = retainProjectObject(objects, maxProjectRetainedObjectBytes-2, WindowsHostObject{Name: "fy_runtime.obj", Bytes: []byte{3, 4}})
	if err != nil || len(objects) != 2 || retained != maxProjectRetainedObjectBytes {
		t.Fatalf("budget boundary: %d objects, %d bytes, %v", len(objects), retained, err)
	}
	// One byte over the limit is rejected without changing the retained set.
	rejected, retained, err := retainProjectObject(objects, maxProjectRetainedObjectBytes-1, WindowsHostObject{Name: "fy_unit_0.obj", Bytes: []byte{5, 6}})
	if err == nil || !strings.Contains(err.Error(), "image-only") || len(rejected) != 2 || rejected[0].Name != first.Name || retained != maxProjectRetainedObjectBytes-1 {
		t.Fatalf("over-budget object: %d objects, %d bytes, %v", len(rejected), retained, err)
	}
}

func TestWindowsProjectImageOnlyBuildMatchesRetained(t *testing.T) {
	if runtime.GOOS != "windows" || runtime.GOARCH != "amd64" {
		t.Skip("requires Windows x64 MSVC")
	}
	toolchain, err := ResolveWindowsHostMSVC()
	if err != nil {
		t.Skipf("MSVC unavailable: %v", err)
	}
	scalar, err := cruntime.ScalarSource(100)
	if err != nil {
		t.Fatal(err)
	}
	header := "#ifndef FY_IMAGE_TEST\n#define FY_IMAGE_TEST\n#include <stdint.h>\nextern const uint8_t payload[];\nint32_t other(void);\n#endif\n"
	literals := "#include \"fy_project.h\"\nconst uint8_t payload[]={42};\n"
	units := []string{
		"#include \"fy_project.h\"\nint main(void){return (int)other()-42;}\n",
		"#include \"fy_project.h\"\nint32_t other(void){return payload[0];}\n",
	}
	retained, err := BuildWindowsMSVCTargetPartsWithStackReserve(toolchain, LowProfile, header, literals, units, scalar, time.Minute, 0)
	if err != nil {
		t.Fatal(err)
	}
	names := []string{"fy_literals.obj", "fy_unit_0.obj", "fy_unit_1.obj", "fy_runtime.obj"}
	if len(retained.Objects) != len(names) {
		t.Fatalf("retaining build kept %d objects, want %d", len(retained.Objects), len(names))
	}
	for index, object := range retained.Objects {
		if object.Name != names[index] || len(object.Bytes) == 0 {
			t.Fatalf("retained object %d = %q with %d bytes", index, object.Name, len(object.Bytes))
		}
		if err := inspectProjectObject(object.Bytes); err != nil {
			t.Fatalf("retained object %s: %v", object.Name, err)
		}
	}
	imageOnly, err := BuildWindowsMSVCTargetPartsImageWithStackReserve(toolchain, LowProfile, header, literals, units, scalar, time.Minute, 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(imageOnly.Objects) != 0 {
		t.Fatalf("image-only build retained %d objects", len(imageOnly.Objects))
	}
	// /Brepro makes the private image reproducible across fresh directories,
	// so dropping objects must not change the linked result.
	if !bytes.Equal(retained.Bytes, imageOnly.Bytes) {
		t.Fatalf("image-only build produced %d bytes, retaining build produced %d", len(imageOnly.Bytes), len(retained.Bytes))
	}
	outcome, err := ExecuteWindowsMSVCTarget(imageOnly)
	if err != nil || outcome.ExitCode != 0 || outcome.Stdout != "" || outcome.Stderr != "" {
		t.Fatalf("image-only result: %+v, %v", outcome, err)
	}
	reserved, err := BuildWindowsMSVCTargetPartsImageWithStackReserve(toolchain, LowProfile, header, literals, units, scalar, time.Minute, 16*1024*1024)
	if err != nil {
		t.Fatal(err)
	}
	if len(reserved.Objects) != 0 {
		t.Fatalf("reserved image-only build retained %d objects", len(reserved.Objects))
	}
	file, err := pe.NewFile(bytes.NewReader(reserved.Bytes))
	if err != nil {
		t.Fatal(err)
	}
	optional, ok := file.OptionalHeader.(*pe.OptionalHeader64)
	if !ok || optional.SizeOfStackReserve != 16*1024*1024 {
		file.Close()
		t.Fatal("reserved image-only build lost its stack reservation")
	}
	file.Close()
	outcome, err = ExecuteWindowsMSVCTarget(reserved)
	if err != nil || outcome.ExitCode != 0 || outcome.Stdout != "" || outcome.Stderr != "" {
		t.Fatalf("reserved image-only result: %+v, %v", outcome, err)
	}
	t.Logf("retained objects=%d image=%d; image-only=%d", len(retained.Objects), len(retained.Bytes), len(imageOnly.Bytes))
}
