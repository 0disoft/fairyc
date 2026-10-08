module hir_validate_drops;
import hir_model as h;
import resource_identity as resources;
import std::bytes as bytes;
import std::vec as v;
import hir_shapes as shapes;

pub fn h::Problem check(ref bytes::Buffer source, ref v::Vec<h::Function> functions, ref v::Vec<h::Binding> bindings, ref v::Vec<usize> edges, ref shapes::Table nominal, ref v::Vec<h::Drop> drops){
    let usize ns=bytes::len(source);
    let usize nf=v::len<h::Function>(functions);
    let usize nb=v::len<h::Binding>(bindings);
    var usize i=0;
    // Classification is owned data; never reconstruct a nonempty body from
    // fields or silently turn a missing private callable into an empty drop.
    var usize resources_count=0;i=0;
    while(i<shapes::count(nominal)){if(resources::valid(shapes::at(nominal,i).type)){resources_count+=1;}i+=1;}
    if(resources_count!=v::len<h::Drop>(drops)){return h::Problem{.code=4,.start=0,.end=ns};}
    var u8[1153] linked=[0;1153];i=0;
    while(i<v::len<h::Drop>(drops)){
        let h::Drop d=*v::at<h::Drop>(drops,i);
        if(!resources::valid(d.type) || shapes::find(nominal,d.type)==0 || d.start>=d.end || d.end>ns || d.function>nb || d.receiver>nb){return h::Problem{.code=4,.start=d.start,.end=d.end};}
        var usize j=0;while(j<i){if(v::at<h::Drop>(drops,j).type==d.type){return h::Problem{.code=4,.start=d.start,.end=d.end};}j+=1;}
        if(!d.nonempty){if(v::at<h::Binding>(bindings,shapes::at(nominal,shapes::find(nominal,d.type)-1).binding-1).effects!=0){return h::Problem{.code=4,.start=d.start,.end=d.end};}if(d.function!=0 || d.receiver!=0){return h::Problem{.code=4,.start=d.start,.end=d.end};}}
        else{
            if(d.function==0 || d.receiver==0 || d.function==d.receiver || linked[d.function]!=0 || linked[d.receiver]!=0){return h::Problem{.code=4,.start=d.start,.end=d.end};}
            let h::Binding b=*v::at<h::Binding>(bindings,d.function-1);let h::Binding receiver=*v::at<h::Binding>(bindings,d.receiver-1);
            let u32 authority=v::at<h::Binding>(bindings,shapes::at(nominal,shapes::find(nominal,d.type)-1).binding-1).effects;
            if(b.effects!=authority || h::binding_code(b.kind)!=9 || h::type_code(b.type)!=7 || b.declaration!=(d.type-98303) as usize || b.signature_count!=1 || b.start!=d.start || b.end!=d.end || h::binding_code(receiver.kind)!=10 || receiver.owner!=d.function || h::type_code(receiver.type)!=d.type || receiver.declaration!=0){return h::Problem{.code=4,.start=d.start,.end=d.end};}
            var usize matches=0;j=0;while(j<nf){let h::Function f=*v::at<h::Function>(functions,j);if(f.binding==d.function){
                if(f.public || f.parameter_count!=1 || *v::at<usize>(edges,f.parameter_start)!=d.receiver || f.start!=d.start || f.end!=d.end){return h::Problem{.code=4,.start=d.start,.end=d.end};}matches+=1;
            }j+=1;}if(matches!=1){return h::Problem{.code=4,.start=d.start,.end=d.end};}
            linked[d.function]=1;linked[d.receiver]=1;
        }i+=1;
    }
    i=1;while(i<=nb){let u32 kind=h::binding_code(v::at<h::Binding>(bindings,i-1).kind);if((kind==9 || kind==10) && linked[i]!=1){return h::Problem{.code=4,.start=0,.end=ns};}i+=1;}
    return h::Problem{.code=0,.start=0,.end=0};
}
