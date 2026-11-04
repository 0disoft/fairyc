package cli

import (
	"bufio"
	"bytes"
	"context"
	"os/exec"
	"strings"
	"testing"
	"time"
)

const mcpStreamMain = `module main;import project_mcp_stream as stream;import std::bytes as bytes;
@effects(console) fn i32 main(){var reader=switch(stream::open()){case .ok(p):{p}case .err(_):{return 2;}};while(true){var bytes::Buffer line=bytes::empty();let more=switch(stream::next(ref mut reader,ref mut line)){case .ok(p):{p}case .err(_):{return 2;}};if(!more){break;}if(stream::write(bytes::view(ref line))!=0 || stream::write(b"\n")!=0){return 4;}}return 0;}`

func TestCandidateMCPStream(t *testing.T) {
	candidateAtlasProbeSize(t, "project_mcp_stream", mcpStreamMain, 16<<10, func(t *testing.T, image string) {
		for _, c := range []struct {
			name, input, output string
			code                int
		}{
			{"empty", "", "", 0}, {"blank", "\n", "\n", 0}, {"crlf", "one\r\ntwo\r\n", "one\ntwo\n", 0},
			{"eof", "one\nlast", "one\nlast\n", 0}, {"eof-cr", "last\r", "last\n", 0},
			{"chunk-boundary", strings.Repeat("x", 4096) + "\r\nlast", strings.Repeat("x", 4096) + "\nlast\n", 0},
			{"exact-frame", strings.Repeat("x", 65536) + "\n", strings.Repeat("x", 65536) + "\n", 0},
			{"over-frame", strings.Repeat("x", 65537) + "\n", "", 2},
			{"exact-frame-eof", strings.Repeat("x", 65536), strings.Repeat("x", 65536) + "\n", 0},
			{"oversize-after-progress", "first\n" + strings.Repeat("x", 65537), "first\n", 2},
			{"many-frames", strings.Repeat("a\n", 1024), strings.Repeat("a\n", 1024), 0},
		} {
			t.Run(c.name, func(t *testing.T) {
				code, out := candidateAtlasProcess(t, image, []byte(c.input))
				if code != c.code || out != c.output {
					t.Fatalf("code=%d want=%d, output bytes=%d want=%d", code, c.code, len(out), len(c.output))
				}
			})
		}
		// Keep stdin open until a response is observed. A read-all-until-EOF
		// implementation cannot pass this interactive stdio observation.
		ctx, cancel := context.WithTimeout(t.Context(), 10*time.Second)
		defer cancel()
		cmd := exec.CommandContext(ctx, image)
		in, err := cmd.StdinPipe()
		if err != nil {
			t.Fatal(err)
		}
		out, err := cmd.StdoutPipe()
		if err != nil {
			t.Fatal(err)
		}
		var diagnostic bytes.Buffer
		cmd.Stderr = &diagnostic
		if err = cmd.Start(); err != nil {
			t.Fatal(err)
		}
		defer func() {
			_ = in.Close()
			_ = out.Close()
			if cmd.ProcessState == nil {
				_ = cmd.Process.Kill()
				_ = cmd.Wait()
			}
		}()
		if _, err = in.Write([]byte("live\n")); err != nil {
			t.Fatal(err)
		}
		reader := bufio.NewReader(out)
		if line, err := reader.ReadString('\n'); err != nil || line != "live\n" {
			t.Fatalf("response before EOF: %q %v %s", line, err, &diagnostic)
		}
		if _, err = in.Write([]byte("second\n")); err != nil {
			t.Fatal(err)
		}
		if line, err := reader.ReadString('\n'); err != nil || line != "second\n" {
			t.Fatalf("second live response: %q %v", line, err)
		}
		_ = in.Close()
		if err = cmd.Wait(); err != nil || diagnostic.Len() != 0 {
			t.Fatalf("stream completion: %v %s", err, &diagnostic)
		}
	})
}

func TestCandidateMCPOptions(t *testing.T) {
	main := `module main;import project_mcp_options as options;import std::io as io;@effects(console) fn i32 main(){return switch(options::arguments()){case .ok(p):{discard io::write_usize(p.packages);discard io::write("/");discard io::write_usize(p.calls);discard io::write("/");discard io::write_usize(p.depth);discard io::write("/");discard io::write_usize(p.symbols);discard io::write("/");discard io::write_usize(p.output);discard io::write("/");discard io::write_usize(p.requests);0}case .err(_):{2}};}`
	valid := []string{"--allow-impact", "--max-packages", "8", "--max-calls", "128", "--max-depth", "0", "--max-symbols", "64", "--max-output-bytes", "2048", "--max-requests", "10"}
	cases := [][]string{valid, {}, valid[1:], append(append([]string{}, valid...), "--allow-impact")}
	for i := range valid {
		if i%2 == 0 && i != 0 {
			for _, value := range []string{"", "-1", "+1", "01", "184467440737095516160", "0", "65537", "4194305"} {
				a := append([]string{}, valid...)
				a[i] = value
				cases = append(cases, a)
			}
		}
	}
	for i, endpoints := range map[int][]string{2: {"1", "65536"}, 4: {"1", "65536"}, 6: {"64", "65"}, 8: {"4096", "4097"}, 10: {"2047", "4194304"}, 12: {"1024", "1025"}} {
		for _, value := range endpoints {
			a := append([]string{}, valid...)
			a[i] = value
			cases = append(cases, a)
		}
	}
	root := t.TempDir()
	candidateAtlasProbeSize(t, "project_mcp_options", main, 16<<10, func(t *testing.T, image string) {
		for _, args := range cases {
			command := append([]string{"experimental", "atlas-mcp"}, args...)
			want, _, _ := runForTest(command, root) // Empty input grants no workspace read.
			code, out := candidateAtlasProcess(t, image, nil, command...)
			wantOutput := ""
			if want == 0 {
				values := map[string]string{}
				for i := 0; i < len(args); i++ {
					if args[i] == "--allow-impact" {
						continue
					}
					values[args[i]] = args[i+1]
					i++
				}
				for _, key := range []string{"--max-packages", "--max-calls", "--max-depth", "--max-symbols", "--max-output-bytes", "--max-requests"} {
					if wantOutput != "" {
						wantOutput += "/"
					}
					wantOutput += values[key]
				}
			}
			if code != want || out != wantOutput {
				t.Fatalf("%v: %d %q want %d %q", args, code, out, want, wantOutput)
			}
		}
	})
}
