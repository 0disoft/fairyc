module module_state_graph_artifact;
import module_state_graph_model as m;
import module_state_graph_program as program;
import module_state_layout as state;
import module_state_artifact as layouts;
import candidate_artifact_wire as wire;
import project_cache_encoding as budget;
import std::bytes as bytes;
import std::vec as v;

// Private detached whole-program format. Never source identity, Complete, a
// restoration ticket or a cache hit. All integers are fixed-width big endian.
// 8MiB framing and a shared 16MiB ledger bound every allocation and scan.
fn void raw(ref mut bytes::Buffer out,ref u8[] value,ref mut wire::Status status,ref mut usize work){
    if(status.code!=0){return;}if(value.len>wire::byte_limit() || bytes::len(out)>wire::byte_limit()-value.len){status.code=1002;return;}
    let code=budget::raw(out,value,work);if(code!=0){status.code=if(code==4){1003}else{1002};}
}
fn Result<bytes::Buffer,u32> data(ref u8[] input,ref mut wire::Cursor c,usize size,ref mut usize work){
    if(c.code!=0){return .err(c.code);}if(!m::range(c.offset,size,input.len)){return .err(1001);}
    if(!budget::charge(work,size+1)){return .err(1002);}let result=switch(bytes::copy(input[c.offset..c.offset+size])){case .ok(p):{p}case .err(_):{return .err(1003);}};c.offset+=size;return .ok(result);
}
fn void write_operand(ref mut bytes::Buffer out,m::Operand row,ref mut wire::Status status,ref mut usize work){
    wire::number(out,row.input,status,work);wire::number(out,row.codec as usize,status,work);wire::number(out,row.start,status,work);wire::number(out,row.end,status,work);
}
fn m::Operand read_operand(ref u8[] input,ref mut wire::Cursor c,ref mut usize work){return m::Operand{.input=wire::number_from(input,c,work),.codec=wire::word_from(input,c,work),.start=wire::number_from(input,c,work),.end=wire::number_from(input,c,work)};}
fn void write_term(ref mut bytes::Buffer out,m::Terminator row,ref mut wire::Status status,ref mut usize work){
    wire::number(out,row.kind as usize,status,work);wire::number(out,row.condition,status,work);wire::number(out,row.target,status,work);wire::number(out,row.argument_start,status,work);wire::number(out,row.argument_count,status,work);wire::number(out,row.alternative,status,work);wire::number(out,row.alternate_start,status,work);wire::number(out,row.alternate_count,status,work);wire::number(out,row.category_start,status,work);wire::number(out,row.category_end,status,work);wire::number(out,row.cost,status,work);
}
fn m::Terminator read_term(ref u8[] input,ref mut wire::Cursor c,ref mut usize work){return m::Terminator{.kind=wire::word_from(input,c,work),.condition=wire::number_from(input,c,work),.target=wire::number_from(input,c,work),.argument_start=wire::number_from(input,c,work),.argument_count=wire::number_from(input,c,work),.alternative=wire::number_from(input,c,work),.alternate_start=wire::number_from(input,c,work),.alternate_count=wire::number_from(input,c,work),.category_start=wire::number_from(input,c,work),.category_end=wire::number_from(input,c,work),.cost=wire::number_from(input,c,work)};}
fn void write_operation(ref mut bytes::Buffer out,m::Operation row,ref mut wire::Status status,ref mut usize work){
    wire::number(out,row.kind as usize,status,work);wire::number(out,row.capability,status,work);wire::number(out,row.field,status,work);
    write_operand(out,row.key,status,work);write_operand(out,row.value,status,work);
    wire::number(out,row.result,status,work);wire::number(out,row.found,status,work);wire::number(out,row.cost,status,work);wire::number(out,row.callee,status,work);wire::number(out,row.borrow_input,status,work);wire::number(out,row.borrow_result,status,work);wire::number(out,row.view_offset,status,work);wire::number(out,row.view_length,status,work);
}
fn m::Operation read_operation(ref u8[] input,ref mut wire::Cursor c,ref mut usize work){return m::Operation{.kind=wire::word_from(input,c,work),.capability=wire::number_from(input,c,work),.field=wire::number_from(input,c,work),.key=read_operand(input,c,work),.value=read_operand(input,c,work),.result=wire::number_from(input,c,work),.found=wire::number_from(input,c,work),.cost=wire::number_from(input,c,work),.callee=wire::number_from(input,c,work),.borrow_input=wire::number_from(input,c,work),.borrow_result=wire::number_from(input,c,work),.view_offset=wire::number_from(input,c,work),.view_length=wire::number_from(input,c,work)};}
fn void write_graph(ref mut bytes::Buffer out,ref m::Graph graph,ref mut wire::Status status,ref mut usize work){
    let ps=v::len<m::Value>(m::parameters(graph));let args=v::len<usize>(m::arguments(graph));let ops=v::len<m::Operation>(m::operations(graph));let bs=v::len<m::Block>(m::blocks(graph));let body=m::data(graph);
    // Bound terms before multiplying/adding even for a future verifier change.
    if(ps>wire::byte_limit()/16 || args>wire::byte_limit()/8 || ops>m::operation_limit() || bs>m::block_limit() || body.len>m::byte_limit()){status.code=1002;return;}
    let size=72+ps*16+args*8+ops*152+bs*128+body.len;if(size>wire::byte_limit()){status.code=1002;return;}
    wire::number(out,size,status,work);let h=m::header(graph);
    wire::number(out,h.effects as usize,status,work);wire::number(out,h.capability,status,work);wire::number(out,h.layout,status,work);wire::number(out,h.entry,status,work);
    wire::number(out,ps,status,work);wire::number(out,args,status,work);wire::number(out,ops,status,work);wire::number(out,bs,status,work);wire::number(out,body.len,status,work);
    var usize i=0;while(i<ps && status.code==0){let row=*v::at<m::Value>(m::parameters(graph),i);wire::number(out,row.id,status,work);wire::number(out,row.codec as usize,status,work);i+=1;}
    i=0;while(i<args && status.code==0){wire::number(out,*v::at<usize>(m::arguments(graph),i),status,work);i+=1;}
    i=0;while(i<ops && status.code==0){write_operation(out,*v::at<m::Operation>(m::operations(graph),i),status,work);i+=1;}
    i=0;while(i<bs && status.code==0){let row=*v::at<m::Block>(m::blocks(graph),i);wire::number(out,row.id,status,work);wire::number(out,row.parameter_start,status,work);wire::number(out,row.parameter_count,status,work);wire::number(out,row.operation_start,status,work);wire::number(out,row.operation_count,status,work);write_term(out,row.term,status,work);i+=1;}
    raw(out,body,status,work);
}
fn Result<m::Graph,u32> read_graph(ref u8[] packet,ref mut wire::Cursor outer,ref mut usize remaining,ref mut usize work){
    let length=wire::count_from(packet,outer,wire::byte_limit(),1,work);if(outer.code!=0){return .err(outer.code);}
    let input=packet[outer.offset..outer.offset+length];outer.offset+=length;var c=wire::Cursor{.offset=0,.code=0};
    let effects=wire::word_from(input,ref mut c,work);let capability=wire::number_from(input,ref mut c,work);let layout=wire::number_from(input,ref mut c,work);let entry=wire::number_from(input,ref mut c,work);
    let ps=wire::count_from(input,ref mut c,wire::byte_limit()/16,16,work);let args=wire::count_from(input,ref mut c,wire::byte_limit()/8,8,work);
    let ops=wire::count_from(input,ref mut c,*remaining,152,work);let bs=wire::count_from(input,ref mut c,m::block_limit(),128,work);let size=wire::count_from(input,ref mut c,m::byte_limit(),1,work);
    if(c.code!=0){return .err(c.code);}
    // The entire section sum must fit this function frame BEFORE any Vec push.
    let fixed=ps*16+args*8+ops*152+bs*128;if(!m::range(c.offset,fixed,input.len) || size!=input.len-c.offset-fixed){return .err(1001);};*remaining-=ops;
    var v::Vec<m::Value> parameters=v::empty<m::Value>();var v::Vec<usize> arguments=v::empty<usize>();var v::Vec<m::Operation> operations=v::empty<m::Operation>();var v::Vec<m::Block> blocks=v::empty<m::Block>();var usize i=0;
    while(i<ps){let row=m::Value{.id=wire::number_from(input,ref mut c,work),.codec=wire::word_from(input,ref mut c,work)};if(c.code!=0){return .err(c.code);}if(!v::push<m::Value>(ref mut parameters,row).is_ok){return .err(1003);}i+=1;}
    i=0;while(i<args){let row=wire::number_from(input,ref mut c,work);if(c.code!=0){return .err(c.code);}if(!v::push<usize>(ref mut arguments,row).is_ok){return .err(1003);}i+=1;}
    i=0;while(i<ops){let row=read_operation(input,ref mut c,work);if(c.code!=0){return .err(c.code);}
        if(row.key.start>row.key.end || row.key.end>size || row.value.start>row.value.end || row.value.end>size){return .err(1001);}
        if(!v::push<m::Operation>(ref mut operations,row).is_ok){return .err(1003);}i+=1;}
    i=0;while(i<bs){let id=wire::number_from(input,ref mut c,work);let start=wire::number_from(input,ref mut c,work);let count=wire::number_from(input,ref mut c,work);let op_start=wire::number_from(input,ref mut c,work);let op_count=wire::number_from(input,ref mut c,work);let term=read_term(input,ref mut c,work);
        if(c.code!=0){return .err(c.code);}if(!m::range(start,count,ps) || !m::range(op_start,op_count,ops) || !m::range(term.argument_start,term.argument_count,args) || !m::range(term.alternate_start,term.alternate_count,args) || term.category_start>term.category_end || term.category_end>size){return .err(1001);}
        if(!v::push<m::Block>(ref mut blocks,m::Block{.id,.parameter_start=start,.parameter_count=count,.operation_start=op_start,.operation_count=op_count,.term}).is_ok){return .err(1003);}i+=1;}
    let body=switch(data(input,ref mut c,size,work)){case .ok(p):{p}case .err(code):{return .err(code);}};let code=wire::end(input,ref mut c,work);if(code!=0){return .err(code);}
    return .ok(m::assemble(m::Header{.effects,.capability,.layout,.entry},body,parameters,arguments,operations,blocks));
}
pub fn Result<wire::Payload,u32> encode(ref program::Verified proof,ref mut usize work){
    if(!budget::charge(work,0)){return .err(1002);}let p=program::program(proof);let code=program::validate(p,program::layouts(proof),work);if(code!=0){return .err(code);}
    let plan=switch(layouts::encode_graph(program::layouts(proof),work)){case .ok(p):{p}case .err(code):{return .err(code);}};
    var status=wire::Status{.code=0};var out=wire::writer(b"FYSGRPH0",ref mut status,work);wire::number(ref mut out,wire::content(ref plan).len,ref mut status,work);raw(ref mut out,wire::content(ref plan),ref mut status,work);
    let count=program::helper_count(proof);var usize name_size=0;var usize i=0;while(i<count){let size=program::helper_name(proof,i).len;if(size>m::byte_limit()-name_size){return .err(1002);}name_size+=size;i+=1;}
    wire::number(ref mut out,count,ref mut status,work);wire::number(ref mut out,name_size,ref mut status,work);
    i=0;var usize previous=0;while(i<count && status.code==0){wire::number(ref mut out,previous,ref mut status,work);previous+=program::helper_name(proof,i).len;wire::number(ref mut out,previous,ref mut status,work);i+=1;}
    write_graph(ref mut out,program::root(proof),ref mut status,work);i=0;while(i<count && status.code==0){write_graph(ref mut out,program::helper(proof,i),ref mut status,work);i+=1;}
    i=0;while(i<count && status.code==0){raw(ref mut out,program::helper_name(proof,i),ref mut status,work);i+=1;}return wire::finish(out,status);
}
pub fn Result<program::Verified,u32> decode(ref u8[] input,ref mut usize work){
    if(!budget::charge(work,0)){return .err(1002);}var c=switch(wire::cursor(input,b"FYSGRPH0",work)){case .ok(c):{c}case .err(code):{return .err(code);}};
    let size=wire::count_from(input,ref mut c,wire::byte_limit(),1,work);if(c.code!=0){return .err(c.code);}
    let plan=switch(layouts::decode_graph(input[c.offset..c.offset+size],work)){case .ok(p):{p}case .err(code):{return .err(code);}};c.offset+=size;
    let count=wire::count_from(input,ref mut c,program::helper_limit(),16,work);let name_size=wire::count_from(input,ref mut c,m::byte_limit(),1,work);if(c.code!=0){return .err(c.code);}
    var v::Vec<program::Name> names=v::empty<program::Name>();var usize i=0;var usize previous=0;
    while(i<count){let start=wire::number_from(input,ref mut c,work);let end=wire::number_from(input,ref mut c,work);if(c.code!=0){return .err(c.code);}if(start!=previous || start>end || end>name_size){return .err(1001);}previous=end;
        if(!v::push<program::Name>(ref mut names,program::Name{.start,.end}).is_ok){return .err(1003);}i+=1;}if(previous!=name_size){return .err(1001);}
    var usize remaining=m::operation_limit();let root=switch(read_graph(input,ref mut c,ref mut remaining,work)){case .ok(p):{p}case .err(code):{return .err(code);}};
    var v::Vec<m::Graph> helpers=v::empty<m::Graph>();i=0;while(i<count){let graph=switch(read_graph(input,ref mut c,ref mut remaining,work)){case .ok(p):{p}case .err(code):{return .err(code);}};if(!v::push<m::Graph>(ref mut helpers,graph).is_ok){return .err(1003);}i+=1;}
    let name_data=switch(data(input,ref mut c,name_size,work)){case .ok(p):{p}case .err(code):{return .err(code);}};let code=wire::end(input,ref mut c,work);if(code!=0){return .err(code);}
    return program::verify(program::assemble(root,helpers,names,name_data),ref plan,work);
}
