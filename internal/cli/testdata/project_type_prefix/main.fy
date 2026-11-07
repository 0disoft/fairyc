module main;
import syntax as syntax;
import scanner as scanner;
import std::bytes as bytes;
import std::io as io;

// Exact original full-map build, renamed and with prefix=0 for its full maps.
fn Result<Plan,clean::Problem> baseline_build(ref signatures::Program metadata,ref v::Vec<types::Graph> inputs){
    let usize count=v::len<types::Graph>(inputs);if(count==0 || count>256){return .err(fail(if(count>256){1002}else{1001}));}
    var usize work=0;var v::Vec<types::Type> graph=v::empty<types::Type>();var v::Vec<Unit> units=v::empty<Unit>();var v::Vec<usize> maps=v::empty<usize>();var v::Vec<usize> table=v::empty<usize>();var usize i=0;
    while(i<8192){if(!budget::charge(ref mut work,1)){return .err(fail(1002));}if(!v::push<usize>(ref mut table,0).is_ok){return .err(fail(1003));}i+=1;}
    var usize unit=0;while(unit<count){let input=v::at<types::Graph>(inputs,unit);let usize nt=types::graph_count(input);if(nt<signatures::type_count(metadata) || nt>4096){return .err(fail(if(nt>4096){1002}else{1001}));}
        var v::Vec<types::Type> copy=v::empty<types::Type>();i=1;while(i<=nt){if(!budget::charge(ref mut work,1)){return .err(fail(1002));}let t=types::graph_at(input,i);if(i<=signatures::type_count(metadata) && !identity::same(t,signatures::type_at(metadata,i))){return .err(fail(1001));}if(!v::push<types::Type>(ref mut copy,t).is_ok){return .err(fail(1003));}i+=1;}
        // Revalidate input canonicality and authentic nominal/SDK coordinates
        // against the shared catalog before using any edge as a map index.
        let validated=switch(signatures::own_types(metadata,copy)){case .ok(p):{p}case .err(code):{return .err(fail(translated(code)));}};
        let usize start=v::len<usize>(ref maps);i=1;while(i<=nt){if(!budget::charge(ref mut work,1)){return .err(fail(1002));}var t=types::graph_at(ref validated,i);
            if(t.first!=0){t.first=*v::at<usize>(ref maps,start+t.first-1);}if(t.kind==6 || t.kind==22 && t.second!=0){t.second=*v::at<usize>(ref maps,start+t.second-1);}
            // Array second is a length; pointer scalar carries ABI bits.
            let id=switch(intern(ref mut graph,ref mut table,t,ref mut work)){case .ok(id):{id}case .err(p):{return .err(p);}};
            if(!v::push<usize>(ref mut maps,id).is_ok){return .err(fail(1003));}i+=1;
        }if(!v::push<Unit>(ref mut units,Unit{.start,.count=nt}).is_ok){return .err(fail(1003));}unit+=1;
    }
    let owned=switch(signatures::own_types(metadata,graph)){case .ok(p):{p}case .err(code):{return .err(fail(translated(code)));}};
    return .ok(Plan{.graph=owned,.units,.maps,.prefix=0,.work});
}
fn usize baseline_mapped(ref Plan p,usize unit,usize id){if(id==0 || unit>=unit_count(p)){return 0;}let u=*v::at<Unit>(ref p.units,unit);return if(id>u.count){0}else{*v::at<usize>(ref p.maps,u.start+id-1)};}
fn types::Type node(u32 kind,u32 scalar,usize module_index,usize declaration,usize first,usize second){return types::Type{.kind,.scalar,.module_index,.declaration,.first,.second};}
fn void text(ref mut bytes::Buffer raw,ref u8[] part){assert(bytes::append(raw,part).is_ok);}
fn void number(ref mut bytes::Buffer raw,usize n){assert(bytes::append_usize(raw,n).is_ok);}
fn syntax::Parsed parsed(bytes::Buffer raw){let tokens=switch(scanner::collect(raw,false,65536,1048576,256,16777216)){case .ok(p):{p}case .err(_):{assert(false);return parsed(bytes::empty());}};return switch(syntax::parse(tokens,512)){case .ok(p):{assert(syntax::problem_count(ref p)==0);p}case .err(_):{assert(false);parsed(bytes::empty())}};}
fn signatures::Program catalog(usize arrays,bool empty){
 var v::Vec<syntax::Parsed> sources=v::empty<syntax::Parsed>();var bytes::Buffer raw=bytes::empty();text(ref mut raw,b"module root;struct Node{i32 n;}");var usize i=0;
 if(!empty){text(ref mut raw,b"fn i32 zero(){return 0;}");}
 while(i<arrays){text(ref mut raw,b"fn i32 f");number(ref mut raw,i);text(ref mut raw,b"(");var usize j=0;
  while(j<8 && i+j<arrays){if(j!=0){text(ref mut raw,b",");}text(ref mut raw,b"i32[");number(ref mut raw,i+j+1);text(ref mut raw,b"] a");number(ref mut raw,j);j+=1;}text(ref mut raw,b"){return 0;}");i+=j;
 }assert(v::push<syntax::Parsed>(ref mut sources,parsed(raw)).is_ok);var bytes::Buffer other=bytes::empty();text(ref mut other,b"module other;pub struct Node{u8 n;}");assert(v::push<syntax::Parsed>(ref mut sources,parsed(other)).is_ok);
 return switch(signatures::build(ref sources)){case .ok(p):{assert(signatures::type_count(ref p)==if(empty){0}else{arrays+1});p}case .err(_):{assert(false);catalog(0,true)}};
}
fn v::Vec<types::Type> prefix_records(ref signatures::Program metadata){var v::Vec<types::Type> rows=v::empty<types::Type>();var usize i=1;while(i<=signatures::type_count(metadata)){assert(v::push<types::Type>(ref mut rows,signatures::type_at(metadata,i)).is_ok);i+=1;}return rows;}
fn types::Graph owned(ref signatures::Program metadata,v::Vec<types::Type> rows){return switch(signatures::own_types(metadata,rows)){case .ok(p):{p}case .err(_):{assert(false);owned(metadata,prefix_records(metadata))}};}
fn types::Graph input(ref signatures::Program metadata,usize unit,bool suffix){
 var rows=prefix_records(metadata);if(suffix){let scalar=types::canonical(ref mut rows,node(1,1,0,0,0,0));var usize nominal=0;var usize pointer=0;
  if(unit%2!=0){nominal=types::canonical(ref mut rows,node(2,0,unit%2,2,0,0));pointer=types::canonical(ref mut rows,node(12,1,0,0,nominal,0));}
  let array=types::canonical(ref mut rows,node(16,0,0,0,scalar,1024+unit));let option=types::canonical(ref mut rows,node(5,0,0,0,array,0));
  if(unit%2==0){nominal=types::canonical(ref mut rows,node(2,0,unit%2,2,0,0));pointer=types::canonical(ref mut rows,node(12,0,0,0,nominal,0));}
  assert(types::canonical(ref mut rows,node(8,0,0,0,array,0))!=0 && types::canonical(ref mut rows,node(6,0,0,0,option,pointer))!=0);
  assert(types::canonical(ref mut rows,node(5,0,0,0,scalar,0))!=0 && types::canonical(ref mut rows,node(6,0,0,0,pointer,scalar))!=0);
 }return owned(metadata,rows);
}
fn void same(ref Plan a,ref Plan b,ref v::Vec<types::Graph> inputs){
 assert(type_count(a)==type_count(b) && unit_count(a)==unit_count(b) && aggregates(a)==aggregates(b));var usize i=1;
 while(i<=type_count(a)){assert(identity::same(type_at(a,i),type_at(b,i)));i+=1;}
 var usize unit=0;while(unit<unit_count(a)){let original=v::at<types::Graph>(inputs,unit);let count=types::graph_count(original);assert(mapped(a,unit,0)==0 && mapped(a,unit,count+1)==0 && mapped(a,unit,4097)==0);i=1;
  while(i<=count){let id=mapped(a,unit,i);assert(id==baseline_mapped(b,unit,i) && id!=0);var expected=types::graph_at(original,i);
   if(expected.first!=0){expected.first=mapped(a,unit,expected.first);}if(expected.kind==6 || expected.kind==22 && expected.second!=0){expected.second=mapped(a,unit,expected.second);}assert(identity::same(expected,type_at(a,id)));i+=1;
  }unit+=1;
 }assert(mapped(a,unit_count(a),1)==0 && mapped(a,unit_count(a)+1,0)==0);
}
fn v::Vec<types::Type> snapshot(ref v::Vec<types::Graph> inputs){var v::Vec<types::Type> rows=v::empty<types::Type>();var usize unit=0;while(unit<v::len<types::Graph>(inputs)){let input=v::at<types::Graph>(inputs,unit);var usize i=1;while(i<=types::graph_count(input)){assert(v::push<types::Type>(ref mut rows,types::graph_at(input,i)).is_ok);i+=1;}unit+=1;}return rows;}
fn void unchanged(ref v::Vec<types::Graph> inputs,ref v::Vec<types::Type> before){var usize at=0;var usize unit=0;while(unit<v::len<types::Graph>(inputs)){let input=v::at<types::Graph>(inputs,unit);var usize i=1;while(i<=types::graph_count(input)){assert(at<v::len<types::Type>(before) && identity::same(types::graph_at(input,i),*v::at<types::Type>(before,at)));at+=1;i+=1;}unit+=1;}assert(at==v::len<types::Type>(before));}
@effects(console) fn void report(usize units,usize prefix,usize baseline_work,usize changed_work,usize baseline_maps,usize changed_maps){discard io::write("units=");discard io::write_usize(units);discard io::write(" prefix=");discard io::write_usize(prefix);discard io::write(" work=");discard io::write_usize(baseline_work);discard io::write("->");discard io::write_usize(changed_work);discard io::write(" maps=");discard io::write_usize(baseline_maps);discard io::write("->");discard io::write_usize(changed_maps);discard io::write(" map_payload_saved=");discard io::write_usize(8*(baseline_maps-changed_maps));discard io::write("\n");}
@effects(console) fn void run(usize arrays,usize units,bool suffix,bool empty){
 let metadata=catalog(arrays,empty);var v::Vec<types::Graph> inputs=v::empty<types::Graph>();var usize i=0;while(i<units){assert(v::push<types::Graph>(ref mut inputs,input(ref metadata,i,suffix)).is_ok);i+=1;}
 let before=snapshot(ref inputs);let old=switch(baseline_build(ref metadata,ref inputs)){case .ok(p):{p}case .err(_):{assert(false);return;}};
 let current=switch(build(ref metadata,ref inputs)){case .ok(p):{p}case .err(_):{assert(false);return;}};same(ref current,ref old,ref inputs);let prefix=signatures::type_count(ref metadata);
 assert(current.prefix==prefix && v::len<usize>(ref old.maps)-v::len<usize>(ref current.maps)==units*prefix && work(ref current)<=work(ref old) && work(ref current)<=1048576);
 if(units>1 && prefix!=0){assert(work(ref old)-work(ref current)>=2*(units-1)*prefix);}
 let again=switch(build(ref metadata,ref inputs)){case .ok(p):{p}case .err(_):{assert(false);return;}};same(ref again,ref old,ref inputs);assert(work(ref again)==work(ref current));unchanged(ref inputs,ref before);
 report(units,prefix,work(ref old),work(ref current),v::len<usize>(ref old.maps),v::len<usize>(ref current.maps));
}
fn clean::Problem refused(Result<Plan,clean::Problem> result){return switch(result){case .ok(_):{assert(false);fail(0)}case .err(e):{e}};}
fn void same_error(clean::Problem a,clean::Problem b,u32 code){assert(a.code==code && a.code==b.code && a.module_index==b.module_index && a.start==b.start && a.end==b.end);}
fn void malformed(){
 let metadata=catalog(8,false);var u32 mode=0;while(mode<5){var v::Vec<types::Graph> inputs=v::empty<types::Graph>();assert(v::push<types::Graph>(ref mut inputs,input(ref metadata,0,true)).is_ok && v::push<types::Graph>(ref mut inputs,input(ref metadata,1,true)).is_ok);var bad=prefix_records(ref metadata);
  if(mode==0){let first=v::at_mut<types::Type>(ref mut bad,0);*first=node(1,6,0,0,0,0);}
  else if(mode==1){let duplicate=*v::at<types::Type>(ref bad,0);assert(v::push<types::Type>(ref mut bad,duplicate).is_ok);}
  else if(mode==2){let forward=v::len<types::Type>(ref bad)+1;assert(v::push<types::Type>(ref mut bad,node(5,0,0,0,forward,0)).is_ok);}
  else if(mode==3){assert(v::push<types::Type>(ref mut bad,node(2,0,1,999,0,0)).is_ok);}
  else{discard v::truncate<types::Type>(ref mut bad,1);assert(v::len<types::Type>(ref bad)==1);}
  assert(v::push<types::Graph>(ref mut inputs,types::prefix_probe_raw(bad)).is_ok);let before=snapshot(ref inputs);same_error(refused(build(ref metadata,ref inputs)),refused(baseline_build(ref metadata,ref inputs)),1001);unchanged(ref inputs,ref before);mode+=1;
 }
 var v::Vec<types::Graph> empty=v::empty<types::Graph>();same_error(refused(build(ref metadata,ref empty)),refused(baseline_build(ref metadata,ref empty)),1001);
 var usize i=0;while(i<257){assert(v::push<types::Graph>(ref mut empty,input(ref metadata,i,false)).is_ok);i+=1;}same_error(refused(build(ref metadata,ref empty)),refused(baseline_build(ref metadata,ref empty)),1002);
}
@effects(console) fn i32 main(){run(0,8,true,true);run(32,1,true,false);run(32,256,false,false);run(128,64,true,false);run(512,64,true,false);malformed();discard io::write("mapping=equal edges=equal malformed=refused inputs=unchanged\n");return 0;}
