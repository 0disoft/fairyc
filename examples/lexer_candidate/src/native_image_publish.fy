module native_image_publish;
import windows_rooted_file as rooted;
import windows_host_paths as host;
import std::bytes as bytes;

// Publication is relative to retained directory handles, including newly
// created parents. A path alias cannot redirect a staged image outside root.
fn bool local(ref u8[] path){
    if(path.len==0 || path.len>4095){return false;}var usize first=0;var usize i=0;
    while(i<=path.len){if(i==path.len || path[i]==47 || path[i]==92){if(!rooted::valid_leaf(path[first..i])){return false;}first=i+1;}i+=1;}return true;
}
@effects(filesystem) pub fn Result<bool,u32> within(ref u8[] root,ref u8[] path,ref u8[] image){
    if(!local(path) || image.len==0 || image.len>67108864){return .err(1);}
    var directory=switch(rooted::open(root)){case .ok(p):{p}case .err(_):{return .err(2);}};
    var usize first=0;var usize i=0;while(i<path.len){if(path[i]==47 || path[i]==92){
        let child=switch(rooted::directory(ref directory,path[first..i])){case .ok(p):{p}case .err(_):{return .err(2);}};directory=child;first=i+1;
    }i+=1;}
    let leaf=path[first..];switch(rooted::open_read(ref directory,leaf)){case .ok(file):{switch(rooted::close(file)){case .ok(_):{}case .err(_):{return .err(3);}};}case .err(e):{if(!rooted::missing(e)){return .err(3);}}};
    var usize slot=0;while(slot<4096){var bytes::Buffer name=bytes::empty();if(!bytes::append(ref mut name,b".fairyc-candidate-").is_ok || !bytes::append_usize(ref mut name,slot).is_ok || !bytes::append(ref mut name,b".tmp").is_ok){return .err(4);}
        var file=switch(rooted::create_new(ref directory,bytes::view(ref name))){case .ok(p):{p}case .err(e):{if(rooted::collision(e)){slot+=1;continue;}return .err(5);}};
        switch(rooted::write_all(ref mut file,image)){case .ok(_):{}case .err(_):{return .err(6);}};
        switch(rooted::sync(ref mut file)){case .ok(_):{}case .err(_):{return .err(7);}};
        switch(rooted::publish(ref directory,ref mut file,leaf)){case .ok(_):{}case .err(_):{return .err(8);}};
        return switch(rooted::close(file)){case .ok(_):{.ok(true)}case .err(_):{.err(9)}};
    }return .err(5);
}
@effects(filesystem,environment) pub fn Result<bool,u32> publish(ref u8[] root,ref u8[] path,ref u8[] image){
    if(root.len!=0){return within(root,path,image);}var usize last=0;var usize i=0;while(i<path.len){if(path[i]==47 || path[i]==92){last=i+1;}i+=1;}
    if(last>0){return within(path[..last],path[last..],image);}let cwd=switch(host::current()){case .ok(p):{p}case .err(_):{return .err(2);}};return within(bytes::view(ref cwd),path,image);
}
