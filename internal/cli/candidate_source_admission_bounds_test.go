package cli

import (
	"context"
	"os/exec"
	"testing"
	"time"
)

// Exercise the authored admission owners without rebuilding the full dispatcher.
// Source bytes live in Buffer/Vec; joined-unit arenas remain independently bounded.
func TestCandidateSourceAdmissionBounds(t *testing.T) {
	candidateAtlasProbeSize(t, "module_catalog", sourceAdmissionBoundsProbe, 256<<10, func(t *testing.T, image string) {
		ctx, cancel := context.WithTimeout(t.Context(), 90*time.Second)
		defer cancel()
		out, err := exec.CommandContext(ctx, image).CombinedOutput()
		if err != nil || string(out) != "manifest\nfiles\nbytes\nok\n" {
			t.Fatalf("source admission boundaries: %v deadline=%v output=%q", err, ctx.Err(), out)
		}
	}, "module_cli", "module_units")
}

const sourceAdmissionBoundsProbe = `module main;
import candidate_limits as limits;
import module_cli as manifests;
import module_catalog as catalog;
import module_units as units;
import module_ast_join as join;
import scanner as scanner;
import syntax as syntax;
import std::bytes as bytes;
import std::vec as v;
import std::io as io;

fn syntax::Parsed source(usize index,usize size){
 var bytes::Buffer raw=bytes::empty();
 assert(bytes::append(ref mut raw,b"module m").is_ok && bytes::append_usize(ref mut raw,index).is_ok && bytes::append_byte(ref mut raw,59).is_ok);
 if(size!=0){assert(size>=bytes::len(ref raw)+4 && size<=limits::source_bytes());assert(bytes::append(ref mut raw,b"/*").is_ok);
  var u8[4096] spaces=[32;4096];let ref u8[] padding=ref spaces;
  while(bytes::len(ref raw)<size-2){let remaining=size-2-bytes::len(ref raw);let usize count=if(remaining<4096){remaining}else{4096};assert(bytes::append(ref mut raw,padding[..count]).is_ok);}
  assert(bytes::append(ref mut raw,b"*/").is_ok);
 }
 let stream=switch(scanner::collect(raw,false,limits::source_bytes(),1048576,256,16777216)){case .ok(p):{p}case .err(_):{assert(false);return source(index,0);}};
 let parsed=switch(syntax::parse(stream,512)){case .ok(p):{p}case .err(_):{assert(false);return source(index,0);}};
 assert(syntax::problem_count(ref parsed)==0);return parsed;
}
fn void manifest_boundaries(){
 var bytes::Buffer text=bytes::empty();var usize i=0;
 while(i<limits::source_files()){assert(bytes::append(ref mut text,b"src/m").is_ok && bytes::append_usize(ref mut text,i).is_ok && bytes::append(ref mut text,b".fy\n").is_ok);i+=1;}
 var v::Vec<manifests::Path> paths=v::empty<manifests::Path>();
 assert(manifests::paths(bytes::view(ref text),ref mut paths)==0 && v::len<manifests::Path>(ref paths)==1024);
 assert(bytes::append(ref mut text,b"src/over.fy\n").is_ok);discard v::clear<manifests::Path>(ref mut paths);
 assert(manifests::paths(bytes::view(ref text),ref mut paths)==5 && v::len<manifests::Path>(ref paths)==1024);
}
fn void file_boundaries(){
 var v::Vec<syntax::Parsed> files=v::empty<syntax::Parsed>();var usize i=0;
 while(i<1024){assert(v::push<syntax::Parsed>(ref mut files,source(i,0)).is_ok);i+=1;}
 let names=switch(catalog::build(ref files)){case .ok(p):{p}case .err(_):{assert(false);return;}};
 assert(catalog::module_count(ref names)==1024 && catalog::source_matches(ref names,1023,v::at<syntax::Parsed>(ref files,1023)));
 let grouped=switch(units::build(ref files)){case .ok(p):{p}case .err(_):{assert(false);return;}};
 assert(units::file_count(ref grouped)==1024 && units::count(ref grouped)==2);
 assert(join::join_range(ref files,0,512).is_ok && !join::join(ref files).is_ok);
 assert(v::push<syntax::Parsed>(ref mut files,source(1024,0)).is_ok);
 assert(!catalog::build(ref files).is_ok && !units::build(ref files).is_ok);
}
fn void byte_boundaries(){
 let count=limits::source_set_bytes()/limits::source_bytes();assert(count==128);
 var v::Vec<syntax::Parsed> files=v::empty<syntax::Parsed>();var usize i=0;
 while(i<count){assert(v::push<syntax::Parsed>(ref mut files,source(i,limits::source_bytes())).is_ok);i+=1;}
 assert(catalog::build(ref files).is_ok && !units::build(ref files).is_ok);
 assert(v::push<syntax::Parsed>(ref mut files,source(count,0)).is_ok);
 assert(!catalog::build(ref files).is_ok);discard v::pop<syntax::Parsed>(ref mut files);discard v::pop<syntax::Parsed>(ref mut files);
 assert(v::push<syntax::Parsed>(ref mut files,source(count-1,limits::source_bytes()-(count-1))).is_ok);
 let grouped=switch(units::build(ref files)){case .ok(p):{p}case .err(_):{assert(false);return;}};
 assert(units::source_bytes(ref grouped)==8388608 && catalog::build(ref files).is_ok);
 discard v::pop<syntax::Parsed>(ref mut files);
 assert(v::push<syntax::Parsed>(ref mut files,source(count-1,limits::source_bytes()-(count-2))).is_ok);
 assert(catalog::build(ref files).is_ok && !units::build(ref files).is_ok);
}
@effects(console) fn i32 main(){
 assert(limits::source_files()==1024 && limits::source_set_bytes()==8388608);
 assert(limits::source_bytes()==65536 && limits::program_bytes()==1048576);
 assert(io::write("manifest\n").is_ok);manifest_boundaries();
 assert(io::write("files\n").is_ok);file_boundaries();
 assert(io::write("bytes\n").is_ok);byte_boundaries();
 assert(io::write("ok\n").is_ok);return 0;
}
`
