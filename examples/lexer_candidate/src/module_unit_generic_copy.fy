module module_unit_generic_copy;
import module_unit_generics as m;
import module_types as types;
import std::vec as v;

// Preserve record order and the original one-charge-per-copy budget. These
// small concrete helpers avoid repeating vector instantiation in large bodies.

pub fn Result<v::Vec<m::Instance>,u32> instances(ref v::Vec<m::Instance> original,ref mut usize work){
    var v::Vec<m::Instance> output=v::empty<m::Instance>();var usize i=0;
    while(i<v::len<m::Instance>(original)){if(*work>=1048576){return .err(4503);};*work=*work+1;if(!v::push<m::Instance>(ref mut output,*v::at<m::Instance>(original,i)).is_ok){return .err(11);}i+=1;}
    return .ok(output);
}

pub fn Result<v::Vec<usize>,u32> indices(ref v::Vec<usize> original,ref mut usize work){
    var v::Vec<usize> output=v::empty<usize>();var usize i=0;
    while(i<v::len<usize>(original)){if(*work>=1048576){return .err(4503);};*work=*work+1;if(!v::push<usize>(ref mut output,*v::at<usize>(original,i)).is_ok){return .err(11);}i+=1;}
    return .ok(output);
}

pub fn Result<v::Vec<m::Call>,u32> calls(ref v::Vec<m::Call> original,ref mut usize work){
    var v::Vec<m::Call> output=v::empty<m::Call>();var usize i=0;
    while(i<v::len<m::Call>(original)){if(*work>=1048576){return .err(4503);};*work=*work+1;if(!v::push<m::Call>(ref mut output,*v::at<m::Call>(original,i)).is_ok){return .err(11);}i+=1;}
    return .ok(output);
}

pub fn Result<v::Vec<m::Dependency>,u32> dependencies(ref v::Vec<m::Dependency> original,ref mut usize work){
    var v::Vec<m::Dependency> output=v::empty<m::Dependency>();var usize i=0;
    while(i<v::len<m::Dependency>(original)){if(*work>=1048576){return .err(4503);};*work=*work+1;if(!v::push<m::Dependency>(ref mut output,*v::at<m::Dependency>(original,i)).is_ok){return .err(11);}i+=1;}
    return .ok(output);
}

pub fn Result<v::Vec<m::Cleanup>,u32> cleanups(ref v::Vec<m::Cleanup> original,ref mut usize work){
    var v::Vec<m::Cleanup> output=v::empty<m::Cleanup>();var usize i=0;
    while(i<v::len<m::Cleanup>(original)){if(*work>=1048576){return .err(4503);};*work=*work+1;if(!v::push<m::Cleanup>(ref mut output,*v::at<m::Cleanup>(original,i)).is_ok){return .err(11);}i+=1;}
    return .ok(output);
}

pub fn Result<v::Vec<m::SourceStamp>,u32> stamps(ref v::Vec<m::SourceStamp> original,ref mut usize work){
    var v::Vec<m::SourceStamp> output=v::empty<m::SourceStamp>();var usize i=0;
    while(i<v::len<m::SourceStamp>(original)){if(*work>=1048576){return .err(4503);};*work=*work+1;if(!v::push<m::SourceStamp>(ref mut output,*v::at<m::SourceStamp>(original,i)).is_ok){return .err(11);}i+=1;}
    return .ok(output);
}

pub fn Result<v::Vec<types::Type>,u32> type_records(ref v::Vec<types::Type> original,ref mut usize work){
    var v::Vec<types::Type> output=v::empty<types::Type>();var usize i=0;
    while(i<v::len<types::Type>(original)){if(*work>=1048576){return .err(4503);};*work=*work+1;if(!v::push<types::Type>(ref mut output,*v::at<types::Type>(original,i)).is_ok){return .err(11);}i+=1;}
    return .ok(output);
}

