module resource_effects;
import effect_declarations as effects;
import syntax as syntax;
import resource_identity as resources;
import tagged_encoding as tagged;
import nominal_types as nominal;
struct Budget { usize work; }
// 4096 is outside every valid caller set: fail closed on bounded traversal.
fn u32 walk(ref syntax::Parsed p,u32 type,usize depth,ref mut Budget budget){
    if(depth>=64 || budget.work>=16384){return 4096;}budget.work+=1;
    if(tagged::option(type)){return walk(p,tagged::inner(type),depth+1,budget);}
    if(tagged::result(type)){let u32 first=walk(p,tagged::success(type),depth+1,budget);return first|walk(p,tagged::failure(type),depth+1,budget);}
    if(!resources::valid(type)){return 0;}let usize at=(type-98304) as usize;let d=syntax::declaration_at(p,at);var u32 bits=effects::read(p,at).bits;var usize i=0;
    while(i<d.field_count){let f=syntax::field_at(p,d.field_start+i);bits|=walk(p,nominal::resolve(p,f.type_root),depth+1,budget);i+=1;}return bits;
}
pub fn u32 required(ref syntax::Parsed p,u32 type){var Budget budget=Budget{.work=0};return walk(p,type,0,ref mut budget);}
