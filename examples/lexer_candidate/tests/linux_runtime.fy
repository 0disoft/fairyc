module tests::linux_runtime;
import linux_runtime as selection;
import std::bytes as bytes;

fn void expected(ref u8[] header,u32 components,u32 hosts,bool vectors,bool arguments){
    let plan=switch(selection::select(header)){case .ok(p):{p}case .err(_):{assert(false);return;}};
    assert(plan.components==components && plan.hosts==hosts && plan.vectors==vectors && plan.arguments==arguments);
}
@test fn void rejects_wrong_target_unknown_symbols_and_windows_dependencies(){
    assert(!selection::select(b"fy_std_buffer_drop_v0();").is_ok);
    let ref u8[] header=b"/* fairyc.candidate-target/x86_64-linux-gnu/v1 */\nfy_std_unknown_v0();";
    switch(selection::select(header)){case .ok(_):{assert(false);}case .err(e):{assert(e.kind==2 && bytes::equal(header[e.start..e.end],b"fy_std_unknown_v0"));}};
    assert(!selection::select(b"/* fairyc.candidate-target/x86_64-linux-gnu/v1 */\nfy_std_windows_file_open_v0();").is_ok);
    assert(!selection::select(b"/* fairyc.candidate-target/x86_64-linux-gnu/v1 */\nfy_private_vector_unknown_v0();").is_ok);
    assert(!selection::select(b"/* fairyc.candidate-target/x86_64-linux-gnu/v1 */\nfairyc_private_linux_unknown_v1();").is_ok);
    assert(!selection::select(b"/* fairyc.candidate-target/x86_64-linux-gnu/v1 */\nfairyc_private_windows_open_v1();").is_ok);
    expected(b"/* fairyc.candidate-target/x86_64-linux-gnu/v1 */\nnot_fy_std_unknown_v0(); fy_std_unknown_v0;",0,0,false,false);
    expected(b"/* fairyc.candidate-target/x86_64-linux-gnu/v1 */\nfy_std_buffer_drop_v0\x0b();",0,0,false,false);
    expected(b"/* fairyc.candidate-target/x86_64-linux-gnu/v1 */\nfy_std_buffer_drop_v0\x0c();",20,0,false,false);
}
