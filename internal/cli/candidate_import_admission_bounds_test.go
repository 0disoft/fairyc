package cli

import (
	"context"
	"os/exec"
	"testing"
	"time"
)

func TestCandidateImportAdmissionBounds(t *testing.T) {
	candidateAtlasProbeSize(t, "module_catalog", importAdmissionBoundsProbe, 256<<10, func(t *testing.T, image string) {
		ctx, cancel := context.WithTimeout(t.Context(), 90*time.Second)
		defer cancel()
		if out, err := exec.CommandContext(ctx, image).CombinedOutput(); err != nil || len(out) != 0 {
			t.Fatalf("import/binding boundaries: %v deadline=%v output=%q", err, ctx.Err(), out)
		}
	})
}

const importAdmissionBoundsProbe = `module main;
import candidate_limits as limits;
import module_catalog as catalog;
import scanner as scanner;
import syntax as syntax;
import std::bytes as bytes;
import std::vec as v;

fn syntax::Parsed source(usize owner,usize count){
 var bytes::Buffer raw=bytes::empty();
 assert(bytes::append(ref mut raw,b"module importer").is_ok && bytes::append_usize(ref mut raw,owner).is_ok && bytes::append_byte(ref mut raw,59).is_ok);
 if(owner==64){discard bytes::clear(ref mut raw);assert(bytes::append(ref mut raw,b"module provider;fn i32 tail(){return 0;}").is_ok);}
 var usize i=0;while(i<count){assert(bytes::append(ref mut raw,b"import provider as a").is_ok && bytes::append_usize(ref mut raw,i).is_ok && bytes::append_byte(ref mut raw,59).is_ok);i+=1;}
 let stream=switch(scanner::collect(raw,false,limits::source_bytes(),1048576,256,16777216)){case .ok(p):{p}case .err(_):{assert(false);return source(owner,0);}};
 let p=switch(syntax::parse(stream,512)){case .ok(p):{p}case .err(_):{assert(false);return source(owner,0);}};assert(syntax::problem_count(ref p)==0);return p;
}
fn i32 main(){
 assert(limits::project_imports()==8192);
 var v::Vec<syntax::Parsed> files=v::empty<syntax::Parsed>();var v::Vec<usize> owners=v::empty<usize>();var v::Vec<catalog::Binding> bindings=v::empty<catalog::Binding>();var usize i=0;
 while(i<64){assert(v::push<syntax::Parsed>(ref mut files,source(i,128)).is_ok && v::push<usize>(ref mut owners,0).is_ok);var usize j=0;
  while(j<128){assert(v::push<catalog::Binding>(ref mut bindings,catalog::Binding{.source=i,.declaration=j+2,.target=64}).is_ok);j+=1;}i+=1;
 }
 assert(v::push<syntax::Parsed>(ref mut files,source(64,0)).is_ok && v::push<usize>(ref mut owners,0).is_ok);
 let accepted=switch(catalog::build(ref files)){case .ok(p):{p}case .err(_):{assert(false);return 1;}};
 assert(catalog::declaration_count(ref accepted)==1 && catalog::module_count(ref accepted)==65);
 assert(catalog::build_scoped(ref files,ref owners,ref bindings).is_ok);
 assert(v::push<catalog::Binding>(ref mut bindings,catalog::Binding{.source=0,.declaration=2,.target=64}).is_ok);
 switch(catalog::build_scoped(ref files,ref owners,ref bindings)){case .ok(_):{assert(false);}case .err(e):{assert(e.code==9);}};
 assert(v::push<syntax::Parsed>(ref mut files,source(65,1)).is_ok);
 switch(catalog::build(ref files)){case .ok(_):{assert(false);}case .err(e):{assert(e.code==10 && e.module_index==65);}};
 return 0;
}
`
