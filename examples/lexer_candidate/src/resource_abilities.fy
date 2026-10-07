module resource_abilities;
import syntax as syntax;
import resource_identity as identity;
import tagged_encoding as tagged;
import nominal_types as nominal;
import type_state as state;
import std::bytes as bytes;

// Closed static metadata: drop=1, store=2, key=4. Unannotated resources drop.
pub fn u32 mask(ref syntax::Parsed p,usize declaration){
    let d=syntax::declaration_at(p,declaration);var usize i=0;
    while(i<d.attribute_count){let a=syntax::attribute_at(p,d.attribute_start+i);let name=syntax::source_view(p)[a.name_start..a.name_end];
        if(bytes::equal(name,b"abilities")){var u32 bits=0;var usize j=0;while(j<a.argument_count){let text=syntax::attribute_argument_text(p,a.argument_start+j);bits|=if(bytes::equal(text,b"drop")){1}else if(bytes::equal(text,b"store")){2}else{4};j+=1;}return bits;}i+=1;}
    return 1;
}
pub fn state::Problem validate(ref syntax::Parsed p){
    var usize i=0;while(i<syntax::declaration_count(p)){let d=syntax::declaration_at(p,i);
        if(d.kind==6){var bool seen=false;var usize at=0;while(at<d.attribute_count){let a=syntax::attribute_at(p,d.attribute_start+at);let name=syntax::source_view(p)[a.name_start..a.name_end];at+=1;
            if(bytes::equal(name,b"effects")){continue;}
            if(!bytes::equal(name,b"abilities") || seen || a.argument_count==0){return state::fail(49,a.start,a.end,0,0);}seen=true;
            var u32 bits=0;var usize j=0;while(j<a.argument_count){let arg=syntax::attribute_argument_at(p,a.argument_start+j);let text=syntax::attribute_argument_text(p,a.argument_start+j);
                let u32 bit=if(bytes::equal(text,b"drop")){1}else if(bytes::equal(text,b"store")){2}else if(bytes::equal(text,b"key")){4}else{0};
                if(arg.kind!=1 || bit==0 || (bits&bit)!=0){return state::fail(49,arg.start,arg.end,0,0);}bits|=bit;j+=1;}
            if((bits&4)!=0 && (bits&2)==0){return state::fail(49,a.start,a.end,0,0);}
        }}i+=1;
    }return state::ok();
}
// Return the declaration withholding drop, including recursive containment.
pub fn usize denied(ref syntax::Parsed p,u32 type,usize depth){
    if(depth>=64){return 1;}
    if(tagged::option(type)){return denied(p,tagged::inner(type),depth+1);}
    if(tagged::result(type)){let usize first=denied(p,tagged::success(type),depth+1);return if(first!=0){first}else{denied(p,tagged::failure(type),depth+1)};}
    if(!identity::valid(type)){return 0;}let usize at=(type-98304) as usize;let d=syntax::declaration_at(p,at);
    if((mask(p,at)&1)==0){return at+1;}
    var usize i=0;while(i<d.field_count){let f=syntax::field_at(p,d.field_start+i);let usize inner=denied(p,nominal::resolve(p,f.type_root),depth+1);if(inner!=0){return inner;}i+=1;}return 0;
}
pub fn bool burnable(ref syntax::Parsed p,u32 type){
    if(!identity::valid(type)){return false;}let usize at=(type-98304) as usize;let d=syntax::declaration_at(p,at);
    if((mask(p,at)&1)!=0){return false;}var usize i=0;while(i<d.field_count){let f=syntax::field_at(p,d.field_start+i);if(nominal::contains_resource(p,nominal::resolve(p,f.type_root),0)){return false;}i+=1;}return true;
}
