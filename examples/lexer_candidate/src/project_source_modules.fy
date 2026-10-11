module project_source_modules;
import project_source_set as sources;
import project_module_path as paths;
import project_path as work;
import project_manifest_values as rules;
import scanner as scanner;
import syntax as syntax;
import std::bytes as bytes;
import std::vec as v;

struct Span{usize start;usize end;}
resource Parts{v::Vec<syntax::Parsed> parsed;bytes::Buffer names;v::Vec<Span> modules;v::Vec<usize> order;cleanup{}}
pub resource Modules{sources::Sources sources;Parts contents;cleanup{}}
// 1 invalid source-module path, 2 lexical diagnostics, 3 syntax diagnostics,
// 4 missing/multiple module, 5 mismatched declared module, 6 duplicate,
// 7 allocation, 8 implementation/work bound. Spans are original source bytes.
pub struct Problem{u32 code;u32 diagnostic;i32 status;usize file;usize start;usize end;}
fn Problem fail(u32 code,u32 diagnostic,i32 status,usize file,usize start,usize end){return Problem{.code,.diagnostic,.status,.file,.start,.end};}
pub resource Error{sources::Sources sources;Problem problem;cleanup{}}
pub fn Problem error(ref Error error){return error.problem;}
pub fn ref sources::Sources error_sources(ref Error error){return ref error.sources;}
pub fn usize count(ref Modules modules){return v::len<syntax::Parsed>(ref modules.contents.parsed);}
pub fn ref u8[] name(ref Modules modules,usize index){if(index>=count(modules)){return bytes::view(ref modules.contents.names)[..0];}let Span span=*v::at<Span>(ref modules.contents.modules,index);return bytes::view(ref modules.contents.names)[span.start..span.end];}
pub fn ref u8[] path(ref Modules modules,usize index){return sources::path(ref modules.sources,index);}
pub fn ref syntax::Parsed parsed(ref Modules modules,usize index){assert(index<count(modules));return v::at<syntax::Parsed>(ref modules.contents.parsed,index);}
pub fn ref v::Vec<syntax::Parsed> parser_inputs(ref Modules modules){return ref modules.contents.parsed;}
pub fn ref sources::Sources source_set(ref Modules modules){return ref modules.sources;}
fn ref u8[] key(ref bytes::Buffer names,ref v::Vec<Span> spans,usize index){let Span span=*v::at<Span>(spans,index);return bytes::view(names)[span.start..span.end];}
fn void put(ref mut v::Vec<usize> values,usize index,usize value){let ref mut usize slot=v::at_mut<usize>(values,index);*slot=value;}
// Compare the already verified module path tokens. Trivia never becomes a
// spelling byte and no second lexer or temporary canonical-name buffer is used.
fn Result<bool,u32> matches_declaration(ref syntax::Parsed tree,syntax::Declaration declaration,ref u8[] expected,ref mut usize meter){
    var usize low=0;var usize high=syntax::token_count(tree);while(low<high){if(!work::charge(meter,1)){return .err(8);}let usize middle=low+(high-low)/2;if(syntax::token_info(tree,middle).start<declaration.name_start){low=middle+1;}else{high=middle;}}
    var usize matched=0;while(low<syntax::token_count(tree)){let token=syntax::token_info(tree,low);if(token.start>=declaration.name_end){break;}if(!work::charge(meter,1)){return .err(8);}if(token.end>declaration.name_end){return .err(3);}if(token.kind!=2){let ref u8[] value=syntax::source_view(tree)[token.start..token.end];if(!work::charge(meter,value.len+1)){return .err(8);}if(value.len>expected.len-matched || !bytes::equal(value,expected[matched..matched+value.len])){return .ok(false);}matched+=value.len;}low+=1;}return .ok(matched==expected.len);
}
fn u32 sorted(ref bytes::Buffer names,ref v::Vec<Span> spans,ref mut v::Vec<usize> output,ref mut usize meter){
    let usize count=v::len<Span>(spans);var v::Vec<usize> scratch=v::empty<usize>();var usize index=0;while(index<count){if(!v::push<usize>(output,index).is_ok || !v::push<usize>(ref mut scratch,0).is_ok){return 7;}index+=1;}
    var usize width=1;while(width<count){var usize start=0;while(start<count){let usize middle=if(width<count-start){start+width}else{count};let usize end=if(width<count-middle){middle+width}else{count};var usize left=start;var usize right=middle;var usize out=start;
        while(out<end){var bool take_left=right==end;if(left<middle && right<end){let ref u8[] a=key(names,spans,*v::at<usize>(output,left));let ref u8[] b=key(names,spans,*v::at<usize>(output,right));if(!work::charge(meter,a.len+b.len+1)){return 8;}take_left=rules::before(a,b);}put(ref mut scratch,out,if(take_left){*v::at<usize>(output,left)}else{*v::at<usize>(output,right)});if(take_left){left+=1;}else{right+=1;}out+=1;
        }start=end;
    }index=0;while(index<count){put(output,index,*v::at<usize>(ref scratch,index));index+=1;}width*=2;}return 0;
}
// Returns the original source-file index plus one; zero means absent.
pub fn usize find(ref Modules modules,ref u8[] name){var usize low=0;var usize high=count(modules);while(low<high){let usize middle=low+(high-low)/2;let usize index=*v::at<usize>(ref modules.contents.order,middle);let ref u8[] candidate=key(ref modules.contents.names,ref modules.contents.modules,index);if(bytes::equal(candidate,name)){return index+1;}if(rules::before(candidate,name)){low=middle+1;}else{high=middle;}}return 0;}
fn Result<Parts,Problem> build(ref sources::Sources input,ref mut usize meter){
    let usize count=sources::count(input);if(count==0 || count>4096){return .err(fail(8,0,0,0,0,0));}var v::Vec<syntax::Parsed> parsed=v::empty<syntax::Parsed>();var bytes::Buffer names=bytes::empty();var v::Vec<Span> spans=v::empty<Span>();var usize index=0;
    while(index<count){let ref u8[] path=sources::path(input,index);
        let ref u8[] original=sources::source(input,index);if(original.len>8388608 || !work::charge(meter,1)){return .err(fail(8,0,0,index,0,0));}let bytes::Buffer owned=switch(bytes::copy(original)){case .ok(value):{value}case .err(_):{return .err(fail(7,0,3,index,0,0));}};
        let scanner::Stream tokens=switch(scanner::collect(owned,false,8388608,1048576,256,16777216)){case .ok(value):{value}case .err(code):{return .err(fail(if(code==3){7}else{8},0,code,index,0,0));}};
        if(scanner::diagnostic_count(ref tokens)>0){let info=scanner::diagnostic_info(ref tokens,0);return .err(fail(if(info.code==10){8}else{2},info.code,0,index,info.start,info.end));}
        let syntax::Parsed tree=switch(syntax::parse(tokens,512)){case .ok(value):{value}case .err(code):{return .err(fail(if(code==3){7}else{8},0,code,index,0,0));}};
        if(syntax::problem_count(ref tree)>0){let problem=syntax::problem_at(ref tree,0);return .err(fail(3,0,0,index,problem.start,problem.end));}
        if(syntax::module_count(ref tree)!=1 || syntax::declaration_count(ref tree)==0){return .err(fail(4,0,0,index,0,original.len));}let declaration=syntax::declaration_at(ref tree,0);if(declaration.kind!=1){return .err(fail(4,0,0,index,declaration.start,declaration.end));}
        let bytes::Buffer expected=switch(paths::expected(path,meter)){case .ok(value):{value}case .err(code):{return .err(fail(if(code==2){7}else if(code==3){8}else{1},if(code==1){2}else{0},0,index,declaration.name_start,declaration.name_end));}};
        let bool matches=switch(matches_declaration(ref tree,declaration,bytes::view(ref expected),meter)){case .ok(value):{value}case .err(code):{return .err(fail(code,0,0,index,declaration.name_start,declaration.name_end));}};
        if(!matches){return .err(fail(5,2,0,index,declaration.name_start,declaration.name_end));}
        let usize start=bytes::len(ref names);if(!bytes::append(ref mut names,bytes::view(ref expected)).is_ok || !v::push<Span>(ref mut spans,Span{.start,.end=bytes::len(ref names)}).is_ok){return .err(fail(7,0,3,index,0,0));}if(!v::push<syntax::Parsed>(ref mut parsed,tree).is_ok){return .err(fail(7,0,3,index,0,0));}index+=1;
    }
    var v::Vec<usize> order=v::empty<usize>();let u32 code=sorted(ref names,ref spans,ref mut order,meter);if(code!=0){return .err(fail(code,0,0,0,0,0));}index=1;while(index<count){let usize before=*v::at<usize>(ref order,index-1);let usize after=*v::at<usize>(ref order,index);if(!work::charge(meter,key(ref names,ref spans,before).len+key(ref names,ref spans,after).len+1)){return .err(fail(8,0,0,after,0,0));}if(bytes::equal(key(ref names,ref spans,before),key(ref names,ref spans,after))){return .err(fail(6,0,0,after,0,0));}index+=1;}
    return .ok(Parts{.parsed,.names,.modules=spans,.order});
}

pub fn Result<Modules,Error> bind(sources::Sources input,ref mut usize meter){
    let Parts contents=switch(build(ref input,meter)){case .ok(value):{value}case .err(problem):{return .err(Error{.sources=input,.problem});}};return .ok(Modules{.sources=input,.contents});
}
