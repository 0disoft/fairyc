module main;

unsafe extern C fn i32 fairy_win32_run();

fn i32 main() {
    return unsafe { fairy_win32_run() };
}
