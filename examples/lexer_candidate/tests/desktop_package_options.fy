module tests::desktop_package_options;
import desktop_package_options as options;
import std::bytes as bytes;
import std::vec as v;

fn void add(ref mut v::Vec<bytes::Buffer> input,ref u8[] value){let copied=switch(bytes::copy(value)){case .ok(p):{p}case .err(_):{assert(false);return;}};assert(v::push<bytes::Buffer>(input,copied).is_ok);}
fn options::Selection accepted(ref v::Vec<bytes::Buffer> input){return switch(options::parse(input)){case .ok(p):{p}case .err(_):{assert(false);return accepted(input);}};}
fn options::Problem refused(ref v::Vec<bytes::Buffer> input){return switch(options::parse(input)){case .ok(_):{assert(false);return refused(input);}case .err(p):{p}};}

@test fn void accepts_only_packaging_flags_in_either_order(){
    var v::Vec<bytes::Buffer> input=v::empty<bytes::Buffer>();let defaults=accepted(ref input);assert(!defaults.locked && !defaults.json);
    add(ref mut input,b"--json");add(ref mut input,b"--locked");let selected=accepted(ref input);assert(selected.locked && selected.json);
    var v::Vec<bytes::Buffer> reversed=v::empty<bytes::Buffer>();add(ref mut reversed,b"--locked");add(ref mut reversed,b"--json");let other=accepted(ref reversed);assert(other.locked && other.json);
}
@test fn void rejects_duplicate_and_build_only_flags(){
    var v::Vec<bytes::Buffer> input=v::empty<bytes::Buffer>();add(ref mut input,b"--locked");add(ref mut input,b"--json");add(ref mut input,b"--locked");let duplicate=refused(ref input);assert(duplicate.index==2 && duplicate.duplicate);
    var v::Vec<bytes::Buffer> other=v::empty<bytes::Buffer>();add(ref mut other,b"--release");let invalid=refused(ref other);assert(invalid.index==0 && !invalid.duplicate);
}
