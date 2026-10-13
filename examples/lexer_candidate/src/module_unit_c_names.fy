module module_unit_c_names;
import module_identified_ir as identified;
import module_unit_body_ir_model as ir;
import module_unit_cleanup_mir_model as clean;
import module_unit_value_model as budget;
import candidate_limits as limits;
import c_text as w;
import std::bytes as bytes;
import std::vec as v;

struct Row{u32 kind;usize start;usize end;clean::Frame frame;}
// Detached private C names, derived only from checked source identity rows.
// Ordinary frames retain their numeric spelling. This is not a cache authority.
pub resource Table{bytes::Buffer text;v::Vec<Row> rows;usize work;cleanup{}}
pub fn Table empty(){return Table{.text=bytes::empty(),.rows=v::empty<Row>(),.work=0};}
pub fn usize work(ref Table p){return p.work;}
pub fn usize count(ref Table p){return v::len<Row>(ref p.rows);}
fn clean::Problem fail(u32 code,ref ir::Program p,usize frame){return clean::Problem{.code,.module_index=if(frame==0){0}else{clean::frame_at(ir::cleanup_ir(p),frame).module_index},.start=if(frame==0){0}else{ir::start(ir::at(p,frame))},.end=if(frame==0){0}else{ir::end(ir::at(p,frame))}};}
fn bool valid(ref u8[] id){if(id.len!=67 || !bytes::equal(id[..3],b"g0_")){return false;}var usize i=3;while(i<id.len){let n=id[i];if(!(n>=48 && n<=57 || n>=97 && n<=102)){return false;}i+=1;}return true;}
fn usize bucket(ref u8[] id,u32 kind,usize mask){var usize hash=kind as usize;var usize i=3;while(i<id.len){hash=(hash*33+(id[i] as usize))&mask;i+=1;}return hash;}
pub fn Result<Table,clean::Problem> build(ref identified::Program input,ref mut usize work){
    let program=identified::ir_program(input);let count=ir::count(program);if(count==0 || count>limits::project_frames() || identified::frame_count(input)!=count){return .err(fail(1001,program,0));}if(!budget::charge(work,0)){return .err(fail(1002,program,0));}let before=*work;var usize size=1;while(size<count*2){size*=2;}
    var v::Vec<usize> seen=v::empty<usize>();var usize i=0;while(i<size){if(!budget::charge(work,1)){return .err(fail(1002,program,0));}if(!v::push<usize>(ref mut seen,0).is_ok){return .err(fail(1003,program,0));}i+=1;}
    var bytes::Buffer text=bytes::empty();var v::Vec<Row> rows=v::empty<Row>();i=1;
    while(i<=count){if(!budget::charge(work,1)){return .err(fail(1002,program,i));}let f=clean::frame_at(ir::cleanup_ir(program),i);let identity=identified::frame_at(input,i);let id=identified::identity(input,identity);var usize start=0;var usize end=0;let u32 kind=if(identity.node==0){0}else if(f.kind==2){2}else{1};
        if(kind==0){if(id.len!=0){return .err(fail(1001,program,i));}}
        else{if(f.kind!=1 && f.kind!=2 || !valid(id)){return .err(fail(1001,program,i));}if(!budget::charge(work,id.len+1)){return .err(fail(1002,program,i));}var usize slot=bucket(id,kind,size-1);
            while(true){if(!budget::charge(work,1)){return .err(fail(1002,program,i));}let old=*v::at<usize>(ref seen,slot);if(old==0){let cell=v::at_mut<usize>(ref mut seen,slot);*cell=i;break;}let row=*v::at<Row>(ref rows,old-1);if(row.kind==kind){if(!budget::charge(work,id.len)){return .err(fail(1002,program,i));}if(bytes::equal(bytes::view(ref text)[row.start..row.end],id)){return .err(fail(1001,program,i));}}slot=(slot+1)&(size-1);}
            start=bytes::len(ref text);if(start>1048576-id.len){return .err(fail(1002,program,i));}if(!bytes::append(ref mut text,id).is_ok){return .err(fail(1003,program,i));}end=bytes::len(ref text);
        }
        if(!v::push<Row>(ref mut rows,Row{.kind,.start,.end,.frame=f}).is_ok){return .err(fail(1003,program,i));}i+=1;
    }return .ok(Table{.text,.rows,.work=*work-before});
}
pub fn bool matches(ref Table names,ref ir::Program program,ref mut usize work){if(count(names)==0){return true;}if(count(names)!=ir::count(program)){return false;}var usize i=1;while(i<=count(names)){if(!budget::charge(work,1)){return false;}let a=v::at<Row>(ref names.rows,i-1);let b=clean::frame_at(ir::cleanup_ir(program),i);if(a.frame.kind!=b.kind || a.frame.context!=b.context || a.frame.module_index!=b.module_index || a.frame.declaration!=b.declaration || a.frame.result!=b.result || a.frame.effects!=b.effects || a.frame.receiver!=b.receiver || a.frame.receiver_type!=b.receiver_type){return false;}i+=1;}return true;}
pub fn bool function(ref Table names,usize frame,ref mut bytes::Buffer out,ref mut w::State s){
    if(s.code!=0){return false;}if(count(names)==0){return w::function(out,s,frame);}if(frame==0 || frame>count(names)){s.code=1001;return false;}let row=*v::at<Row>(ref names.rows,frame-1);return if(row.kind==0){w::function(out,s,frame)}else{w::text(out,s,if(row.kind==2){b"fy_cleanup_"}else{b"fy_fn_"}) && w::text(out,s,bytes::view(ref names.text)[row.start..row.end])};
}
