module module_unit_loan_live_model;
import module_unit_value_model as m;
import std::vec as v;

pub struct Use {usize start;usize end;}
pub struct Local {usize start;usize end;bool borrowed;}
pub struct Row {usize local;usize start;usize end;}
pub struct Header {usize start;usize count;}
pub struct Point {usize source_index;usize start;usize end;usize map;bool checked;}
pub struct Loop {usize break_live;usize continue_live;}
pub struct Context {m::Frame frame;usize work;usize rounds;}
pub resource Plan {v::Vec<Local> locals;v::Vec<Point> expressions;v::Vec<Point> statements;v::Vec<Header> maps;v::Vec<Row> rows;usize start;usize end;usize work;usize rounds;cleanup{}}
pub fn Plan assemble(v::Vec<Local> locals,v::Vec<Point> expressions,v::Vec<Point> statements,v::Vec<Header> maps,v::Vec<Row> rows,usize start,usize end,usize work,usize rounds){return Plan{.locals,.expressions,.statements,.maps,.rows,.start,.end,.work,.rounds};}
pub fn usize start(ref Plan p){return p.start;}
pub fn usize end(ref Plan p){return p.end;}
pub fn usize work(ref Plan p){return p.work;}
pub fn usize rounds(ref Plan p){return p.rounds;}
pub fn usize local_count(ref Plan p){return v::len<Local>(ref p.locals);}
pub fn Local local_at(ref Plan p,usize local){return *v::at<Local>(ref p.locals,local-1);}
pub fn usize expression_count(ref Plan p){return v::len<Point>(ref p.expressions);}
pub fn Point expression_at(ref Plan p,usize index){return *v::at<Point>(ref p.expressions,index);}
pub fn usize statement_count(ref Plan p){return v::len<Point>(ref p.statements);}
pub fn Point statement_at(ref Plan p,usize index){return *v::at<Point>(ref p.statements,index);}
pub fn usize use_count(ref Plan p,Point point){return if(point.map==0){0}else{v::at<Header>(ref p.maps,point.map-1).count};}
pub fn Row use_at(ref Plan p,Point point,usize index){return *v::at<Row>(ref p.rows,v::at<Header>(ref p.maps,point.map-1).start+index);}
pub fn m::Problem fail(ref Context c,u32 code,usize start,usize end){return m::fail(c.frame,code,start,end,0,0);}
pub fn Result<v::Vec<Use>,m::Problem> empty(usize count,ref mut Context c){var v::Vec<Use> result=v::empty<Use>();var usize i=0;while(i<count){if(!m::charge(ref mut c.work,1)){return .err(fail(c,10,0,0));}if(!v::push<Use>(ref mut result,Use{.start=0,.end=0}).is_ok){return .err(fail(c,11,0,0));}i+=1;}return .ok(result);}
pub fn Result<v::Vec<Use>,m::Problem> copy(ref v::Vec<Use> source,ref mut Context c){var v::Vec<Use> result=v::empty<Use>();var usize i=0;while(i<v::len<Use>(source)){if(!m::charge(ref mut c.work,1)){return .err(fail(c,10,0,0));}if(!v::push<Use>(ref mut result,*v::at<Use>(source,i)).is_ok){return .err(fail(c,11,0,0));}i+=1;}return .ok(result);}
pub fn m::Problem merge(ref mut v::Vec<Use> target,ref v::Vec<Use> source,ref mut Context c){if(v::len<Use>(target)!=v::len<Use>(source)){return fail(c,12,0,0);}var usize i=0;while(i<v::len<Use>(source)){if(!m::charge(ref mut c.work,1)){return fail(c,10,0,0);}let next=*v::at<Use>(source,i);if(next.end>v::at<Use>(target,i).end){let slot=v::at_mut<Use>(target,i);*slot=next;}i+=1;}return fail(c,0,0,0);}
pub fn m::Problem replace(ref mut v::Vec<Use> target,ref v::Vec<Use> source,ref mut Context c){if(v::len<Use>(target)!=v::len<Use>(source)){return fail(c,12,0,0);}var usize i=0;while(i<v::len<Use>(source)){if(!m::charge(ref mut c.work,1)){return fail(c,10,0,0);}let slot=v::at_mut<Use>(target,i);*slot=*v::at<Use>(source,i);i+=1;}return fail(c,0,0,0);}
pub fn Result<bool,m::Problem> same(ref v::Vec<Use> a,ref v::Vec<Use> b,ref mut Context c){if(v::len<Use>(a)!=v::len<Use>(b)){return .err(fail(c,12,0,0));}var usize i=0;while(i<v::len<Use>(a)){if(!m::charge(ref mut c.work,1)){return .err(fail(c,10,0,0));}let x=*v::at<Use>(a,i);let y=*v::at<Use>(b,i);if(x.start!=y.start || x.end!=y.end){return .ok(false);}i+=1;}return .ok(true);}
pub fn m::Problem kill(ref mut v::Vec<Use> env,usize local,ref v::Vec<Local> locals,ref mut Context c){if(local==0){return fail(c,0,0,0);}if(local>v::len<Local>(locals) || local>v::len<Use>(env)){return fail(c,12,0,0);}if(!m::charge(ref mut c.work,1)){return fail(c,10,0,0);}if(v::at<Local>(locals,local-1).borrowed){let slot=v::at_mut<Use>(env,local-1);*slot=Use{.start=0,.end=0};}return fail(c,0,0,0);}
