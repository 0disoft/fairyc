module module_state_graph_model;
import std::bytes as bytes;
import std::vec as v;

// Private transaction graph transport. Numeric capabilities are identities,
// never execution authority. Only the independent verifier may publish a proof.
pub struct Value{usize id;u32 codec;}
pub struct Operand{usize input;u32 codec;usize start;usize end;}
// 1..6: cell read/write/delete, map read/write/delete. Helpers/borrows stay closed.
pub struct Operation{u32 kind;usize capability;usize field;Operand key;Operand value;usize result;usize found;usize cost;}
// 1 jump, 2 branch, 3 commit, 4 reject, 5 trap. Edges use ordered typed arguments.
pub struct Terminator{u32 kind;usize condition;usize target;usize argument_start;usize argument_count;usize alternative;usize alternate_start;usize alternate_count;usize category_start;usize category_end;usize cost;}
pub struct Block{usize id;usize parameter_start;usize parameter_count;usize operation_start;usize operation_count;Terminator term;}
pub struct Header{usize layout;usize capability;u32 effects;usize entry;}
pub resource Graph{Header header;bytes::Buffer data;v::Vec<Value> parameters;v::Vec<usize> arguments;v::Vec<Operation> operations;v::Vec<Block> blocks;cleanup{}}
pub fn Operand absent(){return Operand{.input=0,.codec=0,.start=0,.end=0};}
pub fn Operand input(usize id){return Operand{.input=id,.codec=0,.start=0,.end=0};}
pub fn Terminator exit(u32 kind){return Terminator{.kind,.condition=0,.target=0,.argument_start=0,.argument_count=0,.alternative=0,.alternate_start=0,.alternate_count=0,.category_start=0,.category_end=0,.cost=1};}
pub fn Graph assemble(Header header,bytes::Buffer data,v::Vec<Value> parameters,v::Vec<usize> arguments,v::Vec<Operation> operations,v::Vec<Block> blocks){return Graph{.header,.data,.parameters,.arguments,.operations,.blocks};}
pub fn Header header(ref Graph graph){return graph.header;}
pub fn ref u8[] data(ref Graph graph){return bytes::view(ref graph.data);}
pub fn ref v::Vec<Value> parameters(ref Graph graph){return ref graph.parameters;}
pub fn ref v::Vec<usize> arguments(ref Graph graph){return ref graph.arguments;}
pub fn ref v::Vec<Operation> operations(ref Graph graph){return ref graph.operations;}
pub fn ref v::Vec<Block> blocks(ref Graph graph){return ref graph.blocks;}
pub fn usize block_limit(){return 64;}
pub fn usize operation_limit(){return 1024;}
pub fn usize byte_limit(){return 1048576;}
pub fn bool range(usize start,usize count,usize limit){return start<=limit && count<=limit-start;}
