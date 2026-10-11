module project_manifest;
import project_toml_document as document;
import project_toml_model as m;
import project_manifest_values as rules;
import std::bytes as bytes;
import std::vec as v;

// Private reason: 1 parse/type, 2 unknown, 3 missing, 4 uid, 5 name,
// 6 version, 7 language, 8 path, 9 sorted/unique, 10 schema, 11 size,
// 12 UTF-8, 13 allocation, 14 work. Field IDs name the schema fields below.
pub struct Error{u32 reason;u32 field;usize start;usize end;}
pub struct Span{usize start;usize end;}
pub resource Manifest{m::Tree tree;v::Vec<usize> dependencies;usize package;usize workspace;cleanup{}}
pub fn ref u8[] field(u32 id){
    if(id==1){return b"schema";}if(id==2){return b"package";}if(id==3){return b"uid";}if(id==4){return b"name";}if(id==5){return b"version";}if(id==6){return b"language";}if(id==7){return b"source";}if(id==8){return b"workspace";}if(id==9){return b"members";}if(id==10){return b"dependencies";}if(id==11){return b"path";}if(id==12){return b"features";}return b"";
}
fn Error fail(ref m::Tree tree,u32 reason,u32 field,usize id){let m::Node n=m::node(tree,id);return Error{.reason,.field,.start=n.start,.end=n.end};}
fn bool charge(ref mut usize work,usize cost){let usize before=*work;if(before>16777216 || cost>16777216-before){return false;};*work=before+cost;return true;}
fn Result<usize,Error> lookup(ref m::Tree tree,usize parent,u32 field_id,ref mut usize work){
    return switch(m::resolve(tree,parent,field(field_id),work)){case .ok(n):{.ok(n)}case .err(e):{.err(fail(tree,if(e.code==4){14}else{1},field_id,parent))}};
}
fn u32 allowed(u32 role,ref u8[] name){
    if(role==0){if(bytes::equal(name,b"schema")){return 1;}if(bytes::equal(name,b"package")){return 2;}if(bytes::equal(name,b"workspace")){return 8;}if(bytes::equal(name,b"dependencies")){return 10;}}
    if(role==1){var u32 i=3;while(i<=7){if(bytes::equal(name,field(i))){return i;}i+=1;}}
    if(role==2 && bytes::equal(name,b"members")){return 9;}
    if(role==3){if(bytes::equal(name,b"path")){return 11;}if(bytes::equal(name,b"version")){return 5;}if(bytes::equal(name,b"features")){return 12;}}return 0;
}
fn u32 expected(u32 field_id){return if(field_id==2 || field_id==8 || field_id==10){1}else{if(field_id==9 || field_id==12){3}else{2}};}
fn Result<usize,Error> shape(ref m::Tree tree,usize parent,u32 role,ref mut usize work){
    var usize id=m::node(tree,parent).first;
    while(id!=0){let ref u8[] name=m::name(tree,id);if(!charge(work,name.len*6+1)){return .err(fail(tree,14,0,id));}let u32 field_id=allowed(role,name);
        if(field_id==0){return .err(fail(tree,2,0,id));}let m::Node n=m::node(tree,id);if(n.kind!=expected(field_id)){return .err(fail(tree,1,field_id,id));}
        if(n.kind==3){var usize item=n.first;while(item!=0){if(!charge(work,1)){return .err(fail(tree,14,field_id,item));}if(m::node(tree,item).kind!=2){return .err(fail(tree,1,field_id,item));}item=m::node(tree,item).next;}}
        if(field_id==2 || field_id==8){switch(shape(tree,id,if(field_id==2){1}else{2},work)){case .ok(_):{}case .err(e):{return .err(e);}};}
        if(field_id==10){var usize binding=n.first;while(binding!=0){if(!charge(work,1)){return .err(fail(tree,14,10,binding));}if(m::node(tree,binding).kind!=1){return .err(fail(tree,1,10,binding));}switch(shape(tree,binding,3,work)){case .ok(_):{}case .err(e):{return .err(e);}};binding=m::node(tree,binding).next;}}
        id=n.next;
    }return .ok(parent);
}
fn Result<usize,Error> required(ref m::Tree tree,usize parent,u32 field_id,ref mut usize work){
    let usize id=switch(lookup(tree,parent,field_id,work)){case .ok(n):{n}case .err(e):{return .err(e);}};if(id==0){return .err(fail(tree,3,field_id,parent));}return .ok(id);
}
fn Result<usize,Error> scalar(ref m::Tree tree,usize parent,u32 field_id,ref mut usize work){
    let usize id=switch(required(tree,parent,field_id,work)){case .ok(n):{n}case .err(e):{return .err(e);}};let ref u8[] value=m::value(tree,id);if(!charge(work,value.len+1)){return .err(fail(tree,14,field_id,id));}
    var bool valid=false;if(field_id==1){valid=bytes::equal(value,b"fairyc.manifest/v1");}if(field_id==3){valid=rules::uid(value);}if(field_id==4){valid=rules::name(value);}if(field_id==5){valid=rules::version(value);}if(field_id==6){valid=rules::language(value);}if(field_id==7 || field_id==11){valid=rules::path(value);}
    if(!valid){var u32 reason=8;if(field_id==1){reason=10;}if(field_id==3){reason=4;}if(field_id==4){reason=5;}if(field_id==5){reason=6;}if(field_id==6){reason=7;}return .err(fail(tree,reason,field_id,id));}return .ok(id);
}
fn Result<usize,Error> list(ref m::Tree tree,usize parent,u32 field_id,ref mut usize work){
    let usize array=switch(required(tree,parent,field_id,work)){case .ok(n):{n}case .err(e):{return .err(e);}};var usize item=m::node(tree,array).first;var usize previous=0;
    while(item!=0){let ref u8[] value=m::value(tree,item);if(!charge(work,value.len+1)){return .err(fail(tree,14,field_id,item));}
        if(field_id==9 && !rules::path(value)){return .err(fail(tree,8,field_id,item));}if(field_id==12 && !rules::name(value)){return .err(fail(tree,5,field_id,item));}
        if(previous!=0){let ref u8[] before=m::value(tree,previous);if(!charge(work,before.len+value.len+1)){return .err(fail(tree,14,field_id,item));}if(!rules::before(before,value)){return .err(fail(tree,9,field_id,item));}}
        previous=item;item=m::node(tree,item).next;
    }return .ok(array);
}
fn void put(ref mut v::Vec<usize> ids,usize index,usize value){let ref mut usize slot=v::at_mut<usize>(ids,index);*slot=value;}
fn Result<usize,Error> ordered(ref m::Tree tree,usize parent,ref mut v::Vec<usize> ids,ref mut usize work){
    var usize id=m::node(tree,parent).first;while(id!=0){if(!v::push<usize>(ids,id).is_ok){return .err(fail(tree,13,10,parent));}id=m::node(tree,id).next;}
    let usize count=v::len<usize>(ids);var v::Vec<usize> scratch=v::empty<usize>();if(!v::reserve<usize>(ref mut scratch,count).is_ok){return .err(fail(tree,13,10,parent));}
    while(v::len<usize>(ref scratch)<count){if(!v::push<usize>(ref mut scratch,0).is_ok){return .err(fail(tree,13,10,parent));}}
    var usize width=1;while(width<count){var usize start=0;while(start<count){let usize middle=if(width<count-start){start+width}else{count};let usize end=if(width<count-middle){middle+width}else{count};var usize left=start;var usize right=middle;var usize out=start;
        while(out<end){var bool take_left=right==end;if(left<middle && right<end){let usize a=*v::at<usize>(ids,left);let usize b=*v::at<usize>(ids,right);let ref u8[] first=m::name(tree,a);let ref u8[] second=m::name(tree,b);if(!charge(work,first.len+second.len+1)){return .err(fail(tree,14,10,parent));}take_left=rules::before(first,second);}
            let usize selected=if(take_left){*v::at<usize>(ids,left)}else{*v::at<usize>(ids,right)};put(ref mut scratch,out,selected);if(take_left){left+=1;}else{right+=1;}out+=1;
        }start=end;
    }var usize i=0;while(i<count){let usize value=*v::at<usize>(ref scratch,i);put(ids,i,value);i+=1;}width*=2;}return .ok(count);
}
pub fn Result<Manifest,Error> validate(m::Tree tree){
    var usize work=0;switch(shape(ref tree,1,0,ref mut work)){case .ok(_):{}case .err(e):{return .err(e);}};
    switch(required(ref tree,1,1,ref mut work)){case .ok(_):{}case .err(e):{return .err(e);}};let usize package=switch(required(ref tree,1,2,ref mut work)){case .ok(n):{n}case .err(e):{return .err(e);}};
    var u32 i=3;while(i<=7){switch(required(ref tree,package,i,ref mut work)){case .ok(_):{}case .err(e):{return .err(e);}};i+=1;}
    let usize workspace=switch(lookup(ref tree,1,8,ref mut work)){case .ok(n):{n}case .err(e):{return .err(e);}};if(workspace!=0){switch(required(ref tree,workspace,9,ref mut work)){case .ok(_):{}case .err(e):{return .err(e);}};}
    let usize dependencies=switch(lookup(ref tree,1,10,ref mut work)){case .ok(n):{n}case .err(e):{return .err(e);}};var v::Vec<usize> ids=v::empty<usize>();
    if(dependencies!=0){switch(ordered(ref tree,dependencies,ref mut ids,ref mut work)){case .ok(_):{}case .err(e):{return .err(e);}};var usize index=0;while(index<v::len<usize>(ref ids)){let usize binding=*v::at<usize>(ref ids,index);switch(required(ref tree,binding,11,ref mut work)){case .ok(_):{}case .err(e):{return .err(e);}};switch(required(ref tree,binding,5,ref mut work)){case .ok(_):{}case .err(e):{return .err(e);}};switch(required(ref tree,binding,12,ref mut work)){case .ok(_):{}case .err(e):{return .err(e);}};index+=1;}}
    switch(scalar(ref tree,1,1,ref mut work)){case .ok(_):{}case .err(e):{return .err(e);}};i=3;while(i<=7){switch(scalar(ref tree,package,i,ref mut work)){case .ok(_):{}case .err(e):{return .err(e);}};i+=1;}
    if(workspace!=0){switch(list(ref tree,workspace,9,ref mut work)){case .ok(_):{}case .err(e):{return .err(e);}};}
    var usize index=0;while(index<v::len<usize>(ref ids)){let usize binding=*v::at<usize>(ref ids,index);let ref u8[] name=m::name(ref tree,binding);if(!charge(ref mut work,name.len+1)){return .err(fail(ref tree,14,10,binding));}if(!rules::name(name)){return .err(fail(ref tree,5,10,binding));}
        switch(scalar(ref tree,binding,11,ref mut work)){case .ok(_):{}case .err(e):{return .err(e);}};switch(scalar(ref tree,binding,5,ref mut work)){case .ok(_):{}case .err(e):{return .err(e);}};switch(list(ref tree,binding,12,ref mut work)){case .ok(_):{}case .err(e):{return .err(e);}};index+=1;
    }return .ok(Manifest{.tree,.dependencies=ids,.package,.workspace});
}
pub fn Result<Manifest,Error> parse(ref u8[] source){
    return switch(document::parse(source)){case .ok(tree):{validate(tree)}case .err(e):{var u32 reason=1;if(e.code==2){reason=12;}if(e.code==3){reason=13;}if(e.code==4){reason=11;};.err(Error{.reason,.field=0,.start=e.start,.end=e.end})}};
}
pub fn ref u8[] package_value(ref Manifest manifest,ref u8[] key){return m::value(ref manifest.tree,m::lookup(ref manifest.tree,manifest.package,key));}
pub fn bool workspace_present(ref Manifest manifest){return manifest.workspace!=0;}
pub fn usize member_first(ref Manifest manifest){return m::node(ref manifest.tree,m::lookup(ref manifest.tree,manifest.workspace,b"members")).first;}
pub fn usize next_value(ref Manifest manifest,usize id){return m::node(ref manifest.tree,id).next;}
pub fn ref u8[] value(ref Manifest manifest,usize id){return m::value(ref manifest.tree,id);}
pub fn usize dependency_count(ref Manifest manifest){return v::len<usize>(ref manifest.dependencies);}
fn usize dependency(ref Manifest manifest,usize index){return if(index<dependency_count(manifest)){*v::at<usize>(ref manifest.dependencies,index)}else{0};}
pub fn ref u8[] dependency_name(ref Manifest manifest,usize index){return m::name(ref manifest.tree,dependency(manifest,index));}
pub fn ref u8[] dependency_value(ref Manifest manifest,usize index,ref u8[] key){return m::value(ref manifest.tree,m::lookup(ref manifest.tree,dependency(manifest,index),key));}
pub fn usize feature_first(ref Manifest manifest,usize index){return m::node(ref manifest.tree,m::lookup(ref manifest.tree,dependency(manifest,index),b"features")).first;}
pub fn Span package_span(ref Manifest manifest,ref u8[] key){let n=m::node(ref manifest.tree,m::lookup(ref manifest.tree,manifest.package,key));return Span{.start=n.start,.end=n.end};}
pub fn Span dependency_span(ref Manifest manifest,usize index,ref u8[] key){let usize id=dependency(manifest,index);let n=m::node(ref manifest.tree,if(key.len==0){id}else{m::lookup(ref manifest.tree,id,key)});return Span{.start=n.start,.end=n.end};}
