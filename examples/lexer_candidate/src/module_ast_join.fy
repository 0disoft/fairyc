module module_ast_join;
import candidate_limits as limits;
import syntax as syntax;
import scanner as scanner;
import body_state as body;
import bodies as bodies;
import types as types;
import records as records;
import attributes as attributes;
import std::bytes as bytes;
import std::vec as v;
import sdk_authority as sdk;

// Exact span/arena remapping and budgets: docs/architecture/candidate-source-join-v0.md.
pub struct Problem { u32 code; usize module_index; usize start; usize end; }
fn Problem fail(u32 code,usize module_index,usize start,usize end){return Problem{.code,.module_index,.start,.end};}

fn usize moved(usize value,usize offset){return value+offset;}
fn usize present(usize value,usize offset){return if(value==0){0}else{value+offset};}
fn usize linked(usize value,usize base){return if(value==0){0}else{value+base};}
fn usize indexed(usize count,usize value,usize base){return if(count==0){0}else{value+base};}
fn usize token_at(ref v::Vec<usize> tokens,usize id){assert(id<v::len<usize>(tokens));return *v::at<usize>(tokens,id);}

fn syntax::Declaration declaration(syntax::Declaration d,usize offset,usize expressions,usize statements,usize type_base,usize entries,usize fields,usize attributes_base){
    return syntax::Declaration{
        .cleanup_attribute_start=indexed(d.cleanup_attribute_count,d.cleanup_attribute_start,attributes_base),.cleanup_attribute_count=d.cleanup_attribute_count,
        .unsafe_start=present(d.unsafe_start,offset),.unsafe_end=present(d.unsafe_end,offset),.owner_start=present(d.owner_start,offset),.owner_end=present(d.owner_end,offset),
        .kind=d.kind,.start=moved(d.start,offset),.end=moved(d.end,offset),.name_start=present(d.name_start,offset),.name_end=present(d.name_end,offset),
        .expression=linked(d.expression,expressions),.entry_start=indexed(d.entry_count,d.entry_start,entries),.entry_count=d.entry_count,
        .body=linked(d.body,statements),.result_type=linked(d.result_type,type_base),
        .field_start=indexed(d.field_count,d.field_start,fields),.field_count=d.field_count,
        .public=d.public,.public_start=present(d.public_start,offset),.public_end=present(d.public_end,offset),
        .attribute_start=indexed(d.attribute_count,d.attribute_start,attributes_base),.attribute_count=d.attribute_count,
        .cleanup_start=present(d.cleanup_start,offset),.cleanup_end=present(d.cleanup_end,offset),.cleanup_body=linked(d.cleanup_body,statements),
        .generic_parameters=linked(d.generic_parameters,type_base),.generic_start=present(d.generic_start,offset),.generic_end=present(d.generic_end,offset)
    };
}

fn body::Node expression(syntax::Expression n,usize offset,usize expressions,usize statements,usize type_base,ref v::Vec<usize> tokens){
    let usize left=if(n.kind==12 || n.kind==14 || n.kind==31){linked(n.left,statements)}else{linked(n.left,expressions)};
    var usize right=linked(n.right,expressions);
    if(n.kind==18){right=linked(n.right,statements);}else if(n.kind==20 || n.kind==23){right=linked(n.right,type_base);}
    return body::Node{.kind=n.kind,.start=moved(n.start,offset),.end=moved(n.end,offset),.token=token_at(tokens,n.token),.left,.right};
}

fn body::Statement statement(syntax::Statement s,usize offset,usize expressions,usize statements,usize type_base,usize entries,ref v::Vec<usize> tokens){
    var usize second=s.second;
    if(s.kind==7 || s.kind==14 || s.kind==15){second=linked(s.second,statements);}else if(s.kind==18){second=linked(s.second,type_base);}
    let usize operator=if(s.kind==4){token_at(tokens,s.operator)}else{0};
    return body::Statement{.kind=s.kind,.start=moved(s.start,offset),.end=moved(s.end,offset),
        .expression=linked(s.expression,expressions),.target=linked(s.target,expressions),.first=linked(s.first,statements),
        .second,.entry=linked(s.entry,entries),.operator};
}

fn types::Node type_node(types::Node n,usize offset,usize type_base){
    return types::Node{.kind=n.kind,.start=moved(n.start,offset),.end=moved(n.end,offset),.left=linked(n.left,type_base),.right=linked(n.right,type_base)};
}

fn bodies::Entry entry(syntax::Entry e,usize id,usize offset,usize expressions,usize type_base){
    return bodies::Entry{.kind=e.kind,.start=moved(e.start,offset),.end=moved(e.end,offset),
        .name_start=present(e.name_start,offset),.name_end=present(e.name_end,offset),
        .type_start=present(e.type_start,offset),.type_end=present(e.type_end,offset),
        .type_root=linked(e.type_root,type_base),.expression=linked(e.expression,expressions),.id};
}

fn records::Field field(records::Field f,usize offset,usize type_base,usize fields_base){
    return records::Field{.kind=f.kind,.start=moved(f.start,offset),.end=moved(f.end,offset),
        .name_start=moved(f.name_start,offset),.name_end=moved(f.name_end,offset),
        .type_root=linked(f.type_root,type_base),.key_type=linked(f.key_type,type_base),
        .public=f.public,.public_start=present(f.public_start,offset),.public_end=present(f.public_end,offset),
        .payload_start=indexed(f.payload_count,f.payload_start,fields_base),.payload_count=f.payload_count,
        .payload_open=present(f.payload_open,offset),.payload_end=present(f.payload_end,offset)};
}

fn attributes::Attribute attribute(attributes::Attribute a,usize offset,usize arguments_base){
    return attributes::Attribute{.start=moved(a.start,offset),.end=moved(a.end,offset),
        .name_start=moved(a.name_start,offset),.name_end=moved(a.name_end,offset),
        .argument_start=indexed(a.argument_count,a.argument_start,arguments_base),.argument_count=a.argument_count};
}

fn attributes::Argument argument(attributes::Argument a,usize offset,ref v::Vec<usize> tokens){
    return attributes::Argument{.kind=a.kind,.start=moved(a.start,offset),.end=moved(a.end,offset),.token=token_at(tokens,a.token)};
}

fn Problem generic(ref syntax::Parsed file,usize module_index){
    var usize i=0;while(i<syntax::declaration_count(file)){let d=syntax::declaration_at(file,i);
        let bool vector=d.kind==6 && bytes::equal(syntax::source_view(file)[d.name_start..d.name_end],b"Vec") && sdk::authentic(b"std::vec",syntax::source_view(file));
        if(d.generic_parameters!=0 && (d.kind!=3 && !vector || d.owner_end!=0)){return fail(4501,module_index,d.start,d.end);}
        i+=1;
    }
    return Problem{.code=0,.module_index=0,.start=0,.end=0};
}

pub fn Result<syntax::Parsed,Problem> join(ref v::Vec<syntax::Parsed> sources){
    let usize count=v::len<syntax::Parsed>(sources);
    if(count==0 || count>limits::source_files()){return .err(fail(10,0,0,0));}
    var bytes::Buffer joined=bytes::empty();
    var usize index=0;
    while(index<count){
        let ref syntax::Parsed file=v::at<syntax::Parsed>(sources,index);
        if(syntax::problem_count(file)!=0){return .err(fail(9,index,0,syntax::source_view(file).len));}
        if(syntax::declaration_count(file)==0 || syntax::declaration_at(file,0).kind!=1){return .err(fail(9,index,0,0));}
        let Problem rejected=generic(file,index);
        if(rejected.code!=0){return .err(rejected);}
        let usize separator=if(index+1<count){1}else{0};
        if(syntax::source_view(file).len>limits::source_bytes() || bytes::len(ref joined)>limits::program_bytes()-separator || syntax::source_view(file).len>limits::program_bytes()-separator-bytes::len(ref joined)){return .err(fail(10,index,0,syntax::source_view(file).len));}
        if(!bytes::append(ref mut joined,syntax::source_view(file)).is_ok){return .err(fail(11,index,0,0));}
        if(index+1<count && !bytes::append(ref mut joined,b"\n").is_ok){return .err(fail(11,index,0,0));}
        if(bytes::len(ref joined)>limits::program_bytes()){return .err(fail(10,index,0,syntax::source_view(file).len));}
        index+=1;
    }
    let scanner::Stream stream=switch(scanner::collect_joined(joined)){
        case .ok(value):{value}
        case .err(code):{return .err(fail(if(code==3){11}else if(code==4){10}else{9},0,0,0));}
    };
    if(scanner::diagnostic_count(ref stream)!=0){return .err(fail(if(scanner::diagnostic_info(ref stream,0).code==10){10}else{9},0,0,0));}
    var v::Vec<syntax::Declaration> declarations=v::empty<syntax::Declaration>();
    var v::Vec<syntax::Problem> problems=v::empty<syntax::Problem>();
    var v::Vec<body::Node> expressions=v::empty<body::Node>();
    var v::Vec<body::Statement> statements=v::empty<body::Statement>();
    var v::Vec<bodies::Entry> entries=v::empty<bodies::Entry>();
    var v::Vec<types::Node> types_arena=v::empty<types::Node>();
    var v::Vec<records::Field> fields=v::empty<records::Field>();
    var v::Vec<attributes::Attribute> attributes_arena=v::empty<attributes::Attribute>();
    var v::Vec<attributes::Argument> arguments=v::empty<attributes::Argument>();
    var v::Vec<usize> tokens=v::empty<usize>();
    var usize offset=0;
    index=0;
    while(index<count){
        let ref syntax::Parsed file=v::at<syntax::Parsed>(sources,index);
        discard v::clear<usize>(ref mut tokens);
        var usize cursor=0; var usize position_of=0;
        while(position_of<syntax::token_count(file)){
            let usize position=offset+syntax::token_info(file,position_of).start;
            while(cursor<scanner::token_count(ref stream) && scanner::token_info(ref stream,cursor).start<position){cursor+=1;}
            var usize mapped=0;
            if(cursor<scanner::token_count(ref stream) && scanner::token_info(ref stream,cursor).start==position){mapped=cursor;}
            else if(cursor>0 && scanner::token_info(ref stream,cursor-1).start<=position && position<scanner::token_info(ref stream,cursor-1).end){mapped=cursor-1;}
            else{return .err(fail(9,index,syntax::token_info(file,position_of).start,syntax::token_info(file,position_of).end));}
            if(!v::push<usize>(ref mut tokens,mapped).is_ok){return .err(fail(11,index,0,0));}
            position_of+=1;
        }
        let usize expression_base=v::len<body::Node>(ref expressions);
        let usize statement_base=v::len<body::Statement>(ref statements);
        let usize type_base=v::len<types::Node>(ref types_arena);
        let usize entry_base=v::len<bodies::Entry>(ref entries);
        let usize field_base=v::len<records::Field>(ref fields);
        let usize attribute_base=v::len<attributes::Attribute>(ref attributes_arena);
        let usize argument_base=v::len<attributes::Argument>(ref arguments);
        var usize at=0;
        while(at<syntax::declaration_count(file)){
            var syntax::Declaration d=declaration(syntax::declaration_at(file,at),offset,expression_base,statement_base,type_base,entry_base,field_base,attribute_base);
            // A joined module owns the complete original file, including trivia.
            if(d.kind==1){d.start=offset;d.end=offset+syntax::source_view(file).len;}
            if(!v::push<syntax::Declaration>(ref mut declarations,d).is_ok){return .err(fail(11,index,0,0));}
            at+=1;}
        at=1;
        while(at<=syntax::expression_count(file)){
            if(!v::push<body::Node>(ref mut expressions,expression(syntax::expression_at(file,at),offset,expression_base,statement_base,type_base,ref tokens)).is_ok){return .err(fail(11,index,0,0));}
            at+=1;}
        at=1;
        while(at<=syntax::statement_count(file)){
            if(!v::push<body::Statement>(ref mut statements,statement(syntax::statement_at(file,at),offset,expression_base,statement_base,type_base,entry_base,ref tokens)).is_ok){return .err(fail(11,index,0,0));}
            at+=1;}
        at=0;
        while(at<syntax::entry_count(file)){
            if(!v::push<bodies::Entry>(ref mut entries,entry(syntax::entry_at(file,at),at+1+entry_base,offset,expression_base,type_base)).is_ok){return .err(fail(11,index,0,0));}
            at+=1;}
        at=1;
        while(at<=syntax::type_count(file)){
            if(!v::push<types::Node>(ref mut types_arena,type_node(syntax::type_at(file,at),offset,type_base)).is_ok){return .err(fail(11,index,0,0));}
            at+=1;}
        at=0;
        while(at<syntax::field_count(file)){
            if(!v::push<records::Field>(ref mut fields,field(syntax::field_at(file,at),offset,type_base,field_base)).is_ok){return .err(fail(11,index,0,0));}
            at+=1;}
        at=0;
        while(at<syntax::attribute_count(file)){
            if(!v::push<attributes::Attribute>(ref mut attributes_arena,attribute(syntax::attribute_at(file,at),offset,argument_base)).is_ok){return .err(fail(11,index,0,0));}
            at+=1;}
        at=0;
        while(at<syntax::attribute_argument_count(file)){
            if(!v::push<attributes::Argument>(ref mut arguments,argument(syntax::attribute_argument_at(file,at),offset,ref tokens)).is_ok){return .err(fail(11,index,0,0));}
            at+=1;}
        if(v::len<syntax::Declaration>(ref declarations)>512 || v::len<body::Node>(ref expressions)>limits::program_expressions() || v::len<body::Statement>(ref statements)>8192){return .err(fail(10,index,0,0));}
        if(v::len<bodies::Entry>(ref entries)>1024 || v::len<types::Node>(ref types_arena)>4096 || v::len<records::Field>(ref fields)>1024){return .err(fail(10,index,0,0));}
        if(v::len<attributes::Attribute>(ref attributes_arena)>4096 || v::len<attributes::Argument>(ref arguments)>4096){return .err(fail(10,index,0,0));}
        offset+=syntax::source_view(file).len+1;
        index+=1;
    }
    return .ok(syntax::assemble(stream,declarations,problems,expressions,entries,statements,types_arena,fields,attributes_arena,arguments));
}
