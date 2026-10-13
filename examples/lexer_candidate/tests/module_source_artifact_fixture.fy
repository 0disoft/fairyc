module tests::module_source_artifact_fixture;
import generic_package_bindings as bindings;
import module_public_type_text as coordinates;
import module_catalog as catalog;
import syntax as syntax;
import scanner as scanner;
import std::bytes as bytes;
import std::vec as v;
pub resource Input{v::Vec<syntax::Parsed> sources;v::Vec<usize> owners;v::Vec<catalog::Binding> imports;coordinates::Coordinates coordinates;cleanup{}}
pub fn ref v::Vec<syntax::Parsed> sources(ref Input p){return ref p.sources;}
pub fn ref v::Vec<usize> owners(ref Input p){return ref p.owners;}
pub fn ref v::Vec<catalog::Binding> imports(ref Input p){return ref p.imports;}
pub fn ref coordinates::Coordinates coordinate_table(ref Input p){return ref p.coordinates;}
fn ref u8[] text(usize i,usize mode){if(i==0){if(mode==2){return b"module root;import common as a;import common as a0;pub fn T id<T>(T value){return other;}";}if(mode==6){return b"module renamed;import common as a;import common as a0;pub fn T id<T>(T value){return value;}";}return b"module root;import common as a;import common as a0;pub fn T id<T>(T value){return value;}";}if(i==1){return b"module common;pub fn T first<T>(T value){return value;}";}return b"module common;pub fn T second<T>(T value){return value;}";}
fn syntax::Parsed parsed(ref u8[] raw){let b=switch(bytes::copy(raw)){case .ok(p):{p}case .err(_):{assert(false);return parsed(raw);}};let s=switch(scanner::collect(b,false,65536,1048576,256,16777216)){case .ok(p):{p}case .err(_):{assert(false);return parsed(raw);}};let p=switch(syntax::parse(s,512)){case .ok(p):{p}case .err(_):{assert(false);return parsed(raw);}};assert(syntax::problem_count(ref p)==0);return p;}
pub fn Input input(usize mode){var v::Vec<syntax::Parsed> sources=v::empty<syntax::Parsed>();var v::Vec<usize> owners=v::empty<usize>();var usize i=0;while(i<if(mode==5){2}else{3}){let usize old=if(mode==1){if(i==0){2}else{i-1}}else{i};assert(v::push<syntax::Parsed>(ref mut sources,parsed(text(old,mode))).is_ok && v::push<usize>(ref mut owners,if(mode==1){old+7}else{old}).is_ok);i+=1;}
    var bytes::Buffer names=bytes::empty();var v::Vec<coordinates::Coordinate> rows=v::empty<coordinates::Coordinate>();i=0;while(i<3){let first=bytes::len(ref names);assert(bytes::append(ref mut names,b"fypc1-sha256-").is_ok);let digit=if(i==0){if(mode==3){b"4"}else{b"1"}}else if(i==1){b"2"}else{b"3"};var usize j=0;while(j<64){assert(bytes::append(ref mut names,digit).is_ok);j+=1;}assert(v::push<coordinates::Coordinate>(ref mut rows,coordinates::Coordinate{.owner=if(mode==1){i+7}else{i},.start=first,.end=bytes::len(ref names)}).is_ok);i+=1;}
    var v::Vec<catalog::Binding> imports=v::empty<catalog::Binding>();i=2;while(i<=3){assert(v::push<catalog::Binding>(ref mut imports,catalog::Binding{.source=if(mode==1){1}else{0},.declaration=i,.target=if(mode==1 || mode==4){2}else{1}}).is_ok);i+=1;}return Input{.sources,.owners,.imports,.coordinates=coordinates::assemble(names,rows)};
}
pub fn bindings::Bindings original(){let p=input(0);var usize work=0;return switch(bindings::build(sources(ref p),owners(ref p),imports(ref p),coordinate_table(ref p),ref mut work)){case .ok(p):{p}case .err(_):{assert(false);return original();}};}
