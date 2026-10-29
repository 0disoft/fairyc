module linux_cache_file;
import linux_rooted_file as rooted;
import linux_directory as directory;
import linux_retention as retention;
import linux_workspace_fs as workspace;
import std::bytes as bytes;
import std::vec as v;

// Private cache transport. Preserve Linux metadata without converting epochs or
// discarding device/nanosecond identity needed by conditional removal.
pub struct Metadata{rooted::Metadata raw;usize size;}
pub struct Problem{u32 stage;u32 os_code;usize completed;}
pub resource Entries{directory::Entries all;v::Vec<usize> indices;cleanup{}}
fn Problem error(rooted::Problem p){return Problem{.stage=p.stage,.os_code=p.os_code,.completed=p.completed};}
pub fn Metadata empty_metadata(){return Metadata{.raw=rooted::Metadata{.device=0,.identity=0,.size=0,.modified_seconds=0,.modified_nanos=0,.mode=0},.size=0};}
fn Metadata wrap(rooted::Metadata raw){return Metadata{.raw,.size=raw.size};}
pub fn bool regular(Metadata p){return rooted::regular(p.raw);}
pub fn bool earlier(Metadata a,Metadata b){return a.raw.modified_seconds<b.raw.modified_seconds || a.raw.modified_seconds==b.raw.modified_seconds && a.raw.modified_nanos<b.raw.modified_nanos;}
pub fn bool same_time(Metadata a,Metadata b){return a.raw.modified_seconds==b.raw.modified_seconds && a.raw.modified_nanos==b.raw.modified_nanos;}
// Anonymous O_TMPFILE stages never enter inventory, so named temporary aging is
// inapplicable. Foreign/Windows stage names are deliberately left untouched.
pub fn bool expired(Metadata p,usize age){return false;}
pub fn bool missing(Problem p){return (p.stage==3 || p.stage==4) && p.os_code==2;}
pub fn bool collision(Problem p){return p.stage==12 && p.os_code==17;}
pub fn bool publication_collision(Problem p){return collision(p);}
@effects(filesystem) pub fn Result<rooted::Directory,Problem> open(ref u8[] path){return switch(workspace::open_directory(path)){case .ok(root):{.ok(root)}case .err(e):{.err(Problem{.stage=3,.os_code=e.os_code,.completed=e.completed})}};}
@effects(filesystem) pub fn Result<rooted::File,Problem> open_read(ref rooted::Directory root,ref u8[] leaf){return switch(rooted::open_read(root,leaf)){case .ok(file):{.ok(file)}case .err(e):{.err(error(e))}};}
@effects(filesystem) pub fn Result<Metadata,Problem> observe(ref rooted::File file){return switch(rooted::observe(file)){case .ok(p):{.ok(wrap(p))}case .err(e):{.err(error(e))}};}
@effects(filesystem) pub fn Result<usize,Problem> read(ref mut rooted::File file,ref mut u8[] data){return switch(rooted::read(file,data)){case .ok(p):{.ok(p)}case .err(e):{.err(error(e))}};}
@effects(filesystem) pub fn Result<rooted::File,Problem> create_new(ref rooted::Directory root,ref u8[] unused_name){
    if(!rooted::valid_leaf(unused_name)){return .err(Problem{.stage=1,.os_code=0,.completed=0});}
    return switch(rooted::create(root)){case .ok(file):{.ok(file)}case .err(e):{.err(error(e))}};
}
@effects(filesystem) pub fn Result<usize,Problem> write_all(ref mut rooted::File file,ref u8[] data){return switch(rooted::write_all(file,data)){case .ok(p):{.ok(p)}case .err(e):{.err(error(e))}};}
@effects(filesystem) pub fn Result<bool,Problem> sync(ref mut rooted::File file){return switch(rooted::sync(file)){case .ok(p):{.ok(p)}case .err(e):{.err(error(e))}};}
@effects(filesystem) pub fn Result<bool,Problem> publish_new(ref rooted::Directory root,ref mut rooted::File file,ref u8[] leaf){return switch(rooted::publish_new(root,file,leaf)){case .ok(p):{.ok(p)}case .err(e):{.err(error(e))}};}
@effects(filesystem) pub fn Result<bool,Problem> close(rooted::File file){return switch(rooted::close(file)){case .ok(p):{.ok(p)}case .err(e):{.err(error(e))}};}
pub fn usize count(ref Entries p){return v::len<usize>(ref p.indices);}
pub fn ref u8[] name(ref Entries p,usize index){return directory::name(ref p.all,*v::at<usize>(ref p.indices,index));}
pub fn Metadata metadata(ref Entries p,usize index){return wrap(directory::metadata(ref p.all,*v::at<usize>(ref p.indices,index)));}
@effects(filesystem) pub fn Result<Entries,Problem> scan(ref rooted::Directory root,usize maximum,ref mut usize work){
    let all=switch(directory::scan(root,maximum,work)){case .ok(p):{p}case .err(e):{return .err(Problem{.stage=12,.os_code=e.os_code,.completed=e.completed});}};
    var v::Vec<usize> indices=v::empty<usize>();var usize i=0;
    while(i<directory::count(ref all)){
        let leaf=directory::name(ref all,i);let cost=leaf.len+2;
        if(*work>directory::work_limit() || cost>directory::work_limit()-*work){return .err(Problem{.stage=14,.os_code=0,.completed=i});};*work+=cost;
        if(retention::valid_generation(leaf) && !v::push<usize>(ref mut indices,i).is_ok){return .err(Problem{.stage=2,.os_code=0,.completed=i});}i+=1;
    }return .ok(Entries{.all,.indices});
}
@effects(filesystem) pub fn Result<bool,Problem> erase_if(ref rooted::Directory root,ref u8[] leaf,Metadata expected){return switch(retention::erase_if(root,leaf,expected.raw)){case .ok(p):{.ok(p)}case .err(e):{.err(Problem{.stage=if(e.kind==4){7}else{10},.os_code=if(e.os_code!=0){e.os_code}else{e.close_code},.completed=if(e.removed){1}else{0}})}};}
