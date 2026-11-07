module closure_value_source_policy;
import syntax as syntax;
import types as types;
import bodies as bodies;
import records as records;
import std::vec as v;
import std::bytes as bytes;

pub struct Problem{u32 code;usize start;usize end;}
fn Problem problem(u32 code,usize start,usize end){return Problem{.code,.start,.end};}
fn bool charge(ref mut usize work){if(*work>=1048576){return false;};*work=*work+1;return true;}
pub fn bool needed(ref v::Vec<types::Node> nodes){var usize i=0;while(i<v::len<types::Node>(nodes)){if(v::at<types::Node>(nodes,i).kind==17){return true;}i+=1;}return false;}
fn usize module_at(ref v::Vec<syntax::Declaration> declarations,usize start){var usize owner=0;var usize position=0;var usize i=0;while(i<v::len<syntax::Declaration>(declarations)){let d=*v::at<syntax::Declaration>(declarations,i);if(d.kind==1 && d.start<=start && (owner==0 || d.start>=position)){owner=i+1;position=d.start;}i+=1;}return owner;}
fn bool parameter(ref u8[] source,ref v::Vec<types::Node> nodes,syntax::Declaration d,types::Node n){
  if(n.start<d.start || n.end>d.end){return false;}var usize link=d.generic_parameters;var usize count=0;
  while(link!=0 && link<=v::len<types::Node>(nodes) && count<8){let item=*v::at<types::Node>(nodes,link-1);if(item.left==0 || item.left>=link || item.right>=link){return false;}let name=*v::at<types::Node>(nodes,item.left-1);if(bytes::equal(source[n.start..n.end],source[name.start..name.end])){return true;}link=item.right;count+=1;}return false;
}
fn usize nominal(ref u8[] source,ref v::Vec<syntax::Declaration> declarations,types::Node n){
  if(n.left!=0 && n.left<=v::len<syntax::Declaration>(declarations)){let d=*v::at<syntax::Declaration>(declarations,n.left-1);if(d.kind==4 || d.kind==6 || d.kind==11){return n.left;}}
  let owner=module_at(declarations,n.start);var usize i=0;while(i<v::len<syntax::Declaration>(declarations)){let d=*v::at<syntax::Declaration>(declarations,i);if((d.kind==4 || d.kind==6 || d.kind==11) && module_at(declarations,d.start)==owner && bytes::equal(source[n.start..n.end],source[d.name_start..d.name_end])){return i+1;}i+=1;}return 0;
}
fn Result<bool,u32> contains(ref u8[] source,ref v::Vec<syntax::Declaration> declarations,ref v::Vec<types::Node> nodes,ref v::Vec<records::Field> fields,syntax::Declaration template,usize root,ref mut v::Vec<usize> active,usize depth,ref mut usize work){
  if(root==0){return .ok(false);}if(depth>=64 || !charge(work)){return .err(4503);}if(root>v::len<types::Node>(nodes)){return .err(4502);}let n=*v::at<types::Node>(nodes,root-1);if(n.kind==17){return .ok(true);}
  if(n.kind==1){
    if(parameter(source,nodes,template,n)){return .ok(false);}let target=nominal(source,declarations,n);if(target==0){return .ok(false);}var usize i=0;while(i<v::len<usize>(active)){if(*v::at<usize>(active,i)==target){return .ok(false);}i+=1;}
    if(!v::push<usize>(active,target).is_ok){return .err(5);}let d=*v::at<syntax::Declaration>(declarations,target-1);i=0;while(i<d.field_count){if(d.field_start+i>=v::len<records::Field>(fields)){return .err(4502);}let f=*v::at<records::Field>(fields,d.field_start+i);let found=switch(contains(source,declarations,nodes,fields,template,f.type_root,active,depth+1,work)){case .ok(b):{b}case .err(code):{return .err(code);}};if(found){discard v::pop<usize>(active);return .ok(true);}i+=1;}discard v::pop<usize>(active);return .ok(false);
  }
  if(n.left>=root || n.right>=root){return .err(4502);}let left=switch(contains(source,declarations,nodes,fields,template,n.left,active,depth+1,work)){case .ok(b):{b}case .err(code):{return .err(code);}};if(left){return .ok(true);}return contains(source,declarations,nodes,fields,template,n.right,active,depth+1,work);
}
fn Problem typ(ref u8[] source,ref v::Vec<syntax::Declaration> declarations,ref v::Vec<types::Node> nodes,ref v::Vec<records::Field> fields,syntax::Declaration template,usize root,ref mut usize work){
  if(root==0){return problem(0,0,0);}var active=v::empty<usize>();let found=switch(contains(source,declarations,nodes,fields,template,root,ref mut active,0,work)){case .ok(b):{b}case .err(code):{return problem(code,template.start,template.end);}};if(!found){return problem(0,0,0);}let n=*v::at<types::Node>(nodes,root-1);return problem(4502,n.start,n.end);
}
// Inspect the selected written template before cloning can make it monomorphic.
// Unrequested generic declarations and neighboring scalar calls stay untouched.
pub fn Problem function(ref u8[] source,ref v::Vec<syntax::Declaration> declarations,ref v::Vec<types::Node> nodes,ref v::Vec<bodies::Entry> entries,ref v::Vec<records::Field> fields,syntax::Declaration template,ref mut usize work){
  let result=typ(source,declarations,nodes,fields,template,template.result_type,work);if(result.code!=0){return result;}
  var usize i=0;while(i<template.entry_count){let id=template.entry_start+i+1;var usize j=0;while(j<v::len<bodies::Entry>(entries)){if(!charge(work)){return problem(4503,template.start,template.end);}let e=*v::at<bodies::Entry>(entries,j);if(e.id==id){if(e.kind==1){let p=typ(source,declarations,nodes,fields,template,e.type_root,work);if(p.code!=0){return p;}}break;}j+=1;}i+=1;}return problem(0,0,0);
}
