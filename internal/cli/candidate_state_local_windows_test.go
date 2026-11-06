//go:build windows

package cli

import (
	"bufio"
	"bytes"
	"context"
	"encoding/binary"
	"errors"
	"io"
	"math"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/0disoft/fairyc/internal/contractstate"
	"github.com/0disoft/fairyc/internal/workspaceloader"
	"golang.org/x/sys/windows"
)

func stateLocalWindowsLayouts(t *testing.T) (contractstate.Layout, contractstate.Layout) {
	t.Helper()
	declaration := stateGraphIRBase(t).Layout
	build := func(appendField bool) contractstate.Layout {
		builder, err := contractstate.NewLayoutBuilder(declaration.ContractUID)
		if err != nil {
			t.Fatal(err)
		}
		for _, field := range declaration.Fields {
			if field.Kind == contractstate.FieldCell {
				_, err = contractstate.DeclareEncodedCell(builder, field.Name, field.ValueTypeID)
			} else {
				_, err = contractstate.DeclareEncodedMap(builder, field.Name, field.KeyTypeID, field.ValueTypeID)
			}
			if err != nil {
				t.Fatal(err)
			}
		}
		if appendField {
			if _, err = contractstate.DeclareEncodedCell(builder, "after", "u64/be/v1"); err != nil {
				t.Fatal(err)
			}
		}
		schema, err := builder.Build()
		if err != nil {
			t.Fatal(err)
		}
		return schema
	}
	return build(false), build(true)
}

const stateLocalWindowsProbe = `module main;
import module_state_local_windows as store;import module_state_local_codec as codec;import module_state_layout as layout;import std::bytes as bytes;import std::vec as v;import std::io as io;import std::process as process;
fn bytes::Buffer owned(ref u8[] raw){return switch(bytes::copy(raw)){case .ok(p):{p}case .err(_):{assert(false);bytes::empty()}};}
fn bytes::Buffer argument(usize at){var u8[4096] data=[0;4096];let n=switch(process::arg(at,ref mut data)){case .ok(n):{n}case .err(_):{return bytes::empty();}};let ref u8[] raw=ref data;return owned(raw[..n]);}
fn usize n64(ref u8[] raw,usize at){var usize n=0;var usize i=0;while(i<8){n=n*256+(raw[at+i] as usize);i+=1;}return n;}
fn u32[8] hash(ref u8[] raw,usize at){var u32[8] out=[0;8];var usize i=0;while(i<8){let n=at+i*4;out[i]=(raw[n] as u32)<<24|(raw[n+1] as u32)<<16|(raw[n+2] as u32)<<8|(raw[n+3] as u32);i+=1;}return out;}
@effects(console) fn void number(usize n){assert(io::write_usize(n).is_ok && io::write("\n").is_ok);}
@effects(console) fn void failure(store::Failure e){number(e.stage as usize);number(e.code as usize);number(e.os_code as usize);number(if(e.observed){1}else{0});number(e.current.version);}
@effects(console) fn Result<bytes::Buffer,u32> read(usize size){var output=bytes::empty();if(size>20971520 || !bytes::resize_zeroed(ref mut output,size).is_ok){return .err(2);}var usize used=0;while(used<size){let usize n=if(size-used>4096){4096}else{size-used};let got=switch(io::read_stdin(bytes::view_mut(ref mut output)[used..used+n])){case .ok(n):{n}case .err(_):{return .err(1);}};if(got==0){return .err(1);}used+=got;}return .ok(output);}
@effects(filesystem,console,clock) fn i32 main(){let path=argument(1);let action=argument(2);let mode=bytes::view(ref action);let head=switch(read(8)){case .ok(p):{p}case .err(_):{return 1;}};let incoming=switch(read(n64(bytes::view(ref head),0))){case .ok(p):{p}case .err(_):{return 2;}};let raw=bytes::view(ref incoming);if(raw.len<48){return 3;}let expected=n64(raw,0);let expected_hash=hash(raw,8);let size=n64(raw,40);if(size>65536){return 3;}var mutations=v::empty<codec::Mutation>();var usize at=48;var usize i=0;while(i<size){if(at>raw.len || raw.len-at<48){return 3;}let key=hash(raw,at);let flags=n64(raw,at+32);let n=n64(raw,at+40);at+=48;if(n>raw.len-at){return 3;}assert(v::push<codec::Mutation>(ref mut mutations,codec::Mutation{.key,.start=at,.end=at+n,.deleted=(flags&1)!=0,.was_null=(flags&2)!=0}).is_ok);at+=n;i+=1;}if(at!=raw.len){return 3;}let writes=codec::mutations_of(owned(raw),mutations);let initial=schemas();let next=schemas_next();let limits=codec::Limits{.bytes=16777216,.entries=65536,.value=65536};var work=codec::empty_work();
let instance=switch(store::open(bytes::view(ref path),if(bytes::equal(mode,b"snapshot-next")){ref next}else{ref initial},0,limits,bytes::equal(mode,b"create"),ref mut work)){case .ok(p):{p}case .err(e):{failure(e);return 0;}};
if(bytes::equal(mode,b"create")){discard io::write("OK");return 0;}
if(bytes::equal(mode,b"snapshot") || bytes::equal(mode,b"snapshot-next")){let state=switch(store::snapshot(ref instance,expected_hash,ref mut work)){case .ok(p):{p}case .err(e):{failure(e);return 0;}};let encoded=switch(codec::encode(ref state,limits,ref mut work)){case .ok(p):{p}case .err(_):{return 4;}};discard io::write_bytes(bytes::view(ref encoded));return 0;}
var usize version=expected;if(bytes::equal(mode,b"hold")){let state=switch(store::snapshot(ref instance,expected_hash,ref mut work)){case .ok(p):{p}case .err(e):{failure(e);return 0;}};version=codec::version(ref state);discard io::write("R");var u8[1] signal=[0;1];switch(io::read_stdin(ref mut signal)){case .ok(n):{if(n!=1){return 5;}}case .err(_):{return 5;}};}
let result=if(bytes::equal(mode,b"upgrade") || bytes::equal(mode,b"upgrade-same")){store::publish_layout(ref instance,expected_hash,version,if(bytes::equal(mode,b"upgrade-same")){ref initial}else{ref next},0,ref mut work)}else{store::compare_and_publish(ref instance,expected_hash,version,ref writes,ref mut work)};
switch(result){case .ok(receipt):{number(0);number(receipt.version);number(receipt.writes);}case .err(e):{failure(e);}};return 0;}
`

func stateLocalWindowsPacket(hash contractstate.LayoutHash, version uint64, writes []contractstate.PersistentWrite) []byte {
	body := binary.BigEndian.AppendUint64(nil, version)
	body = append(body, hash[:]...)
	body = binary.BigEndian.AppendUint64(body, uint64(len(writes)))
	for _, w := range writes {
		body = append(body, w.Key[:]...)
		flags := uint64(0)
		if w.Deleted {
			flags |= 1
		}
		if w.Value == nil {
			flags |= 2
		}
		body = binary.BigEndian.AppendUint64(body, flags)
		body = binary.BigEndian.AppendUint64(body, uint64(len(w.Value)))
		body = append(body, w.Value...)
	}
	return append(binary.BigEndian.AppendUint64(nil, uint64(len(body))), body...)
}

type stateLocalPublisher struct {
	cmd    *exec.Cmd
	input  io.WriteCloser
	output *bufio.Reader
	stderr bytes.Buffer
	cancel context.CancelFunc
	waited bool
}

func stateLocalPublisherStart(t *testing.T, image, root string, packet []byte) *stateLocalPublisher {
	t.Helper()
	ctx, cancel := context.WithTimeout(t.Context(), 30*time.Second)
	p := &stateLocalPublisher{cmd: exec.CommandContext(ctx, image, root, "hold"), cancel: cancel}
	var err error
	if p.input, err = p.cmd.StdinPipe(); err != nil {
		cancel()
		t.Fatal(err)
	}
	output, err := p.cmd.StdoutPipe()
	if err != nil {
		p.input.Close()
		cancel()
		t.Fatal(err)
	}
	p.output = bufio.NewReader(output)
	p.cmd.Stderr = &p.stderr
	if err = p.cmd.Start(); err != nil {
		p.input.Close()
		output.Close()
		cancel()
		t.Fatal(err)
	}
	t.Cleanup(func() {
		cancel()
		p.input.Close()
		output.Close()
		if !p.waited {
			p.cmd.Wait()
		}
	})
	if _, err = p.input.Write(packet); err != nil {
		t.Fatal(err)
	}
	ready, err := p.output.ReadByte()
	if err != nil || ready != 'R' {
		t.Fatalf("publisher readiness=%q error=%v stderr=%s", ready, err, &p.stderr)
	}
	return p
}
func (p *stateLocalPublisher) finish(t *testing.T) string {
	t.Helper()
	if _, err := p.input.Write([]byte{1}); err != nil {
		t.Fatal(err)
	}
	p.input.Close()
	out, err := io.ReadAll(p.output)
	if err != nil {
		t.Fatal(err)
	}
	err = p.cmd.Wait()
	p.waited = true
	p.cancel()
	if err != nil || p.stderr.Len() != 0 {
		t.Fatalf("publisher exit=%v output=%q stderr=%s", err, out, &p.stderr)
	}
	return string(out)
}

func TestCandidateStateLocalWindows(t *testing.T) {
	schema, next := stateLocalWindowsLayouts(t)
	policy := contractstate.LocalPersistenceLimits{MaxStateBytes: 16 << 20, MaxEntries: 65536, MaxValueBytes: 65536}
	source := stateLocalWindowsProbe + stateLocalCodecPlan(schema) + strings.Replace(stateLocalCodecPlan(next), "fn layout::Plan schemas()", "fn layout::Plan schemas_next()", 1)
	candidateAtlasProbeSize(t, "module_state_local_windows", source, 128<<10, func(t *testing.T, image string) {
		run := func(root, action string, hash contractstate.LayoutHash, version uint64, writes []contractstate.PersistentWrite) string {
			t.Helper()
			code, out := candidateAtlasProcess(t, image, stateLocalWindowsPacket(hash, version, writes), root, action)
			if code != 0 {
				t.Fatalf("local store exit=%d output=%q", code, out)
			}
			return out
		}
		created := func() string {
			t.Helper()
			root := t.TempDir()
			if out := run(root, "create", schema.Hash(), 0, nil); out != "OK" {
				t.Fatal("create", out)
			}
			return root
		}
		stateBytes := func(root string) []byte {
			t.Helper()
			b, err := os.ReadFile(filepath.Join(root, "state-v1.bin"))
			if err != nil {
				t.Fatal(err)
			}
			return b
		}
		clean := func(root string) {
			t.Helper()
			entries, err := os.ReadDir(root)
			if err != nil {
				t.Fatal(err)
			}
			for _, e := range entries {
				if strings.HasPrefix(e.Name(), ".tmp-") {
					t.Fatal("owned staging leaked", e.Name())
				}
			}
		}
		var a, b, untouched contractstate.Key
		a[31] = 1
		b[31] = 2
		untouched[31] = 3
		t.Run("create-reopen-go-compatible", func(t *testing.T) {
			root := created()
			if !bytes.Equal(stateBytes(root), stateLocalCodecRecord(t, schema, 0, nil)) {
				t.Fatal("new Fairy record differs")
			}
			goStore, err := contractstate.OpenLocalPersistence(root, schema, policy)
			if err != nil {
				t.Fatal(err)
			}
			defer goStore.Close()
			if _, err = goStore.CompareAndPublish(schema.Hash(), 0, []contractstate.PersistentWrite{{Key: a, Value: []byte{0, 255, 13, 10}}}); err != nil {
				t.Fatal(err)
			}
			if got := run(root, "snapshot", schema.Hash(), 0, nil); !bytes.Equal([]byte(got), stateBytes(root)) {
				t.Fatal("Fairy read Go record differs")
			}
			before := stateBytes(root)
			if out := run(root, "create", schema.Hash(), 0, nil); !strings.HasPrefix(out, "2\n") {
				t.Fatal("create collision", out)
			}
			if !bytes.Equal(before, stateBytes(root)) {
				t.Fatal("create replaced state")
			}
			clean(root)
		})
		t.Run("canonical-cas-delete-and-reopen", func(t *testing.T) {
			root := created()
			goStore, err := contractstate.OpenLocalPersistence(root, schema, policy)
			if err != nil {
				t.Fatal(err)
			}
			defer goStore.Close()
			seed := []contractstate.PersistentWrite{{Key: a, Value: nil}, {Key: b, Value: []byte{}}, {Key: untouched, Value: []byte("preserve")}}
			if _, err = goStore.CompareAndPublish(schema.Hash(), 0, seed); err != nil {
				t.Fatal(err)
			}
			writes := []contractstate.PersistentWrite{{Key: b, Value: []byte{255, 0}}, {Key: a, Deleted: true}}
			if out := run(root, "publish", schema.Hash(), 1, writes); out != "0\n2\n2\n" {
				t.Fatal("native CAS", out)
			}
			snapshot, err := goStore.Snapshot(schema.Hash())
			if err != nil || snapshot.Version != 2 || len(snapshot.Entries) != 2 || !bytes.Equal(snapshot.Entries[0].Value, []byte{255, 0}) || string(snapshot.Entries[1].Value) != "preserve" {
				t.Fatal("Go native state", snapshot, err)
			}
			before := stateBytes(root)
			if out := run(root, "publish", schema.Hash(), 1, nil); !strings.HasPrefix(out, "6\n1005\n0\n1\n2\n") {
				t.Fatal("stale no-op", out)
			}
			if out := run(root, "publish", schema.Hash(), 2, nil); out != "0\n2\n0\n" {
				t.Fatal("no-op", out)
			}
			if !bytes.Equal(before, stateBytes(root)) {
				t.Fatal("no-op/conflict touched bytes")
			}
			clean(root)
		})
		t.Run("two-process-one-winner", func(t *testing.T) {
			root := created()
			first := stateLocalPublisherStart(t, image, root, stateLocalWindowsPacket(schema.Hash(), 0, []contractstate.PersistentWrite{{Key: a, Value: []byte("first")}}))
			second := stateLocalPublisherStart(t, image, root, stateLocalWindowsPacket(schema.Hash(), 0, []contractstate.PersistentWrite{{Key: b, Value: []byte("second")}}))
			if got := first.finish(t); got != "0\n1\n1\n" {
				t.Fatal("first", got)
			}
			if got := second.finish(t); got != "6\n1005\n0\n1\n1\n" {
				t.Fatal("second", got)
			}
			goStore, err := contractstate.OpenLocalPersistence(root, schema, policy)
			if err != nil {
				t.Fatal(err)
			}
			defer goStore.Close()
			snapshot, err := goStore.Snapshot(schema.Hash())
			if err != nil || len(snapshot.Entries) != 1 || string(snapshot.Entries[0].Value) != "first" {
				t.Fatal("winner snapshot", snapshot, err)
			}
			clean(root)
		})
		t.Run("version-max-noop-refuses-nonempty", func(t *testing.T) {
			root := created()
			before := stateLocalCodecRecord(t, schema, math.MaxUint64, nil)
			if err := os.WriteFile(filepath.Join(root, "state-v1.bin"), before, 0600); err != nil {
				t.Fatal(err)
			}
			if got := run(root, "publish", schema.Hash(), math.MaxUint64, nil); got != "0\n18446744073709551615\n0\n" {
				t.Fatal(got)
			}
			if got := run(root, "publish", schema.Hash(), math.MaxUint64, []contractstate.PersistentWrite{{Key: a, Value: []byte{1}}}); !strings.HasPrefix(got, "6\n1006\n") {
				t.Fatal(got)
			}
			if !bytes.Equal(before, stateBytes(root)) {
				t.Fatal("exhausted version changed")
			}
			clean(root)
		})
		t.Run("append-layout-and-drift", func(t *testing.T) {
			root := created()
			before := stateBytes(root)
			if got := run(root, "upgrade-same", schema.Hash(), 0, nil); got != "0\n0\n0\n" {
				t.Fatal("identical layout", got)
			}
			if !bytes.Equal(before, stateBytes(root)) {
				t.Fatal("identical layout wrote bytes")
			}
			if got := run(root, "upgrade", schema.Hash(), 0, nil); got != "0\n1\n0\n" {
				t.Fatal("upgrade", got)
			}
			goStore, err := contractstate.OpenLocalPersistence(root, next, policy)
			if err != nil {
				t.Fatal(err)
			}
			defer goStore.Close()
			if _, err = goStore.Snapshot(next.Hash()); err != nil {
				t.Fatal(err)
			}
			if got := run(root, "snapshot", schema.Hash(), 0, nil); !strings.HasPrefix(got, "6\n1004\n") {
				t.Fatal("old layout open", got)
			}
			if got := run(root, "snapshot-next", next.Hash(), 0, nil); !bytes.Equal([]byte(got), stateBytes(root)) {
				t.Fatal("new schema reopen", got)
			}
			clean(root)
		})
		t.Run("rename-failure-preserves-state-and-cleans-stage", func(t *testing.T) {
			root := created()
			before := stateBytes(root)
			name, err := windows.UTF16PtrFromString(filepath.Join(root, "state-v1.bin"))
			if err != nil {
				t.Fatal(err)
			}
			if err = windows.SetFileAttributes(name, windows.FILE_ATTRIBUTE_READONLY); err != nil {
				t.Fatal(err)
			}
			defer windows.SetFileAttributes(name, windows.FILE_ATTRIBUTE_NORMAL)
			got := run(root, "publish", schema.Hash(), 0, []contractstate.PersistentWrite{{Key: a, Value: []byte{1}}})
			if !strings.HasPrefix(got, "8\n") {
				t.Fatal("expected rename refusal", got)
			}
			if !bytes.Equal(before, stateBytes(root)) {
				t.Fatal("failed rename changed state")
			}
			clean(root)
		})
		t.Run("retained-root-publication", func(t *testing.T) {
			parent := t.TempDir()
			root := filepath.Join(parent, "root")
			moved := filepath.Join(parent, "moved")
			if err := os.Mkdir(root, 0700); err != nil {
				t.Fatal(err)
			}
			if got := run(root, "create", schema.Hash(), 0, nil); got != "OK" {
				t.Fatal(got)
			}
			publisher := stateLocalPublisherStart(t, image, root, stateLocalWindowsPacket(schema.Hash(), 0, []contractstate.PersistentWrite{{Key: a, Value: []byte("moved")}}))
			if err := os.Rename(root, moved); err != nil {
				// Windows can exclude directory rename while its stable child
				// authority is open. Verify that exclusion preserves authority
				// and still permits the pending CAS; do not count replacement.
				if !errors.Is(err, windows.ERROR_ACCESS_DENIED) && !errors.Is(err, windows.ERROR_SHARING_VIOLATION) {
					t.Fatal(err)
				}
				if got := publisher.finish(t); got != "0\n1\n1\n" {
					t.Fatal("CAS after namespace exclusion", got)
				}
				if _, err := os.Stat(moved); !errors.Is(err, os.ErrNotExist) {
					t.Fatal("failed rename made destination", err)
				}
				opened, err := contractstate.OpenLocalPersistence(root, schema, policy)
				if err != nil {
					t.Fatal(err)
				}
				defer opened.Close()
				snap, err := opened.Snapshot(schema.Hash())
				if err != nil || snap.Version != 1 || len(snap.Entries) != 1 || string(snap.Entries[0].Value) != "moved" {
					t.Fatal("excluded-rename publication", snap, err)
				}
				t.Log("Windows excluded namespace replacement while retained authority was open; original-root CAS passed")
				clean(root)
				return
			}
			if err := os.Mkdir(root, 0700); err != nil {
				t.Fatal(err)
			}
			replacement := []byte("replacement-dir")
			if err := os.WriteFile(filepath.Join(root, "marker"), replacement, 0600); err != nil {
				t.Fatal(err)
			}
			if got := publisher.finish(t); got != "0\n1\n1\n" {
				t.Fatal(got)
			}
			goStore, err := contractstate.OpenLocalPersistence(moved, schema, policy)
			if err != nil {
				t.Fatal(err)
			}
			defer goStore.Close()
			snap, err := goStore.Snapshot(schema.Hash())
			if err != nil || snap.Version != 1 {
				t.Fatal(snap, err)
			}
			if _, err = os.Stat(filepath.Join(root, "state-v1.bin")); !errors.Is(err, os.ErrNotExist) {
				t.Fatal("replacement root was modified", err)
			}
			clean(moved)
		})
		t.Run("missing-corrupt-and-invalid-mutations", func(t *testing.T) {
			root := created()
			before := stateBytes(root)
			if got := run(root, "publish", schema.Hash(), 0, []contractstate.PersistentWrite{{Key: a, Value: []byte{1}}, {Key: a, Value: []byte{2}}}); !strings.HasPrefix(got, "1\n1001\n") {
				t.Fatal("duplicate", got)
			}
			if !bytes.Equal(before, stateBytes(root)) {
				t.Fatal("invalid mutation changed state")
			}
			if err := os.Remove(filepath.Join(root, "state-v1.bin")); err != nil {
				t.Fatal(err)
			}
			if got := run(root, "snapshot", schema.Hash(), 0, nil); !strings.HasPrefix(got, "4\n") {
				t.Fatal("missing initialized", got)
			}
			damaged := append([]byte{}, before...)
			damaged[len(damaged)-1] ^= 1
			if err := os.WriteFile(filepath.Join(root, "state-v1.bin"), damaged, 0600); err != nil {
				t.Fatal(err)
			}
			if got := run(root, "snapshot", schema.Hash(), 0, nil); !strings.HasPrefix(got, "5\n1001\n") {
				t.Fatal("corruption", got)
			}
			if !bytes.Equal(damaged, stateBytes(root)) {
				t.Fatal("corruption repaired implicitly")
			}
			clean(root)
		})
		t.Run("busy-and-invalid-before-lease", func(t *testing.T) {
			root := created()
			before := stateBytes(root)
			for _, invalid := range []bool{false, true} {
				writes := []contractstate.PersistentWrite{{Key: a, Value: []byte{1}}}
				if invalid {
					writes = append(writes, contractstate.PersistentWrite{Key: a, Value: []byte{2}})
				}
				publisher := stateLocalPublisherStart(t, image, root, stateLocalWindowsPacket(schema.Hash(), 0, writes))
				h := stateLockOracle(t, root)
				if err := stateLockTry(h); err != nil {
					t.Fatal(err)
				}
				out := publisher.finish(t)
				if invalid {
					if !strings.HasPrefix(out, "1\n1001\n") {
						t.Fatal("invalid mutation reached lease", out)
					}
				} else if !strings.HasPrefix(out, "3\n0\n33\n") {
					t.Fatal("busy lease did not refuse", out)
				}
				if err := windows.UnlockFileEx(h, 0, 1, 0, &windows.Overlapped{}); err != nil {
					t.Fatal(err)
				}
			}
			if !bytes.Equal(before, stateBytes(root)) {
				t.Fatal("busy/invalid published")
			}
			clean(root)
		})
	})
}

func TestCandidateStateLocalWindowsAdmissionAndOwner(t *testing.T) {
	schema, _ := stateLocalWindowsLayouts(t)
	probe := `module main;import module_state_local_windows as store;import module_state_local_codec as codec;import module_state_layout as layout;import std::bytes as bytes;import std::vec as v;
fn bytes::Buffer owned(ref u8[] b){return switch(bytes::copy(b)){case .ok(p):{p}case .err(_):{assert(false);bytes::empty()}};}
@effects(filesystem,clock)fn i32 main(){let p=schemas();let limits=codec::Limits{.bytes=16777216,.entries=65536,.value=65536};var work=codec::Work{.structure=16777217,.bytes=0};switch(store::open(b"missing-state-admission-root",ref p,0,limits,false,ref mut work)){case .ok(_):{return 1;}case .err(e):{assert(e.stage==1 && e.code==1002);}};work=codec::Work{.structure=0,.bytes=1073741825};switch(store::open(b"missing-state-admission-root",ref p,0,limits,false,ref mut work)){case .ok(_):{return 2;}case .err(e):{assert(e.stage==1 && e.code==1002);}};work=codec::empty_work();switch(store::open(b"missing-state-admission-root",ref p,0,codec::Limits{.bytes=0,.entries=1,.value=1},true,ref mut work)){case .ok(_):{return 3;}case .err(e):{assert(e.stage==1 && e.code==1001);}};return 0;}` + stateLocalCodecPlan(schema)
	candidateAtlasProbeSize(t, "module_state_local_windows", probe, 128<<10, func(t *testing.T, image string) {
		if code, out := candidateAtlasProcess(t, image, nil); code != 0 || out != "" {
			t.Fatal(code, out)
		}
	})
	for _, body := range []string{`fn store::Store forge(){return store::Store{.root=0,.authority=0,.limits=codec::Limits{.bytes=1,.entries=1,.value=1}};}`, `fn void forge(ref store::Store p){let root=ref p.root;}`} {
		root := t.TempDir()
		copyLexerExample(t, root)
		copyCandidateParityModules(t, root, "module_state_local_windows")
		writeTestSource(t, root, "src/main.fy", `module main;import module_state_local_windows as store;import module_state_local_codec as codec;`+body+`fn i32 main(){return 0;}`)
		workspace, err := workspaceloader.LoadVectorCandidate(root)
		if err != nil {
			t.Fatal(err)
		}
		var diagnostic bytes.Buffer
		_, code := lowerLoadedWorkspaceExecutable(workspace, "missing", workspaceCompilePolicy{}, "sealed local provider", &diagnostic)
		if code == ExitSuccess || !strings.Contains(diagnostic.String(), "E0103") {
			t.Fatalf("sealed provider code=%d: %s", code, &diagnostic)
		}
	}
}
