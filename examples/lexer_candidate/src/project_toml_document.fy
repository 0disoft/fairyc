module project_toml_document;
import project_toml_model as m;
import project_toml_context as context;
import project_toml_strings as strings;
import project_toml_scalars as atoms;
import std::bytes as bytes;
import std::text as text;
import std::vec as v;

// Manifest syntax retains typed atoms so valid unknown fields and malformed
// values preserve the executable Go decoder's rejection categories.
struct Cursor{usize offset;usize depth;bool compact;}
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
// Mode 0 reserves an assignment leaf, 1 declares a table, 2 appends a table
// array entry. Header prefixes follow the most recent array-table entry.
fn Result<usize,m::Problem> path(ref u8[] source,ref mut Cursor cursor,ref mut bytes::Buffer context_names,ref mut v::Vec<context::Entry> contexts,ref mut v::Vec<usize> context_buckets,ref mut m::State meta,ref mut bytes::Buffer names,ref mut v::Vec<m::Node> nodes,ref mut v::Vec<usize> buckets,usize root,u32 mode){
    var usize parent=root;var usize logical=switch(context::physical(context_names,contexts,context_buckets,meta,names,nodes,root)){case .ok(id):{id}case .err(e):{return .err(e);}};
    while(true){var bytes::Buffer key=bytes::empty();let usize start=switch(component(source,cursor,ref mut key)){case .ok(n):{n}case .err(e):{return .err(e);}};let usize end=cursor.offset;horizontal(source,cursor);let bool more=peek(source,cursor)==46;
        logical=switch(context::intern(context_names,contexts,context_buckets,meta,logical,bytes::view(ref key),start,end)){case .ok(n):{n}case .err(e):{return .err(e);}};
        let usize existing=switch(m::locate(meta,names,nodes,buckets,parent,bytes::view(ref key),start,end)){case .ok(n):{n}case .err(e):{return .err(e);}};
        if(more){
            if(mode==0 || existing==0){switch(context::mark_implicit(contexts,meta,logical)){case .ok(_):{}case .err(e):{return .err(fail(source,e.code,start,end));}};}
            if(existing==0){parent=switch(m::reserve_key(meta,names,nodes,buckets,parent,bytes::view(ref key),start,end)){case .ok(n):{n}case .err(e):{return .err(e);}};switch(m::container(meta,nodes,parent,1,if(mode==0){1}else{0},end)){case .ok(_):{}case .err(e):{return .err(e);}};}
            else{var m::Node n=m::entry(nodes,existing);if(n.state==3){return .err(fail(source,1,start,end));}
                if(mode==0){let u32 action=switch(context::define(contexts,meta,logical,true)){case .ok(n):{n}case .err(e):{return .err(fail(source,e.code,start,end));}};
                    if(action==2){switch(m::replace_table(meta,nodes,existing)){case .ok(_):{}case .err(e):{return .err(e);}};cursor.compact=true;n=m::entry(nodes,existing);}
                }
                if(n.kind==1){parent=existing;}else if(n.kind==4 && n.last!=0){parent=n.last;}else{return .err(fail(source,1,start,end));}
            }cursor.offset+=1;continue;
        }
        if(mode==0){if(existing!=0){return .err(fail(source,1,start,end));}return m::reserve_key(meta,names,nodes,buckets,parent,bytes::view(ref key),start,end);}
        var usize id=existing;
        if(mode==1){switch(context::define(contexts,meta,logical,id!=0)){case .ok(_):{}case .err(e):{return .err(fail(source,e.code,start,end));}};}
        if(id==0){id=switch(m::reserve_key(meta,names,nodes,buckets,parent,bytes::view(ref key),start,end)){case .ok(n):{n}case .err(e):{return .err(e);}};switch(m::container(meta,nodes,id,if(mode==2){4}else{1},if(mode==2){0}else{2},end)){case .ok(_):{}case .err(e):{return .err(e);}};}
        else if(mode==1){switch(m::define_table(meta,nodes,id,start,end)){case .ok(_):{}case .err(e):{return .err(e);}};}
        else if(m::entry(nodes,id).kind!=4 || m::entry(nodes,id).state==3){return .err(fail(source,1,start,end));}
        switch(context::set_kind(contexts,meta,logical,if(mode==2){4}else{1})){case .ok(_):{}case .err(e):{return .err(fail(source,e.code,start,end));}};
        if(mode==1){return .ok(id);}
        let usize item=switch(m::element(meta,names,nodes,buckets,id,start,end)){case .ok(n):{n}case .err(e):{return .err(e);}};return m::container(meta,nodes,item,1,2,end);
    }return .err(here(source,cursor));
}
fn Result<usize,m::Problem> value(ref u8[] source,ref mut Cursor cursor,ref mut bytes::Buffer context_names,ref mut v::Vec<context::Entry> contexts,ref mut v::Vec<usize> context_buckets,ref mut m::State meta,ref mut bytes::Buffer names,ref mut bytes::Buffer values,ref mut v::Vec<m::Node> nodes,ref mut v::Vec<usize> buckets,usize id){
    horizontal(source,cursor);let usize start=cursor.offset;let u8 c=peek(source,cursor);
    if(c==34 || c==39){var bytes::Buffer cooked=bytes::empty();let usize end=switch(strings::append(source,start,true,ref mut cooked)){case .ok(n):{n}case .err(e):{return .err(fail(source,e.code,e.start,e.end));}};cursor.offset=end;return m::string(meta,values,nodes,id,bytes::view(ref cooked),end);}
    if(source.len-start>=4 && bytes::equal(source[start..start+4],b"true")){cursor.offset+=4;return m::boolean(meta,values,nodes,id,true,cursor.offset);}
    if(source.len-start>=5 && bytes::equal(source[start..start+5],b"false")){cursor.offset+=5;return m::boolean(meta,values,nodes,id,false,cursor.offset);}
    if(c!=91 && c!=123){let atoms::Token token=atoms::read(source,start);if(token.kind==0){return .err(fail(source,1,start,token.end));}cursor.offset=token.end;return m::atom(meta,values,nodes,id,source[start..token.end],token.end,token.kind);}if(cursor.depth>=512){return .err(fail(source,4,start,start+1));}cursor.depth+=1;
    let result=if(c==91){array(source,cursor,context_names,contexts,context_buckets,meta,names,values,nodes,buckets,id)}else{inline_table(source,cursor,context_names,contexts,context_buckets,meta,names,values,nodes,buckets,id)};cursor.depth-=1;return result;
}
fn Result<usize,m::Problem> array(ref u8[] source,ref mut Cursor cursor,ref mut bytes::Buffer context_names,ref mut v::Vec<context::Entry> contexts,ref mut v::Vec<usize> context_buckets,ref mut m::State meta,ref mut bytes::Buffer names,ref mut bytes::Buffer values,ref mut v::Vec<m::Node> nodes,ref mut v::Vec<usize> buckets,usize id){
    let usize start=cursor.offset;cursor.offset+=1;switch(m::container(meta,nodes,id,3,0,cursor.offset)){case .ok(_):{}case .err(e):{return .err(e);}};
    while(true){switch(trivia(source,cursor,true)){case .ok(_):{}case .err(e):{return .err(e);}};if(peek(source,cursor)==93){cursor.offset+=1;return m::finish_container(meta,nodes,id,cursor.offset);}
        if(cursor.offset==source.len){return .err(fail(source,1,start,cursor.offset));}
        let usize item=switch(m::element(meta,names,nodes,buckets,id,cursor.offset,cursor.offset)){case .ok(n):{n}case .err(e):{return .err(e);}};switch(value(source,cursor,context_names,contexts,context_buckets,meta,names,values,nodes,buckets,item)){case .ok(_):{}case .err(e):{return .err(e);}};
        switch(trivia(source,cursor,true)){case .ok(_):{}case .err(e):{return .err(e);}};let u8 c=peek(source,cursor);if(c==44){cursor.offset+=1;continue;}if(c!=93){return .err(here(source,cursor));}
    }return .err(here(source,cursor));
}
fn Result<usize,m::Problem> assignment(ref u8[] source,ref mut Cursor cursor,ref mut bytes::Buffer context_names,ref mut v::Vec<context::Entry> contexts,ref mut v::Vec<usize> context_buckets,ref mut m::State meta,ref mut bytes::Buffer names,ref mut bytes::Buffer values,ref mut v::Vec<m::Node> nodes,ref mut v::Vec<usize> buckets,usize parent){
    let usize id=switch(path(source,cursor,context_names,contexts,context_buckets,meta,names,nodes,buckets,parent,0)){case .ok(n):{n}case .err(e):{return .err(e);}};
    if(peek(source,cursor)!=61){return .err(here(source,cursor));}cursor.offset+=1;switch(value(source,cursor,context_names,contexts,context_buckets,meta,names,values,nodes,buckets,id)){case .ok(_):{}case .err(e):{return .err(e);}};
    let usize logical=switch(context::physical(context_names,contexts,context_buckets,meta,names,nodes,id)){case .ok(n):{n}case .err(e):{return .err(e);}};
    switch(context::record_type(contexts,meta,logical,m::entry(nodes,id).kind)){case .ok(_):{}case .err(e):{return .err(e);}};return .ok(id);
}
fn Result<usize,m::Problem> inline_table(ref u8[] source,ref mut Cursor cursor,ref mut bytes::Buffer context_names,ref mut v::Vec<context::Entry> contexts,ref mut v::Vec<usize> context_buckets,ref mut m::State meta,ref mut bytes::Buffer names,ref mut bytes::Buffer values,ref mut v::Vec<m::Node> nodes,ref mut v::Vec<usize> buckets,usize id){
    let usize start=cursor.offset;cursor.offset+=1;switch(m::container(meta,nodes,id,1,2,cursor.offset)){case .ok(_):{}case .err(e):{return .err(e);}};
    while(true){switch(trivia(source,cursor,true)){case .ok(_):{}case .err(e):{return .err(e);}};if(peek(source,cursor)==125){cursor.offset+=1;return m::close(meta,nodes,id,cursor.offset);}
        if(cursor.offset==source.len){return .err(fail(source,1,start,cursor.offset));}switch(assignment(source,cursor,context_names,contexts,context_buckets,meta,names,values,nodes,buckets,id)){case .ok(_):{}case .err(e):{return .err(e);}};
        switch(trivia(source,cursor,true)){case .ok(_):{}case .err(e):{return .err(e);}};let u8 c=peek(source,cursor);if(c==44){cursor.offset+=1;continue;}if(c!=125){return .err(here(source,cursor));}
    }return .err(here(source,cursor));
}
fn Result<usize,m::Problem> statement_end(ref u8[] source,ref mut Cursor cursor){
    horizontal(source,cursor);if(peek(source,cursor)==35){cursor.offset+=1;while(cursor.offset<source.len && peek(source,cursor)!=10 && peek(source,cursor)!=13){let u8 c=peek(source,cursor);if(c<32 && c!=9 || c==127){return .err(here(source,cursor));}cursor.offset+=1;}}
    if(cursor.offset==source.len){return .ok(cursor.offset);}return newline(source,cursor);
}
pub fn Result<m::Tree,m::Problem> parse(ref u8[] source){
    if(source.len>1048576){return .err(fail(source,4,0,0));}if(!text::validate_utf8(source).is_ok){return .err(fail(source,2,0,source.len));}
    var bytes::Buffer context_names=bytes::empty();var v::Vec<context::Entry> contexts=v::empty<context::Entry>();var v::Vec<usize> context_buckets=v::empty<usize>();
    var m::State meta=m::state(source.len);var bytes::Buffer names=bytes::empty();var bytes::Buffer values=bytes::empty();var v::Vec<m::Node> nodes=v::empty<m::Node>();var v::Vec<usize> buckets=v::empty<usize>();
    let u32 initial=m::initialize(ref meta,ref mut nodes);if(initial!=0){return .err(fail(source,initial,0,0));}var Cursor cursor=Cursor{.offset=0,.depth=0,.compact=false};var usize table=1;
    while(true){switch(trivia(source,ref mut cursor,true)){case .ok(_):{}case .err(e):{return .err(e);}};if(cursor.offset==source.len){break;}
        if(peek(source,ref cursor)==91){cursor.offset+=1;let bool many=peek(source,ref cursor)==91;if(many){cursor.offset+=1;}
            table=switch(path(source,ref mut cursor,ref mut context_names,ref mut contexts,ref mut context_buckets,ref mut meta,ref mut names,ref mut nodes,ref mut buckets,1,if(many){2}else{1})){case .ok(n):{n}case .err(e):{return .err(e);}};
            if(peek(source,ref cursor)!=93){return .err(here(source,ref cursor));}cursor.offset+=1;if(many){if(peek(source,ref cursor)!=93){return .err(here(source,ref cursor));}cursor.offset+=1;}
            switch(m::finish_container(ref meta,ref mut nodes,table,cursor.offset)){case .ok(_):{}case .err(e):{return .err(e);}};
        }else{switch(assignment(source,ref mut cursor,ref mut context_names,ref mut contexts,ref mut context_buckets,ref mut meta,ref mut names,ref mut values,ref mut nodes,ref mut buckets,table)){case .ok(_):{}case .err(e):{return .err(e);}};}
        switch(statement_end(source,ref mut cursor)){case .ok(_):{}case .err(e):{return .err(e);}};
    }return if(cursor.compact){m::publish_compacted(meta,names,values,nodes,buckets)}else{m::publish(meta,names,values,nodes,buckets)};
}
