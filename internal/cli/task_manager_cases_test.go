package cli

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/0disoft/fairyc/internal/testprocess"
)

func taskManagerCases(t *testing.T, exe, root string) {
	t.Helper()
	write := func(path, data string) {
		t.Helper()
		if err := os.WriteFile(filepath.Join(root, path), []byte(data), 0600); err != nil {
			t.Fatal(err)
		}
	}
	read := func(path string) []byte {
		t.Helper()
		data, err := os.ReadFile(filepath.Join(root, path))
		if os.IsNotExist(err) {
			return nil
		}
		if err != nil {
			t.Fatal(err)
		}
		return data
	}
	executions := 0
	defer func() { t.Logf("%d exact-output executions with file/sidecar assertions", executions) }()
	run := func(args []string, code int, out, stderr string, preserve ...string) {
		t.Helper()
		before := make(map[string][]byte)
		for _, path := range preserve {
			before[path] = read(path)
		}
		got, err := testprocess.Run(exe, root, args, 10*time.Second)
		executions++
		if err != nil || got != (testprocess.Observation{ExitCode: code, Stdout: out, Stderr: stderr}) {
			t.Fatalf("argv=%q: err=%v got=%+v want=(%d,%q,%q)", args, err, got, code, out, stderr)
		}
		for path, data := range before {
			after := read(path)
			if (data == nil) != (after == nil) || !bytes.Equal(data, after) {
				t.Fatalf("changed %s", path)
			}
		}
		// No successful or ordinary rejected operation leaves transaction debris.
		if len(args) > 1 && args[0] != "" {
			for _, suffix := range []string{".lock", ".tmp", ".bak"} {
				path := args[0] + suffix
				if _, watched := before[path]; !watched && read(path) != nil {
					t.Fatalf("unexpected sidecar %s", path)
				}
			}
		}
	}
	assertStore := func(data string) {
		t.Helper()
		if string(read("tasks.tsv")) != data {
			t.Fatalf("store=%q want=%q", read("tasks.tsv"), data)
		}
	}
	run([]string{"tasks.tsv", "list"}, 0, "empty\n", "")
	run([]string{"tasks.tsv", "add", "한글"}, 0, "added id=1\n", "")
	assertStore("FYTASK1\t2\n1\t0\t한글\n")
	run([]string{"tasks.tsv", "add", "  alpha  "}, 0, "added id=2\n", "")
	run([]string{"tasks.tsv", "done", "1"}, 0, "completed id=1\n", "")
	run([]string{"tasks.tsv", "done", "1"}, 0, "completed id=1\n", "", "tasks.tsv")
	run([]string{"tasks.tsv", "list", "--sort", "title"}, 0, "2\topen\t  alpha  \n1\tdone\t한글\n", "", "tasks.tsv")
	run([]string{"tasks.tsv", "delete", "2"}, 0, "deleted id=2\n", "")
	run([]string{"tasks.tsv", "delete", "1"}, 0, "deleted id=1\n", "")
	run([]string{"tasks.tsv", "add", "after empty"}, 0, "added id=3\n", "")
	assertStore("FYTASK1\t4\n3\t0\tafter empty\n")
	run([]string{"absent.tsv", "done", "1"}, 1, "", "absent.tsv: error: task ID does not exist\n")
	if read("absent.tsv") != nil {
		t.Fatal("failed operation created store")
	}
	usage := []struct {
		args    []string
		message string
	}{
		{nil, "expected store path and command; use --help"},
		{[]string{"", "list"}, "store path must contain 1..4000 UTF-8 bytes"},
		{[]string{"tasks.tsv", "listing"}, "unknown command; use list, add, done or delete"},
		{[]string{"tasks.tsv", "Add", "x"}, "unknown command; use list, add, done or delete"},
		{[]string{"tasks.tsv", "add"}, "wrong command argument count; use --help"},
	}
	for _, title := range []string{"", "   ", "a\tb", strings.Repeat("a", 513)} {
		usage = append(usage, struct {
			args    []string
			message string
		}{[]string{"tasks.tsv", "add", title}, "title must be valid UTF-8, 1..512 bytes, nonblank, without ASCII controls"})
	}
	usage = append(usage, struct {
		args    []string
		message string
	}{[]string{"tasks.tsv", "add", strings.Repeat("a", 514)}, "cannot read title/ID (maximum title 512 UTF-8 bytes)"})
	for _, id := range []string{"0", "01", "-1", "2147483648", "1x"} {
		usage = append(usage, struct {
			args    []string
			message string
		}{[]string{"tasks.tsv", "done", id}, "ID must be canonical decimal 1..2147483647"})
	}
	for _, c := range usage {
		run(c.args, 2, "", "task-manager-cli: error: "+c.message+"\n", "tasks.tsv")
	}
	corrupt := []struct{ data, message string }{
		{"", "invalid task store header"}, {"OTHER\t1\n", "invalid task store header"}, {"\xef\xbb\xbfFYTASK1\t1\n", "invalid task store header"},
		{"FYTASK1\t0\n", "invalid next task ID"}, {"FYTASK1\t2147483648\n", "invalid next task ID"},
		{"FYTASK1\t2\n1\t0\tx", "task record must end with LF"}, {"FYTASK1\t2\n1\t2\tx\n", "invalid task record fields"},
		{"FYTASK1\t2\n01\t0\tx\n", "invalid task ID in store"}, {"FYTASK1\t1\n1\t0\tx\n", "task IDs must increase and be below next ID"},
		{"FYTASK1\t2\n1\t0\ta\n1\t1\tb\n", "task IDs must increase and be below next ID"},
		{"FYTASK1\t3\n2\t0\tb\n1\t0\ta\n", "task IDs must increase and be below next ID"},
		{"FYTASK1\t2\n1\t0\t   \n", "invalid task title in store"}, {"FYTASK1\t2\n1\t0\ta\tb\n", "invalid task title in store"},
		{"FYTASK1\t2\n1\t0\t\xff\n", "invalid task title in store"}, {strings.Repeat("x", 262145), "store exceeds 262144 bytes"},
	}
	for _, c := range corrupt {
		write("bad.tsv", c.data)
		run([]string{"bad.tsv", "add", "valid"}, 1, "", "bad.tsv: error: "+c.message+"\n", "bad.tsv")
	}
	write("exhausted.tsv", "FYTASK1\t2147483647\n")
	run([]string{"exhausted.tsv", "add", "x"}, 1, "", "exhausted.tsv: error: task ID space exhausted\n", "exhausted.tsv")
	title := strings.Repeat("q", 512)
	run([]string{"boundary.tsv", "add", title}, 0, "added id=1\n", "")
	if string(read("boundary.tsv")) != "FYTASK1\t2\n1\t0\t"+title+"\n" {
		t.Fatal("boundary bytes")
	}
	for _, suffix := range []string{".tmp", ".bak", ".lock"} {
		write("tasks.tsv"+suffix, "retain")
		message := "tasks.tsv" + suffix + ": error: recovery sidecar exists; inspect .tmp/.bak before retry\n"
		if suffix == ".lock" {
			code := 17
			if runtime.GOOS == "windows" {
				code = 80
			}
			message = fmt.Sprintf("tasks.tsv.lock: error: create lock failed; another run or stale lock (status=1 os_code=%d)\n", code)
		}
		run([]string{"tasks.tsv", "list"}, 1, "", message, "tasks.tsv", "tasks.tsv"+suffix)
		if err := os.Remove(filepath.Join(root, "tasks.tsv"+suffix)); err != nil {
			t.Fatal(err)
		}
	}
	taskManagerListingCases(t, write, run)
}

func taskManagerListingCases(t *testing.T, write func(string, string), run func([]string, int, string, string, ...string)) {
	t.Helper()
	titles := []string{"z", "a", "a", "é", "e\u0301", "한", " a", "A", "10", "2", "aa", strings.Repeat("q", 512)}
	state := "FYTASK1\t13\n"
	rows := map[int]string{}
	for i, title := range titles {
		id := i + 1
		flag := id % 2
		label := "open"
		if flag == 0 {
			label = "done"
		}
		state += fmt.Sprintf("%d\t%d\t%s\n", id, 1-flag, title)
		rows[id] = fmt.Sprintf("%d\t%s\t%s\n", id, label, title)
	}
	write("list.tsv", state)
	for _, sort := range []string{"id", "title"} {
		order := []int{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12}
		if sort == "title" {
			order = []int{7, 9, 10, 8, 2, 3, 11, 5, 12, 1, 4, 6}
		}
		for _, status := range []string{"all", "open", "done"} {
			want := ""
			for _, id := range order {
				if status == "all" || (status == "done" && id%2 == 0) || (status == "open" && id%2 != 0) {
					want += rows[id]
				}
			}
			for _, args := range [][]string{{"list.tsv", "list", "--sort", sort, "--status", status}, {"list.tsv", "list", "--status", status, "--sort", sort}} {
				run(args, 0, want, "", "list.tsv")
			}
		}
	}
	bad := []struct {
		args    []string
		message string
	}{
		{[]string{"--status"}, "wrong command argument count; use --help"},
		{[]string{"--status", "open", "--sort"}, "wrong command argument count; use --help"},
		{[]string{"--other", "id"}, "unknown list option; use --status or --sort"},
		{[]string{"--Status", "open"}, "unknown list option; use --status or --sort"},
		{[]string{"--statusx", "open"}, "unknown list option; use --status or --sort"},
		{[]string{"--status", "closed"}, "status must be open, done or all"},
		{[]string{"--status", "Open"}, "status must be open, done or all"},
		{[]string{"--status", ""}, "status must be open, done or all"},
		{[]string{"--sort", "name"}, "sort must be id or title"},
		{[]string{"--sort", ""}, "sort must be id or title"},
		{[]string{"--status", "open", "--status", "done"}, "duplicate --status option"},
		{[]string{"--sort", "id", "--sort", "title"}, "duplicate --sort option"},
		{[]string{"--" + strings.Repeat("x", 14), "id"}, "cannot read list option (maximum 15 UTF-8 bytes)"},
		{[]string{"--status", strings.Repeat("x", 16)}, "cannot read list option (maximum 15 UTF-8 bytes)"},
		{[]string{"--status", "all", "--sort", "id", "extra"}, "expected store path and command; use --help"},
	}
	for _, c := range bad {
		run(append([]string{"list.tsv", "list"}, c.args...), 2, "", "task-manager-cli: error: "+c.message+"\n", "list.tsv")
	}
	run([]string{"list.tsv", "add", "x", "--sort", "title"}, 2, "", "task-manager-cli: error: wrong command argument count; use --help\n", "list.tsv")
	write("hidden.tsv", "FYTASK1\t3\n1\t0\tvalid\n2\t1\t   \n")
	run([]string{"hidden.tsv", "list", "--status", "open"}, 1, "", "hidden.tsv: error: invalid task title in store\n", "hidden.tsv")
	write("none.tsv", "FYTASK1\t2\n1\t0\tonly\n")
	run([]string{"none.tsv", "list", "--status", "done"}, 0, "empty\n", "", "none.tsv")
	write("empty.tsv", "FYTASK1\t1\n")
	run([]string{"empty.tsv", "list", "--sort", "title"}, 0, "empty\n", "", "empty.tsv")
	run([]string{"missing.tsv", "list", "--sort", "title"}, 0, "empty\n", "")
	state = "FYTASK1\t257\n"
	want := ""
	for id := 1; id <= 256; id++ {
		state += fmt.Sprintf("%d\t0\t%03d\n", id, 257-id)
	}
	for id := 256; id >= 1; id-- {
		want += fmt.Sprintf("%d\topen\t%03d\n", id, 257-id)
	}
	write("full.tsv", state)
	run([]string{"full.tsv", "list", "--sort", "title"}, 0, want, "", "full.tsv")
	run([]string{"full.tsv", "add", "extra"}, 1, "", "full.tsv: error: task count exceeds 256\n", "full.tsv")
}
