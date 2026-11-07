module copied_closure_tables;
import std::bytes as bytes;
import std::vec as v;

// Written function JSON and generated helper names are independently owned.
// No generated spelling or function body is inserted into source bytes.
pub struct Row{usize declaration;usize name_start;usize name_end;usize json_start;usize json_end;usize value_captures;}
pub resource Table{bytes::Buffer data;v::Vec<Row> rows;cleanup{}}
pub fn Table empty(){return Table{.data=bytes::empty(),.rows=v::empty<Row>()};}
pub fn usize count(ref Table p){return v::len<Row>(ref p.rows);}
pub fn ref u8[] name(ref Table p,usize declaration){var usize i=0;while(i<count(p)){let r=*v::at<Row>(ref p.rows,i);if(r.declaration==declaration && r.name_start!=r.name_end){return bytes::view(ref p.data)[r.name_start..r.name_end];}i+=1;}return bytes::view(ref p.data)[0..0];}
pub fn ref u8[] written(ref Table p,usize declaration){var usize i=0;while(i<count(p)){let r=*v::at<Row>(ref p.rows,i);if(r.declaration==declaration && r.json_start!=r.json_end){return bytes::view(ref p.data)[r.json_start..r.json_end];}i+=1;}return bytes::view(ref p.data)[0..0];}
pub fn Table assemble(bytes::Buffer data,v::Vec<Row> rows){return Table{.data,.rows};}
fn u32 add_row(ref mut bytes::Buffer data,ref mut v::Vec<Row> rows,usize declaration,ref u8[] name,ref u8[] json,usize value_captures){if(v::len<Row>(rows)>=1024 || bytes::len(data)>1048576 || name.len+json.len>1048576-bytes::len(data)){return 10;}let first=bytes::len(data);if(!bytes::append(data,name).is_ok || !bytes::append(data,json).is_ok){return 11;}let end=bytes::len(data);return if(v::push<Row>(rows,Row{.declaration,.name_start=first,.name_end=first+name.len,.json_start=first+name.len,.json_end=end,.value_captures}).is_ok){0}else{11};}
// Zero retains the old direct-call lane; values encode capture count plus one.
pub fn usize value_captures(ref Table p,usize declaration){var usize i=0;while(i<count(p)){let r=*v::at<Row>(ref p.rows,i);if(r.declaration==declaration && r.name_start!=r.name_end){return r.value_captures;}i+=1;}return 0;}
pub fn u32 add(ref mut bytes::Buffer data,ref mut v::Vec<Row> rows,usize declaration,ref u8[] name,ref u8[] json){return add_row(data,rows,declaration,name,json,0);}
pub fn u32 add_value(ref mut bytes::Buffer data,ref mut v::Vec<Row> rows,usize declaration,ref u8[] name,ref u8[] json,usize captures){if(captures>4 || name.len==0){return 3;}return add_row(data,rows,declaration,name,json,captures+1);}
pub fn u32 append(ref mut bytes::Buffer data,ref mut v::Vec<Row> rows,ref Table input,usize declaration_base){var usize i=0;while(i<count(input)){let r=*v::at<Row>(ref input.rows,i);let code=add_row(data,rows,r.declaration+declaration_base,bytes::view(ref input.data)[r.name_start..r.name_end],bytes::view(ref input.data)[r.json_start..r.json_end],r.value_captures);if(code!=0){return code;}i+=1;}return 0;}
pub fn Result<Table,u32> copy(ref Table input){var bytes::Buffer data=bytes::empty();var v::Vec<Row> rows=v::empty<Row>();let code=append(ref mut data,ref mut rows,input,0);if(code!=0){return .err(code);}return .ok(assemble(data,rows));}
pub fn usize helper_flag(){return 2097152;}
pub fn usize snapshot_flag(){return 3145728;}
