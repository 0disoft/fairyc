package cdriver

import (
	"bytes"
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
	if len(header) == 0 || len(header) > 2*1024*1024 || len(literals) == 0 || len(literals) > 2*1024*1024 || len(units) == 0 || len(units) > 4096 || len(runtimeSource) == 0 || len(runtimeSource) > 8*1024*1024 {
		return WindowsHostArtifact{}, fmt.Errorf("invalid bounded project C inputs")
	}
	total := len(header) + len(literals) + len(runtimeSource)
	for _, unit := range units {
		if len(unit) == 0 || len(unit) > 2*1024*1024 {
			return WindowsHostArtifact{}, fmt.Errorf("invalid bounded project C body")
		}
		total += len(unit)
		if total > 96*1024*1024 {
			return WindowsHostArtifact{}, fmt.Errorf("project C transport exceeds 96 MiB")
		}
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
	objectPaths := make([]string, 0, len(sources))
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
		objects = append(objects, WindowsHostObject{Name: objectName, Bytes: contents})
		objectPaths = append(objectPaths, objectPath)
	}
	executable := filepath.Join(directory, "program.exe")
	arguments := append([]string{"/nologo", "/MT"}, objectPaths...)
	// Private bootstrap images must compare byte-for-byte across generations;
	// timestamp bytes are reproducible inputs, never removed by the comparator.
	arguments = append(arguments, "/Fe:"+executable, "/link", "/INCREMENTAL:NO", "/Brepro")
	if _, err := runWindowsHostCommandWithLimitAndTimeout(toolchain.Path, toolchain.Environment, directory, maxProcessOutput, timeout, arguments...); err != nil {
		return WindowsHostArtifact{}, fmt.Errorf("link project C: %s", strings.ReplaceAll(err.Error(), directory, "<temporary>"))
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

func inspectProjectObject(contents []byte) error {
	if _, err := pecoff.InspectObject(contents); err == nil {
		return nil
	}
	// Literal-only and include-only units need not have a nonempty code
	// section. Keep the existing code-object verifier unchanged, and admit
	// only bounded AMD64 COFF files with no nonempty code in this fallback.
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
