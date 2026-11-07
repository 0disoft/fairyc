package cdriver

import (
	"bytes"
	"context"
	"debug/pe"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/0disoft/fairyc/internal/pecoff"
)

// BuildWindowsMSVCTargetParts consumes already generated private C transport.
// The caller owns frontend and packet validation. Each C file is compiled under
// the existing command deadline, followed by one separately bounded link.
// The shared header is written once; no source concatenation or frontend fallback
// occurs here. Failed compilation/linking returns no publishable image.
func BuildWindowsMSVCTargetParts(toolchain WindowsHostToolchain, profile Profile, header, literals string, units []string, runtimeSource string) (WindowsHostArtifact, error) {
	return BuildWindowsMSVCTargetPartsWithTimeout(toolchain, profile, header, literals, units, runtimeSource, commandTimeout)
}

// BuildWindowsMSVCTargetPartsWithTimeout applies the deadline separately to
// each compiler invocation and the final link; it does not change program execution.
func BuildWindowsMSVCTargetPartsWithTimeout(toolchain WindowsHostToolchain, profile Profile, header, literals string, units []string, runtimeSource string, timeout time.Duration) (WindowsHostArtifact, error) {
	return BuildWindowsMSVCTargetPartsWithStackReserve(toolchain, profile, header, literals, units, runtimeSource, timeout, 0)
}

// ValidateWindowsProjectStackReserve bounds a private compiler-host reservation.
// Zero retains the native linker's default. Commit size is never overridden.
func ValidateWindowsProjectStackReserve(reserve uint64) error {
	if reserve != 0 && (reserve < 65536 || reserve > 16*1024*1024 || reserve%65536 != 0) {
		return fmt.Errorf("project stack reserve must be zero or 64 KiB aligned in [64 KiB,16 MiB]")
	}
	return nil
}

// BuildWindowsMSVCTargetPartsWithStackReserve is for explicitly sized private
// compiler images; ordinary project builds use the zero-reservation wrapper.
func BuildWindowsMSVCTargetPartsWithStackReserve(toolchain WindowsHostToolchain, profile Profile, header, literals string, units []string, runtimeSource string, timeout time.Duration, reserve uint64) (WindowsHostArtifact, error) {
	return buildWindowsMSVCTargetParts(toolchain, profile, header, literals, units, runtimeSource, timeout, reserve, true, false)
}

// BuildWindowsMSVCTargetPartsImageWithStackReserve builds the same private
// image as BuildWindowsMSVCTargetPartsWithStackReserve while inspecting each
// COFF object and releasing its bytes before the link. Publication callers that
// consume only Bytes keep one object resident at a time instead of the whole
// object set; the returned artifact has no Objects.
func BuildWindowsMSVCTargetPartsImageWithStackReserve(toolchain WindowsHostToolchain, profile Profile, header, literals string, units []string, runtimeSource string, timeout time.Duration, reserve uint64) (WindowsHostArtifact, error) {
	return buildWindowsMSVCTargetParts(toolchain, profile, header, literals, units, runtimeSource, timeout, reserve, false, false)
}

// BuildWindowsMSVCCompilerHostPartsWithStackReserve consumes verified private
// compiler-host C with a separately bounded 8MiB header. Ordinary target builders
// retain their 2MiB header limit. All other source, stack, timeout, COFF and image
// policies are unchanged. Each object is inspected and released before linking;
// the result contains the image without retaining the complete object set.
// Compiler-host partitions use at most two concurrent CL commands; the runtime
// remains serial and ordinary target project builds retain serial compilation.
func BuildWindowsMSVCCompilerHostPartsWithStackReserve(toolchain WindowsHostToolchain, profile Profile, header, literals string, units []string, runtimeSource string, timeout time.Duration, reserve uint64) (WindowsHostArtifact, error) {
	return buildWindowsMSVCTargetParts(toolchain, profile, header, literals, units, runtimeSource, timeout, reserve, false, true)
}

// maxProjectRetainedObjectBytes bounds the aggregate COFF bytes kept in
// WindowsHostArtifact.Objects, independently of the source transport limit.
const maxProjectRetainedObjectBytes = 256 << 20

// retainProjectObject appends one already inspected COFF object while the
// aggregate retained size stays inside the budget.
func retainProjectObject(objects []WindowsHostObject, retained int, object WindowsHostObject) ([]WindowsHostObject, int, error) {
	if retained < 0 || retained > maxProjectRetainedObjectBytes || len(object.Bytes) > maxProjectRetainedObjectBytes-retained {
		return objects, retained, fmt.Errorf("retained project objects exceed %d MiB; use the image-only project build", maxProjectRetainedObjectBytes>>20)
	}
	return append(objects, object), retained + len(object.Bytes), nil
}

func buildWindowsMSVCTargetParts(toolchain WindowsHostToolchain, profile Profile, header, literals string, units []string, runtimeSource string, timeout time.Duration, reserve uint64, retainObjects, compilerHost bool) (WindowsHostArtifact, error) {
	if err := ValidateWindowsProjectStackReserve(reserve); err != nil {
		return WindowsHostArtifact{}, err
	}
	if timeout <= 0 || timeout > 10*time.Minute {
		return WindowsHostArtifact{}, fmt.Errorf("MSVC project compile timeout must be in (0,10m]")
	}
	if runtime.GOOS != "windows" || runtime.GOARCH != "amd64" {
		return WindowsHostArtifact{}, fmt.Errorf("MSVC project build requires windows/amd64")
	}
	if toolchain.Name != "msvc" || toolchain.Path == "" || toolchain.Version == "" || len(toolchain.Environment) == 0 {
		return WindowsHostArtifact{}, fmt.Errorf("invalid resolved MSVC project toolchain")
	}
	if profile != LowProfile && profile != OptimizedProfile {
		return WindowsHostArtifact{}, fmt.Errorf("unsupported project C profile %q", profile.Name)
	}
	if err := validateWindowsProjectInputs(header, literals, units, runtimeSource, compilerHost); err != nil {
		return WindowsHostArtifact{}, err
	}
	directory, err := os.MkdirTemp("", "fairyc-msvc-project-")
	if err != nil {
		return WindowsHostArtifact{}, fmt.Errorf("create project build directory: %w", err)
	}
	defer os.RemoveAll(directory)
	if err := os.WriteFile(filepath.Join(directory, "fy_project.h"), []byte(header), 0o600); err != nil {
		return WindowsHostArtifact{}, fmt.Errorf("write project C header: %w", err)
	}
	sources := []windowsMSVCSource{{Name: "fy_literals.c", Contents: literals}}
	for index, unit := range units {
		sources = append(sources, windowsMSVCSource{Name: fmt.Sprintf("fy_unit_%d.c", index), Contents: unit})
	}
	sources = append(sources, windowsMSVCSource{Name: "fy_runtime.c", Contents: runtimeSource})
	optimization := "/Od"
	if profile == OptimizedProfile {
		optimization = "/O2"
	}
	objects := make([]WindowsHostObject, 0, len(sources))
	objectNames := make([]string, 0, len(sources))
	retainedBytes := 0
	if compilerHost {
		objectNames, err = compileWindowsCompilerHostProject(context.Background(), toolchain, directory, sources, optimization, timeout,
			windowsProjectCompileWorkers(min(runtime.GOMAXPROCS(0), runtime.NumCPU())))
		if err != nil {
			return WindowsHostArtifact{}, err
		}
	} else {
		for _, source := range sources {
			sourcePath := filepath.Join(directory, source.Name)
			if err := os.WriteFile(sourcePath, []byte(source.Contents), 0o600); err != nil {
				return WindowsHostArtifact{}, fmt.Errorf("write project source %s: %w", source.Name, err)
			}
			objectName := strings.TrimSuffix(source.Name, ".c") + ".obj"
			objectPath := filepath.Join(directory, objectName)
			// Candidate partitions retain their existing consumer flags. Stable
			// single-unit target builders keep their separate /W4 /WX policy.
			if _, err := runWindowsHostCommandWithLimitAndTimeout(toolchain.Path, toolchain.Environment, directory, maxProcessOutput, timeout,
				"/nologo", "/std:c11", "/TC", "/MT", "/Brepro", optimization, "/c", sourcePath, "/Fo:"+objectPath); err != nil {
				return WindowsHostArtifact{}, fmt.Errorf("compile project source %s: %s", source.Name, strings.ReplaceAll(err.Error(), directory, "<temporary>"))
			}
			contents, err := readArtifact(objectPath)
			if err != nil {
				return WindowsHostArtifact{}, fmt.Errorf("read project object %s: %w", objectName, err)
			}
			if err := inspectProjectObject(contents); err != nil {
				return WindowsHostArtifact{}, fmt.Errorf("inspect project object %s: %w", objectName, err)
			}
			// The link consumes the files on disk, so an image-only build can let
			// each inspected object drop before the next compile starts.
			if retainObjects {
				objects, retainedBytes, err = retainProjectObject(objects, retainedBytes, WindowsHostObject{Name: objectName, Bytes: contents})
				if err != nil {
					return WindowsHostArtifact{}, fmt.Errorf("retain project object %s: %w", objectName, err)
				}
			}
			objectNames = append(objectNames, objectName)
		}
	}
	executable := filepath.Join(directory, "program.exe")
	if err := linkWindowsMSVCProject(toolchain, directory, objectNames, timeout, reserve); err != nil {
		return WindowsHostArtifact{}, err
	}
	contents, err := readArtifact(executable)
	if err != nil {
		return WindowsHostArtifact{}, fmt.Errorf("read project image: %w", err)
	}
	if _, err := pecoff.InspectExecutable(contents); err != nil {
		return WindowsHostArtifact{}, fmt.Errorf("inspect project image: %w", err)
	}
	return WindowsHostArtifact{Bytes: contents, Objects: objects, Toolchain: toolchain, Profile: profile, ConsumerMode: C11ConsumerMode}, nil
}

func linkWindowsMSVCProject(toolchain WindowsHostToolchain, directory string, objectNames []string, timeout time.Duration, reserve uint64) error {
	// Names are fixed ASCII basenames generated by the project builder. Relative
	// response-file entries avoid quoting and CreateProcess command-line growth.
	if err := os.WriteFile(filepath.Join(directory, "fy_objects.rsp"), []byte(strings.Join(objectNames, "\n")+"\n"), 0o600); err != nil {
		return fmt.Errorf("write project object response: %w", err)
	}
	arguments := []string{"/nologo", "/MT", "@fy_objects.rsp"}
	// Private bootstrap images must compare byte-for-byte across generations;
	// timestamp bytes are reproducible inputs, never removed by the comparator.
	arguments = append(arguments, "/Fe:program.exe", "/link", "/INCREMENTAL:NO", "/Brepro")
	if reserve != 0 {
		arguments = append(arguments, fmt.Sprintf("/STACK:%d", reserve))
	}
	if _, err := runWindowsHostCommandWithLimitAndTimeout(toolchain.Path, toolchain.Environment, directory, maxProcessOutput, timeout, arguments...); err != nil {
		return fmt.Errorf("link project C: %s", strings.ReplaceAll(err.Error(), directory, "<temporary>"))
	}
	return nil
}

func inspectProjectObject(contents []byte) error {
	if _, err := pecoff.InspectObject(contents); err == nil {
		return nil
	}
	// Literal-only and include-only units need not have a nonempty code
	// section. Keep the existing code-object verifier unchanged, and admit
	// only bounded AMD64 COFF files with no nonempty code in this fallback.
	if err := pecoff.VerifyObjectParserBudget(contents); err != nil {
		return err
	}
	object, err := pe.NewFile(bytes.NewReader(contents))
	if err != nil {
		return err
	}
	defer object.Close()
	if object.Machine != pe.IMAGE_FILE_MACHINE_AMD64 || object.OptionalHeader != nil || len(object.Sections) == 0 || len(object.Sections) > 96 {
		return fmt.Errorf("invalid project data-only COFF header")
	}
	for _, section := range object.Sections {
		if section.Size != 0 && (section.Characteristics&0x20 != 0 || section.Characteristics&0x20000000 != 0) {
			return fmt.Errorf("project COFF code failed structural verification")
		}
		if section.Offset != 0 && uint64(section.Offset)+uint64(section.Size) > uint64(len(contents)) {
			return fmt.Errorf("project data-only COFF section exceeds object bytes")
		}
	}
	return nil
}
