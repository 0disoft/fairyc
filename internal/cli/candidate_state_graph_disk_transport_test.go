package cli

import (
	"bytes"
	"crypto/sha256"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
	"time"
)

// These deliberately opaque payloads exercise transport authority only. A magic
// prefix and a matching content address never grant semantic graph admission.
const stateGraphDiskTransportProbe = `module main;
import project_cache_disk as disk;
import windows_rooted_file as rooted;
import std::bytes as bytes;
import std::io as io;
import std::process as process;
import std::fs as fs;
import std::text as text;

@effects(console) fn bytes::Buffer input(){
 var out=bytes::empty();var u8[4096] chunk=[0;4096];
 while(true){let n=switch(io::read_stdin(ref mut chunk)){case .ok(n):{n}case .err(_):{assert(false);0}};if(n==0){break;}
 let ref u8[] raw=ref chunk;assert(bytes::append(ref mut out,raw[..n]).is_ok && bytes::len(ref out)<=262145);}
 return out;
}
fn bytes::Buffer argument(usize at){var u8[4096] data=[0;4096];let n=switch(process::arg(at,ref mut data)){case .ok(n):{n}case .err(_):{assert(false);0}};let ref u8[] raw=ref data;return switch(bytes::copy(raw[..n])){case .ok(p):{p}case .err(_):{assert(false);bytes::empty()}};}
@effects(console) fn void failure(disk::Problem e){discard io::write_bytes(b"E");discard io::write_u32(e.kind);}
@effects(console) fn void stats(disk::Stats p){discard io::write_usize(p.entries);discard io::write_bytes(b",");discard io::write_usize(p.bytes);discard io::write_bytes(b",");discard io::write_usize(p.evicted);}
@effects(filesystem,clock,console) fn i32 main(){
 let path=argument(1);let mode_buffer=argument(2);let mode=bytes::view(ref mode_buffer);let data=input();let raw=bytes::view(ref data);
 let key=b"fylookup0-sha256-aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa";
 let limits=disk::Limits{.entries=if(bytes::equal(mode,b"put-entry")){2}else{if(bytes::equal(mode,b"sweep-entry")){1}else{8}},.bytes=if(bytes::equal(mode,b"put-byte")){22}else{524288},.record=if(bytes::equal(mode,b"put-byte")){12}else{262144},.temporary_age=1};
 {
 let store=switch(disk::open(bytes::view(ref path),limits)){case .ok(p):{p}case .err(e):{failure(e);return 0;}};
 var usize work=0;var usize scan=0;
 if(bytes::equal(mode,b"load")){switch(disk::load(ref store,key,ref mut work,ref mut scan)){case .ok(p):{discard io::write_bytes(bytes::view(ref p));}case .err(e):{failure(e);}};return 0;}
 if(bytes::equal(mode,b"next") || bytes::equal(mode,b"vanish")){
  let cursor=switch(disk::candidates(ref store,key,ref mut scan)){case .ok(p):{p}case .err(e):{failure(e);return 0;}};
  if(bytes::equal(mode,b"vanish")){
   let root=switch(rooted::open(bytes::view(ref path))){case .ok(p):{p}case .err(_):{return 1;}};scan=0;
   let entries=switch(rooted::scan(ref root,4096,ref mut scan)){case .ok(p):{p}case .err(_):{return 2;}};
   var usize i=0;while(i<rooted::count(ref entries)){assert(switch(rooted::erase_if(ref root,rooted::name(ref entries,i),rooted::metadata(ref entries,i))){case .ok(done):{done}case .err(_):{false}});i+=1;}
  }
  var usize at=disk::candidate_count(ref cursor);
  while(true){switch(disk::next(ref store,ref cursor,ref mut at,ref mut work)){
   case .ok(p):{let value=disk::candidate_bytes(ref p);discard io::write_usize(value.len);discard io::write_bytes(b":");discard io::write_bytes(value);}
   case .err(e):{failure(e);break;}
  };}return 0;
 }
 if(bytes::equal(mode,b"sweep-entry")){switch(disk::sweep(ref store,b"",ref mut scan)){case .ok(p):{stats(p);}case .err(e):{failure(e);}};return 0;}
 if(bytes::equal(mode,b"budget")){work=disk::work_limit()+1;}
 switch(disk::put(ref store,key,raw,ref mut work,ref mut scan)){case .ok(p):{if(!bytes::equal(mode,b"cleanup")){stats(p);return 0;}}case .err(e):{failure(e);return 0;}};
 // Delete while the Store remains alive: leaked staging/read/write handles
 // would prevent the exact observed deletion on Windows.
 let cursor=switch(disk::candidates(ref store,key,ref mut scan)){case .ok(p):{p}case .err(_):{return 3;}};var usize at=disk::candidate_count(ref cursor);
 {let candidate=switch(disk::next(ref store,ref cursor,ref mut at,ref mut work)){case .ok(p):{p}case .err(_):{return 4;}};assert(bytes::equal(disk::candidate_bytes(ref candidate),raw));}
 let root=switch(rooted::open(bytes::view(ref path))){case .ok(p):{p}case .err(_):{return 5;}};
 let entries=switch(rooted::scan(ref root,4096,ref mut scan)){case .ok(p):{p}case .err(_):{return 6;}};assert(rooted::count(ref entries)==1);
 assert(switch(rooted::erase_if(ref root,rooted::name(ref entries,0),rooted::metadata(ref entries,0))){case .ok(done):{done}case .err(_):{false}});
 assert(rooted::close_directory(root).is_ok);
 }
 let utf8=switch(text::validate_utf8(bytes::view(ref path))){case .ok(p):{p}case .err(_):{return 7;}};
 let cstr=switch(text::cstring_from_text(utf8)){case .ok(p):{p}case .err(_):{return 8;}};
 assert(fs::remove_dir(text::cstring_as_cstr(ref cstr)).is_ok);discard io::write_bytes(b"clean");return 0;
}`

func TestCandidateStateGraphDiskTransport(t *testing.T) {
	candidateAtlasProbeSize(t, "project_cache_disk", stateGraphDiskTransportProbe, 512<<10, func(t *testing.T, image string) {
		run := func(t *testing.T, root, mode string, raw []byte, want []byte) {
			t.Helper()
			code, out := candidateAtlasProcess(t, image, raw, filepath.ToSlash(root), mode)
			if code != 0 || !bytes.Equal([]byte(out), want) {
				t.Fatalf("%s: exit=%d output=%q want=%q", mode, code, out, want)
			}
			if entries, err := os.ReadDir(root); err == nil {
				for _, entry := range entries {
					if strings.HasPrefix(entry.Name(), ".stage") {
						t.Fatalf("%s left staging file %s", mode, entry.Name())
					}
				}
			} else if mode != "cleanup" {
				t.Fatal(err)
			}
		}
		name := func(raw []byte) string {
			return fmt.Sprintf("fyproject0-%s-%x.fyc", strings.Repeat("a", 64), sha256.Sum256(raw))
		}
		inventory := func(t *testing.T, root string, want map[string][]byte) {
			t.Helper()
			entries, err := os.ReadDir(root)
			if err != nil {
				t.Fatal(err)
			}
			if len(entries) != len(want) {
				t.Fatalf("inventory has %d files, want %d: %v", len(entries), len(want), entries)
			}
			for _, entry := range entries {
				raw, err := os.ReadFile(filepath.Join(root, entry.Name()))
				if expected, ok := want[entry.Name()]; err != nil || !ok || !bytes.Equal(raw, expected) || entry.Name() != name(raw) {
					t.Fatalf("content-address inventory mismatch: %s: %v", entry.Name(), err)
				}
			}
		}
		ordered := func(t *testing.T, root string) []byte {
			t.Helper()
			entries, err := os.ReadDir(root)
			if err != nil {
				t.Fatal(err)
			}
			sort.Slice(entries, func(i, j int) bool {
				a, e1 := entries[i].Info()
				b, e2 := entries[j].Info()
				if e1 != nil || e2 != nil {
					t.Fatal(e1, e2)
				}
				if a.ModTime().Equal(b.ModTime()) {
					return a.Name() > b.Name()
				}
				return a.ModTime().After(b.ModTime())
			})
			var want bytes.Buffer
			for _, entry := range entries {
				raw, err := os.ReadFile(filepath.Join(root, entry.Name()))
				if err != nil {
					t.Fatal(err)
				}
				fmt.Fprintf(&want, "%d:", len(raw))
				want.Write(raw)
			}
			want.WriteString("E2")
			return want.Bytes()
		}
		stamp := func(t *testing.T, root string, raw []byte, i int) {
			t.Helper()
			at := time.Unix(1700000000+int64(i), 0)
			if err := os.Chtimes(filepath.Join(root, name(raw)), at, at); err != nil {
				t.Fatal(err)
			}
		}
		write := func(t *testing.T, root, leaf string, raw []byte) {
			t.Helper()
			if err := os.WriteFile(filepath.Join(root, leaf), raw, 0600); err != nil {
				t.Fatal(err)
			}
		}
		t.Run("three-families-and-binary", func(t *testing.T) {
			root := t.TempDir()
			want := make(map[string][]byte)
			total := 0
			for _, marker := range []string{"FYCACHE0", "FYCACHE1", "FYSGCAC0"} {
				raw := []byte(marker)
				total += len(raw)
				run(t, root, "put", raw, []byte(fmt.Sprintf("%d,%d,0", len(want)+1, total)))
				want[name(raw)] = raw
				stamp(t, root, raw, len(want))
				run(t, root, "load", nil, raw)
			}
			raw := make([]byte, 131073)
			copy(raw, "FYSGCAC0")
			raw[65536], raw[len(raw)-1] = 42, 255
			run(t, root, "put", raw, []byte(fmt.Sprintf("4,%d,0", total+len(raw))))
			want[name(raw)] = raw
			inventory(t, root, want)
			run(t, root, "load", nil, raw)
			run(t, root, "next", nil, ordered(t, root))
			before, err := os.Stat(filepath.Join(root, name(raw)))
			if err != nil {
				t.Fatal(err)
			}
			run(t, root, "put", raw, []byte(fmt.Sprintf("4,%d,0", total+len(raw))))
			after, err := os.Stat(filepath.Join(root, name(raw)))
			if err != nil || !os.SameFile(before, after) || !before.ModTime().Equal(after.ModTime()) {
				t.Fatalf("identical collision replaced the existing generation: %v", err)
			}
			for _, invalid := range []string{"", "FYSGCAC", "FYSGCAC1opaque", "FYCACHE2opaque", "unknown!"} {
				run(t, root, "put", []byte(invalid), []byte("E7"))
			}
			run(t, root, "budget", raw, []byte("E9"))
			inventory(t, root, want)
		})
		t.Run("corrupt-collision-and-missing", func(t *testing.T) {
			root := t.TempDir()
			old, newest := []byte("FYSGCAC0old"), []byte("FYSGCAC0new")
			run(t, root, "put", old, []byte("1,11,0"))
			stamp(t, root, old, 0)
			run(t, root, "put", newest, []byte("2,22,0"))
			write(t, root, name(newest), []byte("FYSGCAC0bad"))
			run(t, root, "load", nil, old)
			run(t, root, "next", nil, []byte("11:FYSGCAC0oldE2"))
			run(t, root, "put", newest, []byte("2,22,0"))
			inventory(t, root, map[string][]byte{name(old): old, name(newest): newest})
			// Correctly hashed but unknown magic, and graph bytes under a false
			// digest name, must both be skipped rather than transported.
			unknown := []byte("notcachebad")
			write(t, root, name(unknown), unknown)
			fake := fmt.Sprintf("fyproject0-%s-%s.fyc", strings.Repeat("a", 64), strings.Repeat("0", 64))
			write(t, root, fake, []byte("FYSGCAC0fake"))
			run(t, root, "load", nil, newest)
			run(t, root, "next", nil, []byte("11:FYSGCAC0new11:FYSGCAC0oldE2"))
			// A malformed exact-name collision is repaired only by a real put.
			write(t, root, name(newest), []byte("broken"))
			run(t, root, "put", newest, []byte("4,45,0"))
			if got, err := os.ReadFile(filepath.Join(root, name(newest))); err != nil || !bytes.Equal(got, newest) {
				t.Fatalf("malformed collision was not repaired: %v", err)
			}
			if err := os.Remove(filepath.Join(root, name(old))); err != nil {
				t.Fatal(err)
			}
			if err := os.Remove(filepath.Join(root, name(newest))); err != nil {
				t.Fatal(err)
			}
			run(t, root, "load", nil, []byte("E2"))
			run(t, root, "next", nil, []byte("E2"))
			missing := t.TempDir()
			run(t, missing, "put", newest, []byte("1,11,0"))
			run(t, missing, "vanish", nil, []byte("E2"))
			inventory(t, missing, map[string][]byte{})
			run(t, missing, "put", newest, []byte("1,11,0"))
			inventory(t, missing, map[string][]byte{name(newest): newest})
		})
		t.Run("entry-and-byte-retention", func(t *testing.T) {
			root := t.TempDir()
			a, b, c := []byte("FYCACHE0aaa"), []byte("FYCACHE1bbb"), []byte("FYSGCAC0ccc")
			run(t, root, "put-entry", a, []byte("1,11,0"))
			stamp(t, root, a, 0)
			run(t, root, "put-entry", b, []byte("2,22,0"))
			stamp(t, root, b, 1)
			run(t, root, "put-entry", c, []byte("2,22,1"))
			inventory(t, root, map[string][]byte{name(b): b, name(c): c})
			run(t, root, "sweep-entry", nil, []byte("1,11,1"))
			inventory(t, root, map[string][]byte{name(c): c})
			root = t.TempDir()
			run(t, root, "put-byte", a, []byte("1,11,0"))
			stamp(t, root, a, 0)
			run(t, root, "put-byte", b, []byte("2,22,0"))
			stamp(t, root, b, 1)
			c = []byte("FYSGCAC0cccc")
			run(t, root, "put-byte", c, []byte("1,12,2"))
			run(t, root, "put-byte", []byte("FYSGCAC0extra"), []byte("E9"))
			run(t, root, "load", nil, c)
			inventory(t, root, map[string][]byte{name(c): c})
		})
		t.Run("live-handle-and-stage-cleanup", func(t *testing.T) {
			root := t.TempDir()
			run(t, root, "cleanup", []byte("FYSGCAC0untrusted"), []byte("clean"))
			if _, err := os.Stat(root); !os.IsNotExist(err) {
				t.Fatalf("in-process cleanup retained root: %v", err)
			}
		})
	})
}
