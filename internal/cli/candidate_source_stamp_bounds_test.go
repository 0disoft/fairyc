package cli

import (
	"context"
	"os/exec"
	"testing"
	"time"
)

func TestCandidateSourceStampBounds(t *testing.T) {
	candidateAtlasProbeSize(t, "generic_package_bindings", sourceStampBoundsProbe, 256<<10, func(t *testing.T, image string) {
		ctx, cancel := context.WithTimeout(t.Context(), 90*time.Second)
		defer cancel()
		out, err := exec.CommandContext(ctx, image).CombinedOutput()
		if err != nil || string(out) != "bindings\ncontext\nidentities\nowner\nok\n" {
			t.Fatalf("source stamps: %v deadline=%v output=%q", err, ctx.Err(), out)
		}
	}, "generic_package_context", "module_public_type_ids", "module_unit_bind")
}

const sourceStampBoundsProbe = `module main;
import candidate_limits as limits;
import generic_package_bindings as bindings;
import generic_package_context as context;
import module_catalog as catalog;
import module_public_type_text as coordinates;
import module_public_type_ids as identities;
import module_public_interface_model as public_model;
import module_signatures as signatures;
import module_unit_bind as binding;
import module_unit_interfaces as typed;
import module_types as types;
import scanner as scanner;
import syntax as syntax;
import std::bytes as bytes;
import std::vec as v;
import std::io as io;

fn syntax::Parsed parsed(bytes::Buffer raw){
 let stream=switch(scanner::collect(raw,false,65536,1048576,256,16777216)){case .ok(p):{p}case .err(_):{assert(false);return parsed(bytes::empty());}};
 let p=switch(syntax::parse(stream,512)){case .ok(p):{p}case .err(_):{assert(false);return parsed(bytes::empty());}};
 assert(syntax::problem_count(ref p)==0);return p;
}
fn syntax::Parsed source(usize index){
 var bytes::Buffer raw=bytes::empty();assert(bytes::append(ref mut raw,b"module m").is_ok && bytes::append_usize(ref mut raw,index).is_ok && bytes::append_byte(ref mut raw,59).is_ok);
 if(index==0){assert(bytes::append(ref mut raw,b"fn i32 marker(){return 0;}").is_ok);}return parsed(raw);
}
fn coordinates::Coordinates coordinate(usize owner){
 var bytes::Buffer names=bytes::empty();assert(bytes::append(ref mut names,b"fypc1-sha256-").is_ok);var usize i=0;
 while(i<64){assert(bytes::append_byte(ref mut names,49).is_ok);i+=1;}
 var v::Vec<coordinates::Coordinate> rows=v::empty<coordinates::Coordinate>();
 assert(v::push<coordinates::Coordinate>(ref mut rows,coordinates::Coordinate{.owner,.start=0,.end=77}).is_ok);return coordinates::assemble(names,rows);
}
fn bindings::Bindings collected(ref v::Vec<syntax::Parsed> sources,ref v::Vec<usize> owners){
 let resolved=v::empty<catalog::Binding>();let table=coordinate(0);var usize work=0;
 return switch(bindings::build(sources,owners,ref resolved,ref table,ref mut work)){case .ok(p):{p}case .err(e):{assert(false);return collected(sources,owners);}};
}
fn void copy_and_admission(ref v::Vec<syntax::Parsed> sources,ref v::Vec<usize> owners,ref bindings::Bindings original){
 assert(bindings::count(original)==1024);var usize work=0;
 var copy=switch(bindings::copy(original,ref mut work)){case .ok(p):{p}case .err(_):{assert(false);return;}};
 assert(bindings::count(ref copy)==1024 && bindings::module_at(ref copy,1023).source_size==syntax::source_view(v::at<syntax::Parsed>(sources,1023)).len);
 var v::Vec<catalog::Binding> resolved=v::empty<catalog::Binding>();var usize i=0;
 while(i<limits::project_imports()){assert(v::push<catalog::Binding>(ref mut resolved,catalog::Binding{.source=0,.declaration=2,.target=1}).is_ok);i+=1;}
 let table=coordinate(0);work=16777216;
 assert(switch(bindings::build(sources,owners,ref resolved,ref table,ref mut work)){case .ok(_):{false}case .err(e):{e.code==10 && work==16777216}});
 assert(v::push<catalog::Binding>(ref mut resolved,catalog::Binding{.source=0,.declaration=2,.target=1}).is_ok);work=0;
 assert(switch(bindings::build(sources,owners,ref resolved,ref table,ref mut work)){case .ok(_):{false}case .err(e):{e.code==9 && work==0}});
}
fn void context_stamps(ref v::Vec<syntax::Parsed> sources,ref bindings::Bindings original){
 var context::Budget work=context::empty();
 let retained=switch(context::compute(sources,original,ref mut work)){case .ok(p):{p}case .err(code):{assert(false);return;}};
 assert(context::count(ref retained)==1024 && context::digest(ref retained).len==64);
 var changed=context::Budget{.work=16777216,.encoded_work=0,.hashed_bytes=0,.code=0};
 assert(switch(context::compute(sources,original,ref mut changed)){case .ok(_):{false}case .err(code):{code==10}});
}
fn void type_stamps(ref v::Vec<syntax::Parsed> sources){
 let metadata=switch(signatures::build(sources)){case .ok(p):{p}case .err(_):{assert(false);return;}};
 // One independently bounded unit supplies the real scalar type graph. Global
 // source stamps are authenticated against the separate full-source metadata.
 var v::Vec<syntax::Parsed> local=v::empty<syntax::Parsed>();assert(v::push<syntax::Parsed>(ref mut local,source(0)).is_ok);
 let small=switch(signatures::build(ref local)){case .ok(p):{p}case .err(_):{assert(false);return;}};
 let bound=switch(binding::bind(ref local,ref small,0,1)){case .ok(p):{p}case .err(_):{assert(false);return;}};
 let graph=switch(typed::build(ref local,ref small,ref bound)){case .ok(p):{p}case .err(_):{assert(false);return;}};
 var v::Vec<identities::Row> rows=v::empty<identities::Row>();var usize i=1;
 while(i<=typed::type_count(ref graph)){assert(v::push<identities::Row>(ref mut rows,identities::Row{.type=typed::type_at(ref graph,i),.owner=0,.name=public_model::Text{.start=0,.end=0}}).is_ok);i+=1;}
 var v::Vec<identities::Stamp> stamps=v::empty<identities::Stamp>();i=0;
 while(i<v::len<syntax::Parsed>(sources)){assert(v::push<identities::Stamp>(ref mut stamps,identities::Stamp{.owner=signatures::module_owner(ref metadata,i),.digest=signatures::source_digest(ref metadata,i)}).is_ok);i+=1;}
 var table=identities::assemble(bytes::empty(),rows,stamps);var usize work=0;
 assert(identities::current(ref table,sources,ref metadata,ref graph,ref mut work)==0);
 var v::Vec<identities::Row> copied_rows=v::empty<identities::Row>();i=1;
 while(i<=identities::count(ref table)){assert(v::push<identities::Row>(ref mut copied_rows,identities::at(ref table,i)).is_ok);i+=1;}
 var v::Vec<identities::Stamp> changed=v::empty<identities::Stamp>();i=0;
 while(i<v::len<syntax::Parsed>(sources)){var stamp=identities::Stamp{.owner=signatures::module_owner(ref metadata,i),.digest=signatures::source_digest(ref metadata,i)};if(i==1023){stamp.digest[0]^=1;}assert(v::push<identities::Stamp>(ref mut changed,stamp).is_ok);i+=1;}
 let stale=identities::assemble(bytes::empty(),copied_rows,changed);work=0;
 assert(identities::current(ref stale,sources,ref metadata,ref graph,ref mut work)==12);
}
fn void owner_boundary(){
 var v::Vec<syntax::Parsed> one=v::empty<syntax::Parsed>();assert(v::push<syntax::Parsed>(ref mut one,source(0)).is_ok);
 var v::Vec<usize> owners=v::empty<usize>();assert(v::push<usize>(ref mut owners,511).is_ok);
 let resolved=v::empty<catalog::Binding>();let table=coordinate(511);var usize work=0;
 assert(bindings::build(ref one,ref owners,ref resolved,ref table,ref mut work).is_ok);
 let owner=v::at_mut<usize>(ref mut owners,0);*owner=512;work=0;
 assert(switch(bindings::build(ref one,ref owners,ref resolved,ref table,ref mut work)){case .ok(_):{false}case .err(e):{e.code==9 && e.module_index==0}});
 let vector=types::Type{.kind=13,.scalar=0,.module_index=0,.declaration=1,.first=1,.second=0};
 var bytes::Buffer name=bytes::empty();assert(bytes::append(ref mut name,b"std::vec::_fy_generic_g0_").is_ok);var usize i=0;while(i<64){assert(bytes::append_byte(ref mut name,49).is_ok);i+=1;}
 var v::Vec<identities::Row> rows=v::empty<identities::Row>();assert(v::push<identities::Row>(ref mut rows,identities::Row{.type=vector,.owner=511,.name=public_model::Text{.start=0,.end=bytes::len(ref name)}}).is_ok);
 let stamps=v::empty<identities::Stamp>();var identified=identities::assemble(name,rows,stamps);
 assert(switch(identities::owner(ref identified,1,vector)){case .ok(n):{n==511}case .err(_):{false}});
 var row=identities::at(ref identified,1);row.owner=512;var v::Vec<identities::Row> invalid_rows=v::empty<identities::Row>();assert(v::push<identities::Row>(ref mut invalid_rows,row).is_ok);
 let copied=switch(bytes::copy(identities::name(ref identified,1))){case .ok(p):{p}case .err(_):{assert(false);return;}};
 let no_stamps=v::empty<identities::Stamp>();let invalid=identities::assemble(copied,invalid_rows,no_stamps);
 assert(switch(identities::owner(ref invalid,1,vector)){case .ok(_):{false}case .err(code):{code==9}});
}
@effects(console) fn i32 main(){
 assert(limits::source_files()==1024 && limits::project_imports()==8192);
 var v::Vec<syntax::Parsed> sources=v::empty<syntax::Parsed>();var v::Vec<usize> owners=v::empty<usize>();var usize i=0;
 while(i<limits::source_files()){assert(v::push<syntax::Parsed>(ref mut sources,source(i)).is_ok && v::push<usize>(ref mut owners,0).is_ok);i+=1;}
 assert(io::write("bindings\n").is_ok);let original=collected(ref sources,ref owners);copy_and_admission(ref sources,ref owners,ref original);
 assert(io::write("context\n").is_ok);context_stamps(ref sources,ref original);
 assert(io::write("identities\n").is_ok);type_stamps(ref sources);
 assert(io::write("owner\n").is_ok);owner_boundary();
 assert(v::push<syntax::Parsed>(ref mut sources,source(1024)).is_ok && v::push<usize>(ref mut owners,0).is_ok);
 let resolved=v::empty<catalog::Binding>();let table=coordinate(0);var usize work=0;
 assert(switch(bindings::build(ref sources,ref owners,ref resolved,ref table,ref mut work)){case .ok(_):{false}case .err(e):{e.code==9 && work==0}});
 assert(io::write("ok\n").is_ok);return 0;
}
`
