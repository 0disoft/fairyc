module tests::project_check_options;
import project_check_options as cli;
import std::bytes as bytes;
import std::vec as v;

fn void add(ref mut v::Vec<bytes::Buffer> input,ref u8[] value){let copy=switch(bytes::copy(value)){case .ok(p):{p}case .err(_):{assert(false);return;}};assert(v::push<bytes::Buffer>(input,copy).is_ok);}
fn cli::Options parsed(ref v::Vec<bytes::Buffer> input){return switch(cli::parse(input)){case .ok(p):{p}case .err(_):{assert(false);parsed(input)}};}
fn u32 rejected(ref v::Vec<bytes::Buffer> input){return switch(cli::parse(input)){case .ok(_):{assert(false);0}case .err(e):{e.code}};}
@test fn void cache_selection_preserves_old_flags_and_owns_unicode_path(){
    var v::Vec<bytes::Buffer> input=v::empty<bytes::Buffer>();{let old=parsed(ref input);assert(!cli::locked(ref old) && !cli::status(ref old) && cli::cache_root(ref old).len==0);}
    add(ref mut input,b"--locked");add(ref mut input,b"--cache-root");add(ref mut input,b"C:/cache \xed\x95\x9c\xea\xb8\x80");add(ref mut input,b"--cache-status");let p=parsed(ref input);assert(cli::locked(ref p) && cli::status(ref p));assert(v::clear<bytes::Buffer>(ref mut input)==4);assert(bytes::equal(cli::cache_root(ref p),b"C:/cache \xed\x95\x9c\xea\xb8\x80"));
}
@test fn void malformed_duplicate_and_unbound_cache_flags_reject(){
    var v::Vec<bytes::Buffer> input=v::empty<bytes::Buffer>();add(ref mut input,b"--cache-root");assert(rejected(ref input)==2);add(ref mut input,b"--locked");assert(rejected(ref input)==2);
    discard v::clear<bytes::Buffer>(ref mut input);add(ref mut input,b"--cache-root");add(ref mut input,b"");assert(rejected(ref input)==2);
    discard v::clear<bytes::Buffer>(ref mut input);add(ref mut input,b"--cache-root");add(ref mut input,b"bad\0path");assert(rejected(ref input)==2);
    discard v::clear<bytes::Buffer>(ref mut input);add(ref mut input,b"--cache-status");assert(rejected(ref input)==5);add(ref mut input,b"--cache-status");assert(rejected(ref input)==4);
    discard v::clear<bytes::Buffer>(ref mut input);add(ref mut input,b"--cache-root");add(ref mut input,b"one");add(ref mut input,b"--cache-root");add(ref mut input,b"two");assert(rejected(ref input)==3);
    discard v::clear<bytes::Buffer>(ref mut input);add(ref mut input,b"--locked");add(ref mut input,b"--locked");assert(rejected(ref input)==1);
}
