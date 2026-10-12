module tests::native_cli_options;
import native_cli_options as cli;
import std::bytes as bytes;
import std::vec as v;
import std::process as process;

fn void add(ref mut v::Vec<bytes::Buffer> input,ref u8[] value){let copied=switch(bytes::copy(value)){case .ok(p):{p}case .err(_):{assert(false);return;}};assert(v::push<bytes::Buffer>(input,copied).is_ok);}
fn cli::Options parsed(bool run,ref v::Vec<bytes::Buffer> input){return switch(cli::parse(run,input)){case .ok(p):{p}case .err(_):{assert(false);return parsed(run,input);}};}
fn cli::Problem rejected(bool run,ref v::Vec<bytes::Buffer> input){return switch(cli::parse(run,input)){case .ok(_):{assert(false);return rejected(run,input);}case .err(p):{p}};}
@test fn void preserves_flags_and_all_arguments_after_the_first_delimiter(){
    var v::Vec<bytes::Buffer> input=v::empty<bytes::Buffer>();let empty=parsed(false,ref input);assert(!cli::locked(ref empty) && !cli::release(ref empty) && v::len<bytes::Buffer>(cli::arguments(ref empty))==0);
    add(ref mut input,b"--locked");add(ref mut input,b"--release");let build=parsed(false,ref input);assert(cli::locked(ref build) && cli::release(ref build) && v::len<bytes::Buffer>(cli::arguments(ref build))==0);
    add(ref mut input,b"--");add(ref mut input,b"--release");add(ref mut input,b"");add(ref mut input,b"--");add(ref mut input,b"\xed\x95\x9c\xea\xb8\x80\xf0\x9f\xa7\x9a");let run=parsed(true,ref input);let args=cli::arguments(ref run);assert(cli::locked(ref run) && cli::release(ref run) && v::len<bytes::Buffer>(args)==4 && bytes::equal(bytes::view(v::at<bytes::Buffer>(args,0)),b"--release") && bytes::len(v::at<bytes::Buffer>(args,1))==0 && bytes::equal(bytes::view(v::at<bytes::Buffer>(args,2)),b"--") && bytes::equal(bytes::view(v::at<bytes::Buffer>(args,3)),b"\xed\x95\x9c\xea\xb8\x80\xf0\x9f\xa7\x9a"));assert(v::len<bytes::Buffer>(ref input)==7);
    assert(v::clear<bytes::Buffer>(ref mut input)==7);add(ref mut input,b"--");let no_args=parsed(true,ref input);assert(v::len<bytes::Buffer>(cli::arguments(ref no_args))==0 && !cli::locked(ref no_args) && !cli::release(ref no_args));
}
@test fn void rejects_unknown_duplicates_and_nul_with_reference_precedence(){
    var v::Vec<bytes::Buffer> input=v::empty<bytes::Buffer>();add(ref mut input,b"--");let build=rejected(false,ref input);assert(build.kind==1 && build.index==0);
    assert(v::clear<bytes::Buffer>(ref mut input)==1);add(ref mut input,b"--locked");add(ref mut input,b"--locked");let locked=rejected(true,ref input);assert(locked.kind==2 && locked.flag==1 && locked.index==1);var bytes::Buffer text=bytes::empty();assert(cli::diagnostic(true,locked,ref mut text) && bytes::equal(bytes::view(ref text),b"fairyc: command \"run\" received --locked more than once\n"));
    assert(v::clear<bytes::Buffer>(ref mut input)==2);add(ref mut input,b"--release");add(ref mut input,b"--release");let release=rejected(false,ref input);assert(release.kind==2 && release.flag==2 && release.index==1);assert(bytes::resize_zeroed(ref mut text,0).is_ok && cli::diagnostic(false,release,ref mut text) && bytes::equal(bytes::view(ref text),b"fairyc: command \"build\" received --release more than once\n"));
    assert(v::clear<bytes::Buffer>(ref mut input)==2);add(ref mut input,b"--unknown");add(ref mut input,b"--");add(ref mut input,b"bad\0argument");let nul=rejected(true,ref input);assert(nul.kind==3 && nul.index==2 && cli::exit_code(nul)==2);assert(bytes::resize_zeroed(ref mut text,0).is_ok && cli::diagnostic(true,nul,ref mut text) && bytes::equal(bytes::view(ref text),b"fairyc: command \"run\" rejects program arguments containing a NUL byte\n"));
}
@test fn void grows_real_process_argument_storage_without_a_fixed_path_limit(){
    var u8[4096] data=[0;4096];let size=switch(process::arg(1,ref mut data)){case .ok(n):{n}case .err(_):{assert(false);return;}};assert(size>128);let actual=switch(cli::read(1)){case .ok(p):{p}case .err(_):{assert(false);return;}};let ref u8[] expected=ref data;assert(v::len<bytes::Buffer>(ref actual)==1 && bytes::equal(bytes::view(v::at<bytes::Buffer>(ref actual,0)),expected[..size]));let empty=switch(cli::read(process::arg_count())){case .ok(p):{p}case .err(_):{assert(false);return;}};assert(v::len<bytes::Buffer>(ref empty)==0 && !cli::read(process::arg_count()+1).is_ok);
}
