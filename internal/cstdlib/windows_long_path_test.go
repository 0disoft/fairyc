package cstdlib

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/0disoft/fairyc/internal/cdriver"
)

// Exercise real file access and enumeration beyond MAX_PATH, not only conversion.
func TestWindowsLongPathNative(t *testing.T) {
	if runtime.GOOS != "windows" || runtime.GOARCH != "amd64" {
		t.Skip("requires Windows AMD64 MSVC")
	}
	root := t.TempDir()
	relative := filepath.Join(strings.Repeat("a", 130), strings.Repeat("b", 130))
	directory := filepath.Join(root, relative)
	if err := os.MkdirAll(directory, 0700); err != nil {
		t.Fatal(err)
	}
	implementation, err := WindowsMSVCSourceForSymbols([]string{
		"fy_std_file_open_utf8_v0", "fy_std_windows_read_dir_open_v0",
	})
	if err != nil {
		t.Fatal(err)
	}
	junctionProbe, err := os.ReadFile("stage0_windows_file_remove_tree_probe.c")
	if err != nil {
		t.Fatal(err)
	}
	junctionHelper, _, found := strings.Cut(string(junctionProbe), "int fy_std_windows_file_remove_tree_probe_v0(void)")
	if !found {
		t.Fatal("missing junction probe helper")
	}
	toolchain, err := cdriver.ResolveWindowsHostMSVC()
	if err != nil {
		t.Fatal(err)
	}
	for _, profile := range []cdriver.Profile{cdriver.LowProfile, cdriver.OptimizedProfile} {
		t.Run(profile.Name, func(t *testing.T) {
			artifact, err := cdriver.BuildWindowsMSVCProbe(toolchain, profile, implementation+junctionHelper+longPathProbe)
			if err != nil {
				t.Fatal(err)
			}
			outcome, err := cdriver.ExecuteWindowsMSVCTargetWithInputAndArgs(artifact, strings.NewReader(""), []string{root, relative})
			if err != nil || outcome.ExitCode != 0 || outcome.Stdout != "" || outcome.Stderr != "" {
				t.Fatalf("long path probe: %+v %v", outcome, err)
			}
			data, err := os.ReadFile(filepath.Join(directory, "한글.txt"))
			if err != nil || string(data) != "long path payload" {
				t.Fatalf("written file = %q, %v", data, err)
			}
			t.Logf("native long path bytes=%d, executable bytes=%d", len(directory)+len("/한글.txt"), len(artifact.Bytes))
		})
	}
}

const longPathProbe = `
#include <stdio.h>
#define REQUIRE(test) do { if (!(test)) return __LINE__; } while (0)
int main(int argc, char **argv) {
    REQUIRE(argc == 3);
    REQUIRE(SetCurrentDirectoryA(argv[1]));
    char path[4096];
    const char name[] = "\xed\x95\x9c\xea\xb8\x80.txt";
    REQUIRE(snprintf(path, sizeof path, "%s/./%s", argv[2], name) > 260);
    for (size_t i = 0; path[i]; ++i) { if (path[i] == '\\') path[i] = '/'; }
    struct fy_std_windows_file_v0 file = {NULL, 0};
    struct fy_std_windows_open_options_v0 options = {0, 1, 1, 1, 0, 0};
    struct fy_std_windows_file_result_v0 result = fy_std_file_open_utf8_v0(
        (const uint8_t *)path, strlen(path), &options, &file);
    REQUIRE(result.status == 0);
    const uint8_t payload[] = "long path payload";
    REQUIRE(fy_std_windows_file_write_all_v0(&file, payload, sizeof payload - 1).status == 0);
    REQUIRE(fy_std_windows_file_close_v0(&file).status == 0);
    REQUIRE(snprintf(path, sizeof path, "%s/%s/../%s", argv[1], argv[2], name) > 260);
    /* Parent normalization resolves to a different, absent file. */
    options = (struct fy_std_windows_open_options_v0){1, 0, 0, 0, 0, 0};
    REQUIRE(fy_std_file_open_utf8_v0((const uint8_t *)path, strlen(path), &options, &file).status == 1);
    REQUIRE(!file.open && file.handle == NULL);
    REQUIRE(snprintf(path, sizeof path, "%s/%s/./%s", argv[1], argv[2], name) > 260);
    REQUIRE(fy_std_file_open_utf8_v0((const uint8_t *)path, strlen(path), &options, &file).status == 0);
    uint8_t read[sizeof payload] = {0};
    result = fy_std_windows_file_read_v0(&file, read, sizeof read);
    REQUIRE(result.status == 0 && result.count == sizeof payload - 1 && memcmp(payload, read, sizeof payload - 1) == 0);
    REQUIRE(fy_std_windows_file_close_v0(&file).status == 0);
    REQUIRE(snprintf(path, sizeof path, "%s/./", argv[2]) > 260);
    struct fy_std_directory_v0 dir = {NULL, 0};
    REQUIRE(fy_std_windows_read_dir_open_v0((const uint8_t *)path, strlen(path), &dir).status == 0);
    uint8_t entry[128];
    struct fy_std_directory_entry_result_v0 item = fy_std_windows_read_dir_next_v0(&dir, entry, sizeof entry);
    REQUIRE(item.status == 0 && item.name_length == strlen(name) && memcmp(entry, name, strlen(name)) == 0);
    REQUIRE(fy_std_windows_read_dir_next_v0(&dir, entry, sizeof entry).name_length == 0);
    REQUIRE(fy_std_windows_read_dir_close_v0(&dir).status == 0);
    REQUIRE(fy_std_windows_create_junction_probe_v0(L"link", L"."));
    REQUIRE(snprintf(path, sizeof path, "link/%s", argv[2]) > 260);
    struct fy_std_directory_result_v0 linked = fy_std_windows_read_dir_open_v0((const uint8_t *)path, strlen(path), &dir);
    REQUIRE(RemoveDirectoryW(L"link"));
    REQUIRE(linked.status == 1 && linked.os_code == ERROR_CANT_ACCESS_FILE && !dir.open && dir.state == NULL);
    REQUIRE(fy_std_windows_read_dir_open_v0((const uint8_t *)"\\\\?\\C:\\", 7, &dir).status == 2);
    REQUIRE(fy_std_windows_read_dir_open_v0((const uint8_t *)"\\\\.\\C:\\", 7, &dir).status == 2);
    REQUIRE(fy_std_windows_read_dir_open_v0((const uint8_t *)"a*", 2, &dir).status == 2);
    REQUIRE(fy_std_windows_read_dir_open_v0((const uint8_t *)"a\0b", 3, &dir).status == 2);
    REQUIRE(fy_std_windows_read_dir_open_v0((const uint8_t *)"\xff", 1, &dir).status == 5);
    WCHAR wide[4096] = L"\\\\server\\share\\";
    size_t n = (size_t)lstrlenW(wide);
    for (size_t i = 0; i < 300; ++i) { wide[n+i] = L'x'; } wide[n+300] = 0;
    REQUIRE(fy_std_windows_long_path_v0(wide, 4096) == ERROR_SUCCESS);
    REQUIRE(wcsncmp(wide, L"\\\\?\\UNC\\server\\share\\", 21) == 0);
    wide[0] = L'C'; wide[1] = L':'; wide[2] = L'\\';
    for (size_t i = 3; i < 4094; ++i) { wide[i] = L'x'; } wide[4094] = 0;
    REQUIRE(fy_std_windows_long_path_v0(wide, 4096) == ERROR_FILENAME_EXCED_RANGE);
    return 0;
}
`
