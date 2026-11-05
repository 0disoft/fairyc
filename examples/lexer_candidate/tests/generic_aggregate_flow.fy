module tests::generic_aggregate_flow;
import module_unit_checked_bodies as checking;
import module_unit_moves as moves;
import module_unit_cleanup_policy as policy;
import module_unit_values as values;
import module_unit_value_model as model;
import syntax as syntax;
import std::bytes as bytes;
import std::vec as v;
import std::io as io;

@effects(console) pub fn void verify(ref checking::Plan owner,ref v::Vec<syntax::Parsed> sources){
 let checked=checking::checked_values(owner);let moved=checking::checked_moves(owner);require(moves::count(moved)==0 && policy::count(checking::cleanup_policy(owner))==0,b"flow diagnostics");
 var usize work=0;require(checking::validate_sources(owner,sources,ref mut work).code==0,b"fresh checked source");work=16777216;require(checking::validate_sources(owner,sources,ref mut work).code==10,b"source budget");
 var usize i=0;while(i<values::count(checked)){let context=values::at(checked,i);let frame=model::frame(context);let source=v::at<syntax::Parsed>(sources,frame.module_index);let declaration=syntax::declaration_at(source,frame.declaration-1);let name=syntax::source_view(source)[declaration.name_start..declaration.name_end];var usize count=0;var usize first=0;var usize second=0;var usize j=0;
  while(j<moves::cleanup_count(moved)){let fact=moves::cleanup_at(moved,j);if(fact.frame==i+1){assert(fact.kind==5 && fact.type!=0 && fact.start<fact.end && fact.end<=syntax::source_view(source).len && fact.before==1 && fact.after==1);count+=1;if(bytes::equal(name,b"cleanup_pair")){assert(fact.binding!=0 && fact.expression==0 && fact.exit==1 && values::type_at(checked,fact.type).kind==21);let local=model::local_at(context,fact.binding);let text=syntax::source_view(source)[local.start..local.end];if(bytes::equal(text,b"first")){first+=1;}else{assert(bytes::equal(text,b"second"));second+=1;}}}j+=1;}
  if(bytes::equal(name,b"cleanup_pair")){require(count==2 && first==1 && second==1,b"two cleanup facts");}if(bytes::equal(name,b"hold") || bytes::equal(name,b"transfer")){require(count==0,b"transferred owner cleanup");}if(bytes::equal(name,b"release_holder") || bytes::equal(name,b"consume_echo")){require(count==1,b"parameter cleanup");}i+=1;
 }
}
@effects(console) fn void require(bool condition,ref u8[] label){if(!condition){discard io::write_stderr_bytes(label);assert(false);}}
