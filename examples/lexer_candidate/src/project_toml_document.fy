module project_toml_document;
import project_toml_model as m;
import project_toml_values as g;
import project_toml_context as context;
import project_toml_strings as strings;
import project_toml_scalars as atoms;
import std::bytes as bytes;
import std::text as text;
import std::vec as v;

// Manifest syntax retains typed atoms so valid unknown fields and malformed
// values preserve the executable Go decoder's rejection categories.
struct Cursor{usize offset;usize depth;}
fn m::Problem fail(ref u8[] source,u32 code,usize start,usize end){let usize a=if(start<source.len){start}else{source.len};let usize b=if(end<source.len){end}else{source.len};return m::Problem{.code,.start=a,.end=if(b>=a){b}else{a}};}
fn m::Problem here(ref u8[] source,ref Cursor cursor){return fail(source,1,cursor.offset,cursor.offset+1);}
fn u8 peek(ref u8[] source,ref Cursor cursor){return if(cursor.offset<source.len){source[cursor.offset]}else{0};}
fn void horizontal(ref u8[] source,ref mut Cursor cursor){while(cursor.offset<source.len && (source[cursor.offset]==32 || source[cursor.offset]==9)){cursor.offset+=1;}}
fn Result<usize,m::Problem> newline(ref u8[] source,ref mut Cursor cursor){
    if(peek(source,cursor)==10){cursor.offset+=1;return .ok(cursor.offset);}
    if(peek(source,cursor)==13 && source.len-cursor.offset>=2 && source[cursor.offset+1]==10){cursor.offset+=2;return .ok(cursor.offset);}return .err(here(source,cursor));
}
fn Result<usize,m::Problem> trivia(ref u8[] source,ref mut Cursor cursor,bool multiline){
    while(true){horizontal(source,cursor);if(cursor.offset==source.len){return .ok(cursor.offset);}let u8 c=peek(source,cursor);
        if(c==35 && multiline){cursor.offset+=1;while(cursor.offset<source.len && peek(source,cursor)!=10 && peek(source,cursor)!=13){let u8 b=peek(source,cursor);if(b<32 && b!=9 || b==127){return .err(here(source,cursor));}cursor.offset+=1;}continue;}
        if((c==10 || c==13) && multiline){switch(newline(source,cursor)){case .ok(_):{}case .err(e):{return .err(e);}};continue;}return .ok(cursor.offset);
    }return .err(here(source,cursor));
}
fn bool bare(u8 c){return c>=65 && c<=90 || c>=97 && c<=122 || c>=48 && c<=57 || c==45 || c==95;}
fn Result<usize,m::Problem> component(ref u8[] source,ref mut Cursor cursor,ref mut bytes::Buffer key){
    horizontal(source,cursor);let usize start=cursor.offset;let u8 c=peek(source,cursor);
    if(c==34 || c==39){let usize end=switch(strings::append(source,start,false,key)){case .ok(n):{n}case .err(e):{return .err(fail(source,e.code,e.start,e.end));}};cursor.offset=end;return .ok(start);}
    while(cursor.offset<source.len && bare(peek(source,cursor))){cursor.offset+=1;}
    if(cursor.offset==start){return .err(here(source,cursor));}
    if(!bytes::append(key,source[start..cursor.offset]).is_ok){return .err(fail(source,3,start,cursor.offset));}return .ok(start);
}
struct Key{usize logical;usize parent;usize start;usize end;}
fn Result<usize,m::Problem> map(ref mut m::State meta,ref bytes::Buffer context_names,ref v::Vec<context::Entry> contexts,ref bytes::Buffer names,ref v::Vec<g::Value> objects,ref v::Vec<g::Slot> slots,ref v::Vec<usize> buckets,usize logical){
    var usize[512] path=[0;512];var usize size=0;var usize cursor=logical;
    while(cursor!=0){if(cursor>v::len<context::Entry>(contexts) || size==512){return .err(m::Problem{.code=4,.start=0,.end=0});}if(!g::charge(meta,1)){return .err(m::Problem{.code=4,.start=0,.end=0});}path[size]=cursor;size+=1;let context::Entry e=*v::at<context::Entry>(contexts,cursor-1);if(e.parent>=cursor){return .err(m::Problem{.code=1,.start=0,.end=0});}cursor=e.parent;}
    var usize table=1;while(size>0){size-=1;let context::Entry e=*v::at<context::Entry>(contexts,path[size]-1);let usize found=switch(g::lookup(meta,names,objects,slots,buckets,table,bytes::view(context_names)[e.start..e.end],0,0)){case .ok(n):{n}case .err(x):{return .err(x);}};if(found==0){return .err(m::Problem{.code=1,.start=0,.end=0});}let usize value=g::slot(slots,found).value;let g::Value n=g::node(objects,value);
        if(n.kind==1){table=value;}else if(n.kind==4 && n.last!=0){table=g::slot(slots,n.last).value;}else{return .err(m::Problem{.code=1,.start=0,.end=0});}
    }return .ok(table);
}
fn Result<usize,m::Problem> set_global(ref u8[] source,ref mut m::State meta,ref bytes::Buffer context_names,ref mut v::Vec<context::Entry> contexts,ref mut bytes::Buffer names,ref mut v::Vec<g::Value> objects,ref mut v::Vec<g::Slot> slots,ref mut v::Vec<usize> buckets,Key key,usize value){
    let context::Entry e=*v::at<context::Entry>(contexts,key.logical-1);let usize table=switch(map(meta,context_names,contexts,names,objects,slots,buckets,key.parent)){case .ok(n):{n}case .err(x):{return .err(fail(source,x.code,key.start,key.end));}};let usize found=switch(g::lookup(meta,names,objects,slots,buckets,table,bytes::view(context_names)[e.start..e.end],key.start,key.end)){case .ok(n):{n}case .err(x):{return .err(x);}};
    let u32 action=switch(context::define(contexts,meta,key.logical,found!=0)){case .ok(n):{n}case .err(x):{return .err(fail(source,x.code,key.start,key.end));}};if(action==1){let usize prior=g::slot(slots,found).value;let g::Value old=g::node(objects,prior);let g::Value fresh=g::node(objects,value);
        if((fresh.kind==1 || fresh.kind==3) && (old.kind==1 || old.kind==3 || old.kind==4)){switch(g::finish(meta,objects,prior,fresh.end,old.state)){case .ok(_):{}case .err(x):{return .err(x);}};}return .ok(prior);}
    switch(g::put(meta,names,objects,slots,buckets,table,bytes::view(context_names)[e.start..e.end],value,key.start,key.end)){case .ok(_):{}case .err(x):{return .err(x);}};return .ok(value);
}
// Header parents are created at their own component spans without consuming
// existing implicit flags. This matches addContext's parent traversal.
fn Result<usize,m::Problem> header_parent(ref u8[] source,ref mut m::State meta,ref bytes::Buffer context_names,ref mut v::Vec<context::Entry> contexts,ref mut bytes::Buffer names,ref mut v::Vec<g::Value> objects,ref mut v::Vec<g::Slot> slots,ref mut v::Vec<usize> buckets,Key key){
    let usize parent=switch(map(meta,context_names,contexts,names,objects,slots,buckets,key.parent)){case .ok(n):{n}case .err(x):{return .err(fail(source,x.code,key.start,key.end));}};let context::Entry e=*v::at<context::Entry>(contexts,key.logical-1);let usize found=switch(g::lookup(meta,names,objects,slots,buckets,parent,bytes::view(context_names)[e.start..e.end],key.start,key.end)){case .ok(n):{n}case .err(x):{return .err(x);}};var usize value=if(found==0){0}else{g::slot(slots,found).value};
    if(found==0){switch(context::mark_implicit(contexts,meta,key.logical)){case .ok(_):{}case .err(x):{return .err(x);}};value=switch(g::make(meta,objects,1,0,key.start,key.end)){case .ok(n):{n}case .err(x):{return .err(x);}};switch(g::put(meta,names,objects,slots,buckets,parent,bytes::view(context_names)[e.start..e.end],value,key.start,key.end)){case .ok(_):{}case .err(x):{return .err(x);}};}
    let g::Value n=g::node(objects,value);if(n.kind==1){return .ok(value);}if(n.kind==4 && n.last!=0){return .ok(g::slot(slots,n.last).value);}return .err(fail(source,1,key.start,key.end));
}
fn Result<usize,m::Problem> establish(ref u8[] source,ref mut m::State meta,ref bytes::Buffer context_names,ref mut v::Vec<context::Entry> contexts,ref mut bytes::Buffer names,ref mut v::Vec<g::Value> objects,ref mut v::Vec<g::Slot> slots,ref mut v::Vec<usize> buckets,Key key,bool array,u32 state){
    var usize[512] path=[0;512];var usize size=0;var usize cursor=key.parent;
    while(cursor!=0){if(size==512 || cursor>v::len<context::Entry>(contexts) || !g::charge(meta,1)){return .err(fail(source,4,key.start,key.end));}path[size]=cursor;size+=1;cursor=v::at<context::Entry>(contexts,cursor-1).parent;}
    var usize table=1;while(size>0){size-=1;let usize id=path[size];let context::Entry e=*v::at<context::Entry>(contexts,id-1);let usize found=switch(g::lookup(meta,names,objects,slots,buckets,table,bytes::view(context_names)[e.start..e.end],key.start,key.end)){case .ok(n):{n}case .err(x):{return .err(x);}};var usize value=0;
        if(found==0){switch(context::mark_implicit(contexts,meta,id)){case .ok(_):{}case .err(x):{return .err(x);}};value=switch(g::make(meta,objects,1,0,key.start,key.end)){case .ok(n):{n}case .err(x):{return .err(x);}};switch(g::put(meta,names,objects,slots,buckets,table,bytes::view(context_names)[e.start..e.end],value,key.start,key.end)){case .ok(_):{}case .err(x):{return .err(x);}};}else{value=g::slot(slots,found).value;}
        let g::Value n=g::node(objects,value);if(n.kind==1){table=value;}else if(n.kind==4 && n.last!=0){table=g::slot(slots,n.last).value;}else{return .err(fail(source,1,key.start,key.end));}
    }
    let context::Entry e=*v::at<context::Entry>(contexts,key.logical-1);let usize found=switch(g::lookup(meta,names,objects,slots,buckets,table,bytes::view(context_names)[e.start..e.end],key.start,key.end)){case .ok(n):{n}case .err(x):{return .err(x);}};
    if(array){var usize list=if(found==0){0}else{g::slot(slots,found).value};if(list==0){list=switch(g::make(meta,objects,4,0,key.start,key.end)){case .ok(n):{n}case .err(x):{return .err(x);}};switch(g::put(meta,names,objects,slots,buckets,table,bytes::view(context_names)[e.start..e.end],list,key.start,key.end)){case .ok(_):{}case .err(x):{return .err(x);}};}else if(g::node(objects,list).kind!=4){return .err(fail(source,1,key.start,key.end));}
        let usize item=switch(g::make(meta,objects,1,2,key.start,key.end)){case .ok(n):{n}case .err(x):{return .err(x);}};switch(g::element(meta,names,objects,slots,buckets,list,item,key.start,key.end)){case .ok(_):{}case .err(x):{return .err(x);}};return .ok(item);
    }
    let u32 action=switch(context::define(contexts,meta,key.logical,found!=0)){case .ok(n):{n}case .err(x):{return .err(fail(source,x.code,key.start,key.end));}};var usize value=if(found==0){0}else{g::slot(slots,found).value};
    if(action!=1){value=switch(g::make(meta,objects,1,state,key.start,key.end)){case .ok(n):{n}case .err(x):{return .err(x);}};switch(g::put(meta,names,objects,slots,buckets,table,bytes::view(context_names)[e.start..e.end],value,key.start,key.end)){case .ok(_):{}case .err(x):{return .err(x);}};}
    let g::Value n=g::node(objects,value);if(n.kind==1){return .ok(value);}if(n.kind==4 && n.last!=0){return .ok(g::slot(slots,n.last).value);}return .err(fail(source,1,key.start,key.end));
}
fn Result<Key,m::Problem> path(ref u8[] source,ref mut Cursor cursor,ref mut m::State meta,ref mut bytes::Buffer context_names,ref mut v::Vec<context::Entry> contexts,ref mut v::Vec<usize> context_buckets,ref mut bytes::Buffer names,ref mut v::Vec<g::Value> objects,ref mut v::Vec<g::Slot> slots,ref mut v::Vec<usize> buckets,usize root,bool dotted){
    var usize parent=root;while(true){var bytes::Buffer key=bytes::empty();let usize start=switch(component(source,cursor,ref mut key)){case .ok(n):{n}case .err(x):{return .err(x);}};let usize end=cursor.offset;let usize id=switch(context::intern(context_names,contexts,context_buckets,meta,parent,bytes::view(ref key),start,end)){case .ok(n):{n}case .err(x):{return .err(x);}};horizontal(source,cursor);let Key result=Key{.logical=id,.parent,.start,.end};if(peek(source,cursor)!=46){return .ok(result);}
        if(dotted){switch(context::mark_implicit(contexts,meta,id)){case .ok(_):{}case .err(x):{return .err(x);}};switch(establish(source,meta,context_names,contexts,names,objects,slots,buckets,result,false,1)){case .ok(_):{}case .err(x):{return .err(x);}};}else{switch(header_parent(source,meta,context_names,contexts,names,objects,slots,buckets,result)){case .ok(_):{}case .err(x):{return .err(x);}};}parent=id;cursor.offset+=1;
    }return .err(here(source,cursor));
}
fn Result<usize,m::Problem> set_local(ref u8[] source,ref mut m::State meta,ref bytes::Buffer context_names,ref v::Vec<context::Entry> contexts,ref mut bytes::Buffer names,ref mut v::Vec<g::Value> objects,ref mut v::Vec<g::Slot> slots,ref mut v::Vec<usize> buckets,usize table,usize base,Key key,usize value){
    var usize[512] path=[0;512];var usize size=0;var usize cursor=key.parent;while(cursor!=base){if(cursor==0 || size==512 || !g::charge(meta,1)){return .err(fail(source,4,key.start,key.end));}path[size]=cursor;size+=1;cursor=v::at<context::Entry>(contexts,cursor-1).parent;}var usize parent=table;
    while(size>0){size-=1;let context::Entry e=*v::at<context::Entry>(contexts,path[size]-1);let usize found=switch(g::lookup(meta,names,objects,slots,buckets,parent,bytes::view(context_names)[e.start..e.end],key.start,key.end)){case .ok(n):{n}case .err(x):{return .err(x);}};var usize child=if(found==0){0}else{g::slot(slots,found).value};if(child==0){child=switch(g::make(meta,objects,1,1,key.start,key.end)){case .ok(n):{n}case .err(x):{return .err(x);}};switch(g::put(meta,names,objects,slots,buckets,parent,bytes::view(context_names)[e.start..e.end],child,key.start,key.end)){case .ok(_):{}case .err(x):{return .err(x);}};}else if(g::node(objects,child).kind!=1){return .err(fail(source,1,key.start,key.end));}parent=child;
    }let context::Entry e=*v::at<context::Entry>(contexts,key.logical-1);return g::put(meta,names,objects,slots,buckets,parent,bytes::view(context_names)[e.start..e.end],value,key.start,key.end);
}
fn Result<usize,m::Problem> value(ref u8[] source,ref mut Cursor cursor,ref mut m::State meta,ref mut bytes::Buffer context_names,ref mut v::Vec<context::Entry> contexts,ref mut v::Vec<usize> context_buckets,ref mut bytes::Buffer names,ref mut bytes::Buffer data,ref mut v::Vec<g::Value> objects,ref mut v::Vec<g::Slot> slots,ref mut v::Vec<usize> buckets,Key key,bool parent_array){
    horizontal(source,cursor);let usize start=cursor.offset;let u8 c=peek(source,cursor);if(c==34 || c==39){var bytes::Buffer cooked=bytes::empty();let usize end=switch(strings::append(source,start,true,ref mut cooked)){case .ok(n):{n}case .err(x):{return .err(fail(source,x.code,x.start,x.end));}};cursor.offset=end;return g::atom(meta,data,objects,2,bytes::view(ref cooked),start,end);}
    if(source.len-start>=4 && bytes::equal(source[start..start+4],b"true")){cursor.offset+=4;return g::atom(meta,data,objects,5,b"true",start,cursor.offset);}if(source.len-start>=5 && bytes::equal(source[start..start+5],b"false")){cursor.offset+=5;return g::atom(meta,data,objects,5,b"false",start,cursor.offset);}
    if(c!=91 && c!=123){let atoms::Token token=atoms::read(source,start);if(token.kind==0){return .err(fail(source,1,start,token.end));}cursor.offset=token.end;return g::atom(meta,data,objects,token.kind,source[start..token.end],start,token.end);}if(cursor.depth>=512){return .err(fail(source,4,start,start+1));}cursor.depth+=1;
    let result=if(c==91){array(source,cursor,meta,context_names,contexts,context_buckets,names,data,objects,slots,buckets,key)}else{inline_table(source,cursor,meta,context_names,contexts,context_buckets,names,data,objects,slots,buckets,key,parent_array)};cursor.depth-=1;return result;
}
fn Result<usize,m::Problem> array(ref u8[] source,ref mut Cursor cursor,ref mut m::State meta,ref mut bytes::Buffer context_names,ref mut v::Vec<context::Entry> contexts,ref mut v::Vec<usize> context_buckets,ref mut bytes::Buffer names,ref mut bytes::Buffer data,ref mut v::Vec<g::Value> objects,ref mut v::Vec<g::Slot> slots,ref mut v::Vec<usize> buckets,Key key){
    let usize start=cursor.offset;cursor.offset+=1;switch(context::record_type(contexts,meta,key.logical,3)){case .ok(_):{}case .err(x):{return .err(x);}};let usize id=switch(g::make(meta,objects,3,0,start,cursor.offset)){case .ok(n):{n}case .err(x):{return .err(x);}};
    while(true){switch(trivia(source,cursor,true)){case .ok(_):{}case .err(x):{return .err(x);}};if(peek(source,cursor)==93){cursor.offset+=1;return g::finish(meta,objects,id,cursor.offset,0);}if(cursor.offset==source.len){return .err(fail(source,1,start,cursor.offset));}let usize item_start=cursor.offset;
        let usize item=switch(value(source,cursor,meta,context_names,contexts,context_buckets,names,data,objects,slots,buckets,key,true)){case .ok(n):{n}case .err(x):{return .err(x);}};switch(g::element(meta,names,objects,slots,buckets,id,item,item_start,cursor.offset)){case .ok(_):{}case .err(x):{return .err(x);}};switch(trivia(source,cursor,true)){case .ok(_):{}case .err(x):{return .err(x);}};let u8 c=peek(source,cursor);if(c==44){cursor.offset+=1;continue;}if(c!=93){return .err(here(source,cursor));}
    }return .err(here(source,cursor));
}
fn Result<usize,m::Problem> assignment(ref u8[] source,ref mut Cursor cursor,ref mut m::State meta,ref mut bytes::Buffer context_names,ref mut v::Vec<context::Entry> contexts,ref mut v::Vec<usize> context_buckets,ref mut bytes::Buffer names,ref mut bytes::Buffer data,ref mut v::Vec<g::Value> objects,ref mut v::Vec<g::Slot> slots,ref mut v::Vec<usize> buckets,usize root,usize local){
    let Key key=switch(path(source,cursor,meta,context_names,contexts,context_buckets,names,objects,slots,buckets,root,true)){case .ok(n):{n}case .err(x):{return .err(x);}};if(peek(source,cursor)!=61){return .err(here(source,cursor));}cursor.offset+=1;
    let usize item=switch(value(source,cursor,meta,context_names,contexts,context_buckets,names,data,objects,slots,buckets,key,false)){case .ok(n):{n}case .err(x):{return .err(x);}};switch(set_global(source,meta,context_names,contexts,names,objects,slots,buckets,key,item)){case .ok(_):{}case .err(x):{return .err(x);}};switch(context::record_type(contexts,meta,key.logical,g::node(objects,item).kind)){case .ok(_):{}case .err(x):{return .err(x);}};
    if(local!=0){switch(set_local(source,meta,context_names,contexts,names,objects,slots,buckets,local,root,key,item)){case .ok(_):{}case .err(x):{return .err(x);}};}return .ok(item);
}
fn Result<usize,m::Problem> inline_table(ref u8[] source,ref mut Cursor cursor,ref mut m::State meta,ref mut bytes::Buffer context_names,ref mut v::Vec<context::Entry> contexts,ref mut v::Vec<usize> context_buckets,ref mut bytes::Buffer names,ref mut bytes::Buffer data,ref mut v::Vec<g::Value> objects,ref mut v::Vec<g::Slot> slots,ref mut v::Vec<usize> buckets,Key key,bool parent_array){
    let usize start=cursor.offset;cursor.offset+=1;let usize id=switch(g::make(meta,objects,1,2,start,cursor.offset)){case .ok(n):{n}case .err(x):{return .err(x);}};switch(context::mark_implicit(contexts,meta,key.logical)){case .ok(_):{}case .err(x):{return .err(x);}};switch(establish(source,meta,context_names,contexts,names,objects,slots,buckets,key,parent_array,2)){case .ok(_):{}case .err(x):{return .err(x);}};
    while(true){switch(trivia(source,cursor,true)){case .ok(_):{}case .err(x):{return .err(x);}};if(peek(source,cursor)==125){cursor.offset+=1;return g::finish(meta,objects,id,cursor.offset,2);}if(cursor.offset==source.len){return .err(fail(source,1,start,cursor.offset));}
        switch(assignment(source,cursor,meta,context_names,contexts,context_buckets,names,data,objects,slots,buckets,key.logical,id)){case .ok(_):{}case .err(x):{return .err(x);}};switch(trivia(source,cursor,true)){case .ok(_):{}case .err(x):{return .err(x);}};let u8 c=peek(source,cursor);if(c==44){cursor.offset+=1;continue;}if(c!=125){return .err(here(source,cursor));}
    }return .err(here(source,cursor));
}
fn Result<usize,m::Problem> statement_end(ref u8[] source,ref mut Cursor cursor){
    horizontal(source,cursor);if(peek(source,cursor)==35){cursor.offset+=1;while(cursor.offset<source.len && peek(source,cursor)!=10 && peek(source,cursor)!=13){let u8 c=peek(source,cursor);if(c<32 && c!=9 || c==127){return .err(here(source,cursor));}cursor.offset+=1;}}if(cursor.offset==source.len){return .ok(cursor.offset);}return newline(source,cursor);
}
pub fn Result<m::Tree,m::Problem> parse(ref u8[] source){
    if(source.len>1048576){return .err(fail(source,4,0,0));}if(!text::validate_utf8(source).is_ok){return .err(fail(source,2,0,source.len));}var m::State meta=m::state(source.len);var bytes::Buffer context_names=bytes::empty();var v::Vec<context::Entry> contexts=v::empty<context::Entry>();var v::Vec<usize> context_buckets=v::empty<usize>();var bytes::Buffer names=bytes::empty();var bytes::Buffer data=bytes::empty();var v::Vec<g::Value> objects=v::empty<g::Value>();var v::Vec<g::Slot> slots=v::empty<g::Slot>();var v::Vec<usize> buckets=v::empty<usize>();switch(g::make(ref mut meta,ref mut objects,1,2,0,source.len)){case .ok(_):{}case .err(x):{return .err(x);}};var Cursor cursor=Cursor{.offset=0,.depth=0};var usize table=0;
    while(true){switch(trivia(source,ref mut cursor,true)){case .ok(_):{}case .err(x):{return .err(x);}};if(cursor.offset==source.len){break;}
        if(peek(source,ref cursor)==91){cursor.offset+=1;let bool many=peek(source,ref cursor)==91;if(many){cursor.offset+=1;}let Key key=switch(path(source,ref mut cursor,ref mut meta,ref mut context_names,ref mut contexts,ref mut context_buckets,ref mut names,ref mut objects,ref mut slots,ref mut buckets,0,false)){case .ok(n):{n}case .err(x):{return .err(x);}};if(peek(source,ref cursor)!=93){return .err(here(source,ref cursor));}cursor.offset+=1;if(many){if(peek(source,ref cursor)!=93){return .err(here(source,ref cursor));}cursor.offset+=1;}
            let usize id=switch(establish(source,ref mut meta,ref context_names,ref mut contexts,ref mut names,ref mut objects,ref mut slots,ref mut buckets,key,many,2)){case .ok(n):{n}case .err(x):{return .err(x);}};switch(g::finish(ref meta,ref mut objects,id,cursor.offset,2)){case .ok(_):{}case .err(x):{return .err(x);}};switch(context::set_kind(ref mut contexts,ref meta,key.logical,if(many){4}else{1})){case .ok(_):{}case .err(x):{return .err(x);}};table=key.logical;
        }else{switch(assignment(source,ref mut cursor,ref mut meta,ref mut context_names,ref mut contexts,ref mut context_buckets,ref mut names,ref mut data,ref mut objects,ref mut slots,ref mut buckets,table,0)){case .ok(_):{}case .err(x):{return .err(x);}};}switch(statement_end(source,ref mut cursor)){case .ok(_):{}case .err(x):{return .err(x);}};
    }return g::publish(meta,ref names,ref data,ref objects,ref slots);
}
