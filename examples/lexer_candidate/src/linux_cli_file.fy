module linux_cli_file;
import linux_host_paths as paths;
import linux_rooted_file as rooted;
import candidate_limits as limits;
import std::bytes as bytes;

// Same input/allocation/limit codes as file_source, with rooted Linux opens.
@effects(filesystem)
pub fn Result<bytes::Buffer,u32> load(ref u8[] path) {
    if(!paths::decode(path).is_ok) { return .err(1); }
    var usize leaf=0; var usize at=0;
    while(at<path.len) { if(path[at]==47) { leaf=at+1; } at+=1; }
    if(!rooted::valid_leaf(path[leaf..])) { return .err(1); }
    let parent=if(leaf==0){b"."}else if(leaf==1){b"/"}else{path[..leaf-1]};
    let root=switch(rooted::open(parent)) { case .ok(p):{p} case .err(_):{return .err(1);} };
    var file=switch(rooted::open_read(ref root,path[leaf..])) { case .ok(p):{p} case .err(_):{return .err(1);} };
    let before=switch(rooted::observe(ref file)) { case .ok(p):{p} case .err(_):{return .err(1);} };
    let maximum=limits::source_bytes();
    if(before.size>maximum) { return .err(4); }
    var bytes::Buffer input=bytes::empty();
    if(!bytes::resize_zeroed(ref mut input,maximum+1).is_ok) { return .err(3); }
    var usize used=0;
    while(true) {
        let remaining=maximum+1-used;
        let count=switch(rooted::read(ref mut file,bytes::view_mut(ref mut input)[used..used+if(remaining>65536){65536}else{remaining}])) {
            case .ok(p):{p} case .err(_):{return .err(1);}
        };
        used+=count; if(used>maximum) { return .err(4); } if(count==0) { break; }
    }
    let after=switch(rooted::observe(ref file)) { case .ok(p):{p} case .err(_):{return .err(1);} };
    if(before.device!=after.device || before.identity!=after.identity || before.size!=after.size || used!=after.size ||
       before.modified_seconds!=after.modified_seconds || before.modified_nanos!=after.modified_nanos) { return .err(1); }
    if(!rooted::close(file).is_ok) { return .err(1); }
    if(!rooted::close_directory(root).is_ok) { return .err(1); }
    if(!bytes::resize_zeroed(ref mut input,used).is_ok) { return .err(3); }
    return .ok(input);
}
