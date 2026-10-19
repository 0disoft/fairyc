module tests::project_compiler_identity;
import project_cache_environment as compiler;
import project_compiler_image_hash as image;
import std::bytes as bytes;
import std::io as io;

// The maintained runner independently hashes the complete executed image and
// verifies the exact stdout again with an intentionally false argv[0].
@test @effects(filesystem,environment,console)
fn void reports_actual_running_image_and_rejects_invalid_work(){var usize work=16777217;assert(switch(compiler::compiler_current(ref mut work)){case .ok(_):{false}case .err(e):{e.stage==5 && e.completed==0}});assert(work==16777217);work=0;
    let actual=switch(compiler::compiler_current(ref mut work)){case .ok(p):{p}case .err(e):{discard io::write_stderr_u32(e.stage);assert(false);return;}};assert(image::size(ref actual)>0 && work>0 && work<=16777216 && bytes::equal(image::value(ref actual)[..image::prefix().len],image::prefix()));assert(io::write_bytes(image::value(ref actual)).is_ok && io::write("\n").is_ok);
}
