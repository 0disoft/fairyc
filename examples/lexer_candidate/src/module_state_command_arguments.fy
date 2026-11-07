module module_state_command_arguments;
import module_state_command_options as options;
import std::bytes as bytes;
import std::vec as v;
import std::process as process;

// OS argv admission only. No source/store/asset I/O or command selection.
// Own the exact argv backing the parser's copied source/store indexes.
pub resource Invocation{v::Vec<bytes::Buffer> argv;options::Parsed parsed;cleanup{}}
pub enum Failure{
    Invocation{options::InvocationError error;};
    Host{u32 status;usize index;usize needed;};
    Allocation;
}
pub fn ref v::Vec<bytes::Buffer> arguments(ref Invocation value){return ref value.argv;}
pub fn options::Parsed parsed(ref Invocation value){return value.parsed;}
pub fn i32 exit_code(Failure error){return switch(error){case .Invocation{.error=e}:{options::exit_code(e)}case .Host{.status=s,.index=i,.needed=n}:{4}case .Allocation:{4}};}
pub fn Result<Invocation,Failure> read(){
    let count=process::arg_count();
    if(count==0){return .err(.Host{.status=1,.index=0,.needed=0});}
    let n=count-1;
    if(n!=1 && n!=3 && n!=7){return .err(.Invocation{.error=.Invalid});}
    var input=v::empty<bytes::Buffer>();var scratch=bytes::empty();
    if(!bytes::resize_zeroed(ref mut scratch,128).is_ok || !v::reserve<bytes::Buffer>(ref mut input,n).is_ok){return .err(.Allocation);}
    var usize total=0;var usize i=1;
    while(i<count){
        let size=switch(process::arg(i,bytes::view_mut(ref mut scratch))){case .ok(p):{p}case .err(e):{
            if(e.status!=3 || e.needed<=bytes::len(ref scratch)){return .err(.Host{.status=e.status,.index=i,.needed=e.needed});}
            if(e.needed>options::byte_limit()-total){return .err(.Invocation{.error=.TooLarge});}
            if(!bytes::resize_zeroed(ref mut scratch,e.needed).is_ok){return .err(.Allocation);}
            switch(process::arg(i,bytes::view_mut(ref mut scratch))){case .ok(p):{p}case .err(e):{return .err(.Host{.status=e.status,.index=i,.needed=e.needed});}}
        }};
        if(size>bytes::len(ref scratch)){return .err(.Host{.status=3,.index=i,.needed=size});}
        if(size>options::byte_limit()-total){return .err(.Invocation{.error=.TooLarge});}
        total+=size;
        let copied=switch(bytes::copy(bytes::view(ref scratch)[..size])){case .ok(p):{p}case .err(_):{return .err(.Allocation);}};
        if(!v::push<bytes::Buffer>(ref mut input,copied).is_ok){return .err(.Allocation);}i+=1;
    }
    let selected=switch(options::parse(ref input)){case .ok(p):{p}case .err(e):{return .err(.Invocation{.error=e});}};
    return .ok(Invocation{.argv=input,.parsed=selected});
}
