module project_check_options;
import native_cli_options as arguments;
import std::bytes as bytes;
import std::vec as v;

pub resource Options{bool locked;bool status;bytes::Buffer cache;cleanup{}}
// 1 unknown/duplicate ordinary option, 2 missing/invalid root, 3 duplicate cache
// root, 4 duplicate status, 5 status without cache, 6 allocation/argv decoding.
pub struct Problem{u32 code;}
pub fn bool locked(ref Options p){return p.locked;}
pub fn bool status(ref Options p){return p.status;}
pub fn ref u8[] cache_root(ref Options p){return bytes::view(ref p.cache);}
pub fn Result<Options,Problem> parse(ref v::Vec<bytes::Buffer> input){
    var bool locked=false;var bool status=false;var bytes::Buffer cache=bytes::empty();var usize i=0;
    while(i<v::len<bytes::Buffer>(input)){let arg=bytes::view(v::at<bytes::Buffer>(input,i));
        if(bytes::equal(arg,b"--locked")){if(locked){return .err(Problem{.code=1});}locked=true;}
        else if(bytes::equal(arg,b"--cache-status")){if(status){return .err(Problem{.code=4});}status=true;}
        else if(bytes::equal(arg,b"--cache-root")){
            if(bytes::len(ref cache)!=0){return .err(Problem{.code=3});}i+=1;if(i==v::len<bytes::Buffer>(input)){return .err(Problem{.code=2});}
            let root=bytes::view(v::at<bytes::Buffer>(input,i));if(root.len==0 || root[0]==45){return .err(Problem{.code=2});}
            var usize at=0;while(at<root.len){if(root[at]==0){return .err(Problem{.code=2});}at+=1;}
            cache=switch(bytes::copy(root)){case .ok(p):{p}case .err(_):{return .err(Problem{.code=6});}};
        }else{return .err(Problem{.code=1});}i+=1;
    }
    if(status && bytes::len(ref cache)==0){return .err(Problem{.code=5});}
    return .ok(Options{.locked,.status,.cache});
}
pub fn Result<Options,Problem> from_process(){let input=switch(arguments::read(2)){case .ok(p):{p}case .err(e):{return .err(Problem{.code=if(e.kind==5){6}else{1}});}};return parse(ref input);}
pub fn ref str diagnostic(Problem e){
    if(e.code==2){return "fairyc: --cache-root requires a nonempty directory path\n";}
    if(e.code==3){return "fairyc: command \"check\" received --cache-root more than once\n";}
    if(e.code==4){return "fairyc: command \"check\" received --cache-status more than once\n";}
    if(e.code==5){return "fairyc: --cache-status requires --cache-root\n";}
    if(e.code==6){return "fairyc: internal argument processing failed\n";}
    return "fairyc: command \"check\" accepts only optional --locked\n";
}
