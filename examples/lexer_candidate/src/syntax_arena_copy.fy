module syntax_arena_copy;
import syntax as syntax;
import body_state as state;
import bodies as bodies;
import types as types;
import records as records;
import attributes as attributes;
import std::vec as v;

// Separate small typed copies preserve exact arena order and work charging.
// Their ordinary signatures use the existing supported vector ABI.
pub fn Result<v::Vec<syntax::Declaration>,u32> copy_declarations(ref v::Vec<syntax::Declaration> original,ref mut usize work){
    var v::Vec<syntax::Declaration> output=v::empty<syntax::Declaration>();var usize index=0;
    while(index<v::len<syntax::Declaration>(original)){if(*work>=1048576){return .err(10);};*work=*work+1;if(!v::push<syntax::Declaration>(ref mut output,*v::at<syntax::Declaration>(original,index)).is_ok){return .err(11);}index+=1;}
    return .ok(output);
}
pub fn Result<v::Vec<syntax::Problem>,u32> copy_problems(ref v::Vec<syntax::Problem> original,ref mut usize work){
    var v::Vec<syntax::Problem> output=v::empty<syntax::Problem>();var usize index=0;
    while(index<v::len<syntax::Problem>(original)){if(*work>=1048576){return .err(10);};*work=*work+1;if(!v::push<syntax::Problem>(ref mut output,*v::at<syntax::Problem>(original,index)).is_ok){return .err(11);}index+=1;}
    return .ok(output);
}
pub fn Result<v::Vec<state::Node>,u32> copy_expressions(ref v::Vec<state::Node> original,ref mut usize work){
    var v::Vec<state::Node> output=v::empty<state::Node>();var usize index=0;
    while(index<v::len<state::Node>(original)){if(*work>=1048576){return .err(10);};*work=*work+1;if(!v::push<state::Node>(ref mut output,*v::at<state::Node>(original,index)).is_ok){return .err(11);}index+=1;}
    return .ok(output);
}
pub fn Result<v::Vec<bodies::Entry>,u32> copy_entries(ref v::Vec<bodies::Entry> original,ref mut usize work){
    var v::Vec<bodies::Entry> output=v::empty<bodies::Entry>();var usize index=0;
    while(index<v::len<bodies::Entry>(original)){if(*work>=1048576){return .err(10);};*work=*work+1;if(!v::push<bodies::Entry>(ref mut output,*v::at<bodies::Entry>(original,index)).is_ok){return .err(11);}index+=1;}
    return .ok(output);
}
pub fn Result<v::Vec<state::Statement>,u32> copy_statements(ref v::Vec<state::Statement> original,ref mut usize work){
    var v::Vec<state::Statement> output=v::empty<state::Statement>();var usize index=0;
    while(index<v::len<state::Statement>(original)){if(*work>=1048576){return .err(10);};*work=*work+1;if(!v::push<state::Statement>(ref mut output,*v::at<state::Statement>(original,index)).is_ok){return .err(11);}index+=1;}
    return .ok(output);
}
pub fn Result<v::Vec<types::Node>,u32> copy_types(ref v::Vec<types::Node> original,ref mut usize work){
    var v::Vec<types::Node> output=v::empty<types::Node>();var usize index=0;
    while(index<v::len<types::Node>(original)){if(*work>=1048576){return .err(10);};*work=*work+1;if(!v::push<types::Node>(ref mut output,*v::at<types::Node>(original,index)).is_ok){return .err(11);}index+=1;}
    return .ok(output);
}
pub fn Result<v::Vec<records::Field>,u32> copy_fields(ref v::Vec<records::Field> original,ref mut usize work){
    var v::Vec<records::Field> output=v::empty<records::Field>();var usize index=0;
    while(index<v::len<records::Field>(original)){if(*work>=1048576){return .err(10);};*work=*work+1;if(!v::push<records::Field>(ref mut output,*v::at<records::Field>(original,index)).is_ok){return .err(11);}index+=1;}
    return .ok(output);
}
pub fn Result<v::Vec<attributes::Attribute>,u32> copy_attributes(ref v::Vec<attributes::Attribute> original,ref mut usize work){
    var v::Vec<attributes::Attribute> output=v::empty<attributes::Attribute>();var usize index=0;
    while(index<v::len<attributes::Attribute>(original)){if(*work>=1048576){return .err(10);};*work=*work+1;if(!v::push<attributes::Attribute>(ref mut output,*v::at<attributes::Attribute>(original,index)).is_ok){return .err(11);}index+=1;}
    return .ok(output);
}
pub fn Result<v::Vec<attributes::Argument>,u32> copy_attribute_arguments(ref v::Vec<attributes::Argument> original,ref mut usize work){
    var v::Vec<attributes::Argument> output=v::empty<attributes::Argument>();var usize index=0;
    while(index<v::len<attributes::Argument>(original)){if(*work>=1048576){return .err(10);};*work=*work+1;if(!v::push<attributes::Argument>(ref mut output,*v::at<attributes::Argument>(original,index)).is_ok){return .err(11);}index+=1;}
    return .ok(output);
}
