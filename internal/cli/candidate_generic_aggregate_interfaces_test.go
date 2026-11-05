package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCandidateGenericAggregateInterfaces(t *testing.T) {
	const factory = `pub fn Holder<T> make<T>(T value){return Holder<T>{.payload=Cell<T>{.value=value}};}`
	const root = `module root;import maker as m;fn i32 main(){let value=m::make<i32>(42);discard value;return 0;}`
	maker := aggregateLayoutMaker + factory
	wire := candidateAggregateLayoutOracle(t,
		candidateAggregateLayoutFixture{name: "ordinary constructor", maker: aggregateLayoutMaker + `pub fn Cell<i32> direct(i32 value){return Cell<i32>{.value=value};}`},
		candidateAggregateLayoutFixture{name: "generic factory", maker: maker, root: root},
		candidateAggregateLayoutFixture{name: "extended factory argument", maker: maker + `pub fn Holder<f64> extra(f64 value){return make<f64>(value);}`},
		candidateAggregateLayoutFixture{name: "constructor in cleanup", maker: strings.Replace(maker, "cleanup{discard h::finish<T>(this.payload.value);}", "cleanup{let copied=Cell<T>{.value=this.payload.value};discard h::finish<T>(copied.value);}", 1), root: root},
	)
	raw, err := os.ReadFile(filepath.Join("..", "..", "examples", "lexer_candidate", "tests", "generic_aggregate_layouts.fy"))
	if err != nil {
		t.Fatal(err)
	}
	main := strings.Replace(string(raw), "module tests::generic_aggregate_layouts;", "module main;", 1)
	main = strings.Replace(main, "import module_unit_interfaces as interfaces;", "import module_unit_interfaces as interfaces;import module_unit_bind as binding;import module_unit_generics as generics;import module_unit_aggregate_plan as aggregate_plan;import module_ast_join as join;", 1)
	start := strings.Index(main, "fn u32 build_fields(")
	stop := strings.Index(main, "@effects(console) fn void verify_fields(")
	if start < 0 || stop < start {
		t.Fatal("missing layout probe field builder")
	}
	signature := main[start : start+strings.Index(main[start:], "{")+1]
	main = main[:start] + signature + aggregateInterfacesFieldBuilder + main[stop:] + aggregateInterfacesUnitGuards + aggregateInterfacesRefusalGuards + "\n@effects(console) fn i32 main(){return run();}\n"
	main = strings.Replace(main, "if(k==0){guards(ref metadata,ref sources);}", "if(k==0){guards(ref metadata,ref sources);interface_guards(ref metadata,ref sources);}", 1)
	candidateAtlasProbeSize(t, "module_unit_interfaces", main, 512<<10, func(t *testing.T, image string) {
		t.Setenv("PATH", "")
		if code, out := candidateAtlasProcess(t, image, wire); code != 0 || out != "" {
			t.Fatalf("aggregate Interfaces: %d %q", code, out)
		}
	}, "project_source_plan")
}

const aggregateInterfacesFieldBuilder = `
 let bound=switch(binding::bind(sources,metadata,0,v::len<syntax::Parsed>(sources))){case .ok(p):{p}case .err(p):{return p.code;}};
 let p=switch(interfaces::build(sources,metadata,ref bound)){case .ok(p):{p}case .err(p):{return p.code;}};
 let valid=interfaces::validate_specializations(ref p,sources,metadata);assert(valid.code==0);verify_constructions(ref p,sources);verify_units(metadata,sources);
 var usize i=types::count(records)+1;while(i<=interfaces::type_count(ref p)){assert(v::push<types::Type>(records,interfaces::type_at(ref p,i)).is_ok);i+=1;}
 i=0;var usize resources=0;
 while(i<interfaces::shape_count(ref p)){let s=interfaces::shape_at(ref p,i);assert(v::push<interfaces::Shape>(shapes,s).is_ok);let t=interfaces::type_at(ref p,s.type);if(t.kind==21){resources+=1;var usize matched=0;var usize j=1;let plan=interfaces::specializations(ref p);while(j<=generics::cleanup_count(plan)){let c=generics::cleanup_at(plan,j);if(c.type==s.type){assert(c.module_index==s.module_index && c.declaration==s.declaration && c.argument_count==1);let tuple=interfaces::type_at(ref p,t.first);assert(generics::argument_at(plan,c.argument_start)==tuple.first);matched+=1;}j+=1;}assert(matched==1);}
  var usize j=0;while(j<s.field_count){let name=interfaces::field_name(ref p,s.field_start+j);assert(switch(interfaces::field(ref p,s.type,s.module_index,name)){case .ok(n):{n==s.field_start+j+1}case .err(_):{false}});if(t.kind==21){assert(switch(interfaces::field(ref p,s.type,(s.module_index+1)%v::len<syntax::Parsed>(sources),name)){case .ok(_):{false}case .err(code):{code==6201}});}j+=1;}i+=1;
 }
 assert(generics::cleanup_count(interfaces::specializations(ref p))==resources);
 i=0;while(i<interfaces::field_count(ref p)){assert(v::push<interfaces::Field>(fields,interfaces::field_at(ref p,i)).is_ok);i+=1;}
 assert(bytes::append(names,bytes::view(interfaces::name_records(ref p))).is_ok);return 0;
}
`

const aggregateInterfacesUnitGuards = `
fn void verify_constructions(ref interfaces::Interfaces p,ref v::Vec<syntax::Parsed> sources){
 let plan=interfaces::specializations(p);var usize module_index=interfaces::first(p);var usize offset=0;
 while(module_index<interfaces::stop(p)){let source=v::at<syntax::Parsed>(sources,module_index);var usize i=1;while(i<=syntax::expression_count(source)){let c=generics::call_at(plan,offset+i);if(c.aggregate_type!=0){assert(c.instance==0 && c.intrinsic==0 && !c.deferred);var usize matched=0;var usize j=0;while(j<interfaces::body_type_count(p)){let r=interfaces::body_type_at(p,j);if(r.kind==2 && r.context_kind==3 && r.module_index==module_index && syntax::expression_at(source,r.source_index).left==i){assert(r.type==c.aggregate_type);matched+=1;}j+=1;}assert(matched==1);}i+=1;}offset+=syntax::expression_count(source);module_index+=1;}
 var u32 kind=1;while(kind<=2){var usize i=0;let count=if(kind==1){generics::dependency_count(plan)}else{generics::cleanup_dependency_count(plan)};
  while(i<count){let d=if(kind==1){generics::dependency_at(plan,i)}else{generics::cleanup_dependency_at(plan,i)};if(d.aggregate_type!=0){let module_index=if(kind==1){generics::at(plan,d.owner).module_index}else{generics::cleanup_at(plan,d.owner).module_index};let source=v::at<syntax::Parsed>(sources,module_index);var usize matched=0;var usize j=0;while(j<interfaces::body_type_count(p)){let r=interfaces::body_type_at(p,j);if(r.kind==2 && r.context_kind==kind && r.context==d.owner && r.module_index==module_index && syntax::expression_at(source,r.source_index).left==d.source_index){assert(r.type==d.aggregate_type);matched+=1;}j+=1;}assert(matched==1);}i+=1;
  }kind+=1;
 }
}
fn void verify_units(ref signatures::Program metadata,ref v::Vec<syntax::Parsed> sources){
 var usize root=0;while(root<v::len<syntax::Parsed>(sources)){let source=v::at<syntax::Parsed>(sources,root);let d=syntax::declaration_at(source,0);if(bytes::equal(syntax::source_view(source)[d.name_start..d.name_end],b"root")){break;}root+=1;}assert(root<v::len<syntax::Parsed>(sources));
 let raw=syntax::source_view(v::at<syntax::Parsed>(sources,root));let factory=bytes::equal(raw,b"module root;import maker as m;fn i32 main(){let value=m::make<i32>(42);discard value;return 0;}");let bound=switch(binding::bind(sources,metadata,root,root+1)){case .ok(p):{p}case .err(_):{assert(false);return;}};
 var usize mode=0;while(mode<2){let seeds=v::empty<binding::Coordinate>();let unit=switch(if(mode==0){interfaces::build(sources,metadata,ref bound)}else{interfaces::build_unit_seeded(sources,metadata,ref bound,ref seeds)}){case .ok(p):{p}case .err(_):{assert(false);return;}};
  assert(interfaces::validate_specializations(ref unit,sources,metadata).code==0);verify_constructions(ref unit,sources);let plan=interfaces::specializations(ref unit);assert(interfaces::shape_count(ref unit)==if(factory){2}else{0});assert(generics::cleanup_count(plan)==if(factory){1}else{0});
  if(factory){assert(generics::count(plan)==2);var usize constructors=0;var usize i=0;while(i<interfaces::body_type_count(ref unit)){let r=interfaces::body_type_at(ref unit,i);if(r.kind==2 && r.context_kind==1){constructors+=1;}i+=1;}assert(constructors==2);}
  mode+=1;
 }
}
`

const aggregateInterfacesRefusalGuards = `
fn void interface_guards(ref signatures::Program metadata,ref v::Vec<syntax::Parsed> sources){
 assert(switch(join::join_range(sources,0,v::len<syntax::Parsed>(sources))){case .ok(_):{false}case .err(p):{p.code==4501}});
 var graph=graph(metadata);var usize id=0;var usize i=1;while(i<=types::count(ref graph)){if(types::at(ref graph,i).kind==21){id=i;break;}i+=1;}assert(id!=0);let owner=types::at(ref graph,id);var usize work=0;
 let args=switch(aggregate_plan::arguments(metadata,ref graph,id,ref mut work)){case .ok(p):{p}case .err(_):{assert(false);return;}};assert(v::len<usize>(ref args)==1);work=1048576;assert(switch(aggregate_plan::arguments(metadata,ref graph,id,ref mut work)){case .ok(_):{false}case .err(code):{code==4503}});
 let source=v::at<syntax::Parsed>(sources,owner.module_index);var usize scalar_root=0;var usize composite_root=0;i=1;while(i<=syntax::type_count(source)){let n=syntax::type_at(source,i);let d=syntax::declaration_at(source,owner.declaration-1);if(n.start>=d.start && n.end<=d.end){if(n.kind==1 && bytes::equal(syntax::type_text(source,i),b"T")){scalar_root=i;}if(n.kind==6){composite_root=i;}}i+=1;}assert(scalar_root!=0 && composite_root!=0);
 let scalar=signatures::resolve_argument(metadata,source,owner.module_index,owner.declaration,scalar_root,ref args,ref mut graph);assert(scalar.problem.code==0 && scalar.type==*v::at<usize>(ref args,0));let before=types::count(ref graph);assert(signatures::resolve_argument(metadata,source,owner.module_index,owner.declaration,composite_root,ref args,ref mut graph).problem.code==4502 && types::count(ref graph)==before);
 var wrong=v::empty<usize>();assert(v::push<usize>(ref mut wrong,0).is_ok);work=0;assert(switch(aggregate_plan::context(metadata,ref graph,owner.module_index,owner.declaration,ref wrong,ref mut work)){case .ok(_):{false}case .err(code):{code==4502}});
 let empty=v::empty<usize>();work=0;assert(switch(aggregate_plan::construction(metadata,v::at<syntax::Parsed>(sources,0),0,3,ref empty,1,false,ref graph,ref mut work)){case .ok(_):{false}case .err(code):{code==12}});
 var changed=v::empty<syntax::Parsed>();i=0;while(i<v::len<syntax::Parsed>(sources)){var raw=bytes::empty();assert(bytes::append(ref mut raw,syntax::source_view(v::at<syntax::Parsed>(sources,i))).is_ok);if(i==owner.module_index){assert(bytes::append(ref mut raw,b"\n").is_ok);}assert(v::push<syntax::Parsed>(ref mut changed,parsed(bytes::view(ref raw))).is_ok);i+=1;}let bound=switch(binding::bind(sources,metadata,0,v::len<syntax::Parsed>(sources))){case .ok(p):{p}case .err(_):{assert(false);return;}};
 assert(switch(interfaces::build(ref changed,metadata,ref bound)){case .ok(_):{false}case .err(p):{p.code==12}});
}
`
