module module_unit_value_model;
import module_types as types;
import module_unit_body_ast as ast;
import std::vec as v;

pub struct Frame {u32 kind;usize context;usize module_index;usize declaration;usize result;}
// state 0 unresolved value; 1 known value type; 2 non-value syntax/callee.
pub struct Field {usize id;usize owner;usize module_index;usize declaration;usize source_index;}
// Place kind 0 is a value/provenance only; 1 local, 2 field, 3 dereference, 4 index.
pub struct Place {u32 kind;usize root;usize scope;usize projection;bool mutable;bool indirect;bool resource_field;bool borrowable;}
pub fn Place no_place(){return Place{.kind=0,.root=0,.scope=0,.projection=0,.mutable=false,.indirect=false,.resource_field=false,.borrowable=false};}
pub struct Builtin {u32 kind;usize element;bool slice;bool unsafe_context;}
pub struct Expression {usize source_index;u32 kind;usize start;usize end;usize left;usize right;usize target;Field field;Place place;Builtin builtin;usize expected;usize type;u32 state;bool flexible;bool minimum;}
pub struct Atoms {usize[8] scalar;usize text;usize bytes;usize cstr;}
pub struct Seed {usize result;usize instance;Builtin builtin;}
pub fn Seed seed(usize result,usize instance){return Seed{.result,.instance,.builtin=Builtin{.kind=0,.element=0,.slice=false,.unsafe_context=false}};}
pub struct Invocation {u32 kind;usize expression;usize source_index;usize target_module;usize target_declaration;usize instance;usize argument_start;usize argument_count;usize start;usize end;bool checked;}
pub struct Argument {usize invocation;usize position;usize expression;usize source_index;usize expected;usize actual;usize start;usize end;usize label_start;usize label_end;bool checked;bool shared;}
pub struct Scan {Problem problem;bool changed;}
pub struct Local {usize header_start;usize header_end;u32 kind;usize source_index;usize initializer;usize expected;usize type;usize scope;Place origin;usize start;usize end;}
pub struct Consumer {u32 kind;usize source_index;usize expression;usize target;usize operator;u32 header;usize expected;usize actual;bool checked;usize start;usize end;}
pub struct Return {usize expression;usize expected;usize actual;bool checked;usize start;usize end;}
pub struct Problem {u32 code;usize module_index;usize start;usize end;usize expected;usize actual;}
pub resource Values {bool checked_completion;Frame frame;v::Vec<Expression> expressions;v::Vec<Local> locals;v::Vec<Return> returns;v::Vec<Invocation> calls;v::Vec<Argument> arguments;v::Vec<Consumer> consumers;usize unresolved;usize inferred;usize checked_returns;usize work;cleanup{}}
pub fn Values assemble(Frame frame,v::Vec<Expression> expressions,v::Vec<Local> locals,v::Vec<Return> returns,v::Vec<Invocation> calls,v::Vec<Argument> arguments,v::Vec<Consumer> consumers,usize unresolved,usize inferred,usize checked_returns,usize work,bool checked_completion){return Values{.checked_completion,.frame,.expressions,.locals,.returns,.calls,.arguments,.consumers,.unresolved,.inferred,.checked_returns,.work};}
pub fn Frame frame(ref Values p){return p.frame;}
pub fn usize work(ref Values p){return p.work;}
pub fn usize unresolved(ref Values p){return p.unresolved;}
pub fn usize inferred(ref Values p){return p.inferred;}
pub fn usize checked_returns(ref Values p){return p.checked_returns;}
pub fn Problem fail(Frame f,u32 code,usize start,usize end,usize expected,usize actual){return Problem{.code,.module_index=f.module_index,.start,.end,.expected,.actual};}
pub fn bool charge(ref mut usize work,usize amount){return ast::charge(work,amount);}
pub fn usize index(ref v::Vec<Expression> expressions,usize source_index){if(source_index==0){return 0;}var usize low=0;var usize high=v::len<Expression>(expressions);while(low<high){let usize mid=low+(high-low)/2;if(v::at<Expression>(expressions,mid).source_index<source_index){low=mid+1;}else{high=mid;}}return if(low<v::len<Expression>(expressions) && v::at<Expression>(expressions,low).source_index==source_index){low+1}else{0};}
pub fn usize find(ref v::Vec<types::Type> graph,u32 kind,u32 scalar){var usize i=1;while(i<=types::count(graph)){let t=types::at(graph,i);if(t.kind==kind && t.scalar==scalar && t.module_index==0 && t.declaration==0 && t.first==0 && t.second==0){return i;}i+=1;}return 0;}
pub fn Atoms atoms(ref v::Vec<types::Type> graph){var Atoms p=Atoms{.scalar=[0;8],.text=0,.bytes=0,.cstr=0};var usize i=1;while(i<=types::count(graph)){let t=types::at(graph,i);if(t.kind==1 && t.scalar<=7){let usize slot=t.scalar as usize;p.scalar[slot]=i;}else if(t.kind==9){p.text=i;}else if(t.kind==10){p.bytes=i;}else if(t.kind==14){p.cstr=i;}i+=1;}return p;}
pub fn usize atom(ref Atoms p,u32 scalar){return p.scalar[scalar as usize];}
pub fn usize local_count(ref Values p){return v::len<Local>(ref p.locals);}
pub fn Local local_at(ref Values p,usize id){return *v::at<Local>(ref p.locals,id-1);}
pub fn usize expression_count(ref Values p){return v::len<Expression>(ref p.expressions);}
pub fn Expression expression_at(ref Values p,usize id){return *v::at<Expression>(ref p.expressions,id-1);}
pub fn usize return_count(ref Values p){return v::len<Return>(ref p.returns);}
pub fn Return return_at(ref Values p,usize index){return *v::at<Return>(ref p.returns,index);}
pub fn usize call_count(ref Values p){return v::len<Invocation>(ref p.calls);}
pub fn Invocation call_at(ref Values p,usize index){return *v::at<Invocation>(ref p.calls,index);}
pub fn usize argument_count(ref Values p){return v::len<Argument>(ref p.arguments);}
pub fn Argument argument_at(ref Values p,usize index){return *v::at<Argument>(ref p.arguments,index);}
pub fn usize checked_calls(ref Values p){var usize count=0;var usize i=0;while(i<call_count(p)){if(call_at(p,i).checked){count+=1;}i+=1;}return count;}
pub fn usize checked_arguments(ref Values p){var usize count=0;var usize i=0;while(i<argument_count(p)){if(argument_at(p,i).checked){count+=1;}i+=1;}return count;}
pub fn usize shared_arguments(ref Values p){var usize count=0;var usize i=0;while(i<argument_count(p)){if(argument_at(p,i).shared){count+=1;}i+=1;}return count;}
pub fn usize cast_count(ref Values p){var usize count=0;var usize i=1;while(i<=expression_count(p)){if(expression_at(p,i).kind==23){count+=1;}i+=1;}return count;}
pub fn usize checked_casts(ref Values p){var usize count=0;var usize i=1;while(i<=expression_count(p)){let e=expression_at(p,i);if(e.kind==23 && e.state==1 && e.type==e.target){count+=1;}i+=1;}return count;}
pub fn usize field_count(ref Values p){var usize count=0;var usize i=1;while(i<=expression_count(p)){if(expression_at(p,i).kind==8){count+=1;}i+=1;}return count;}
pub fn usize checked_fields(ref Values p){var usize count=0;var usize i=1;while(i<=expression_count(p)){let e=expression_at(p,i);if(e.kind==8 && e.state==1 && e.field.owner!=0){count+=1;}i+=1;}return count;}

pub fn usize record_count(ref Values p){var usize count=0;var usize i=1;while(i<=expression_count(p)){if(expression_at(p,i).kind==10){count+=1;}i+=1;}return count;}
pub fn usize checked_records(ref Values p){var usize count=0;var usize i=1;while(i<=expression_count(p)){let e=expression_at(p,i);if(e.kind==10 && e.state==1 && e.target!=0 && e.type==e.target){count+=1;}i+=1;}return count;}
pub fn usize record_fields(ref Values p){var usize count=0;var usize i=1;while(i<=expression_count(p)){let e=expression_at(p,i);if(e.kind==11 && e.field.id!=0){count+=1;}i+=1;}return count;}

pub fn usize consumer_count(ref Values p){return v::len<Consumer>(ref p.consumers);}
pub fn Consumer consumer_at(ref Values p,usize id){return *v::at<Consumer>(ref p.consumers,id);}
pub fn usize checked_consumers(ref Values p){var usize count=0;var usize i=0;while(i<consumer_count(p)){if(consumer_at(p,i).checked){count+=1;}i+=1;}return count;}

pub fn usize assignment_count(ref Values p){var usize n=0;var usize i=0;while(i<consumer_count(p)){if(consumer_at(p,i).kind==4){n+=1;}i+=1;}return n;}
pub fn usize checked_assignments(ref Values p){var usize n=0;var usize i=0;while(i<consumer_count(p)){let c=consumer_at(p,i);if(c.kind==4 && c.checked){n+=1;}i+=1;}return n;}

pub fn bool checked_completion(ref Values p){return p.checked_completion;}
