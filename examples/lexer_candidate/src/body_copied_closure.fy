module body_copied_closure;
import bodies as bodies;
import body_state as state;
import scanner as scanner;
import types as types;
import std::vec as v;

// 34 is a written closure, left block / right header. Header 35 retains its
// result type, first logical entry ID, and capture/parameter counts (base 129).
pub fn usize parse(ref scanner::Stream stream,ref mut state::Cursor cursor,ref mut v::Vec<state::Node> nodes,ref mut v::Vec<bodies::Entry> entries,ref mut v::Vec<state::Statement> statements,ref mut v::Vec<types::Node> types){
    let token=cursor.index;let start=state::current(stream,cursor).start;if(!state::expect(stream,cursor,b"fn",1) || !state::expect(stream,cursor,b"[",11)){return 0;}let first=cursor.entries+1;var usize captures=0;var usize parameters=0;
    while(!state::word(stream,cursor.index,b"]")){if(captures>=128){cursor.status=4;return 0;}if(!bodies::entry(stream,cursor,nodes,entries,statements,types,1)){return 0;}{let last=v::len<bodies::Entry>(entries)-1;let row=v::at_mut<bodies::Entry>(entries,last);row.kind=104;}captures+=1;if(!state::take(stream,cursor,b",")){break;}}
    if(!state::expect(stream,cursor,b"]",11)){return 0;}let result=bodies::read_type(stream,cursor,types);if(result==0 || !state::expect(stream,cursor,b"(",4)){return 0;}
    while(!state::word(stream,cursor.index,b")")){if(parameters>=128){cursor.status=4;return 0;}if(!bodies::entry(stream,cursor,nodes,entries,statements,types,1)){return 0;}{let last=v::len<bodies::Entry>(entries)-1;let row=v::at_mut<bodies::Entry>(entries,last);row.kind=105;}parameters+=1;if(!state::take(stream,cursor,b",")){break;}}
    if(!state::expect(stream,cursor,b")",5)){return 0;}let depth=cursor.block_depth;let body=bodies::block(stream,cursor,nodes,entries,statements,types,depth,true);if(body==0){return 0;}
    let header=state::append(state::node_value(35,start,cursor.previous_end,result,first,captures*129+parameters),nodes,cursor);if(header==0){return 0;}return state::append(state::node_value(34,start,cursor.previous_end,token,body,header),nodes,cursor);
}
