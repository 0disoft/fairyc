module examples::windows_file::windows_file;

@repr(C)
struct fy_std_windows_file_result_v0 {
    u32 status;
    u32 os_code;
    usize count;
}

@repr(C)
struct fy_std_windows_file_position_result_v0 {
    u32 status;
    u32 os_code;
    usize position;
}

@repr(C)
struct fy_std_windows_open_options_v0 {
    u8 read;
    u8 write;
    u8 create;
    u8 truncate;
}

pub struct FileError {
    u32 status;
    u32 os_code;
    usize completed;
}

@effects(filesystem)
pub resource File {
    ptr mut void handle;
    u8 open;

    cleanup {
        let u32 dropped = unsafe {
            file_drop_raw(ref mut this as ptr mut void)
        };
    }
}

@effects(filesystem)
@link_name("fy_std_windows_file_open_v0")
unsafe extern C fn fy_std_windows_file_result_v0 file_open_raw(ptr u8 path, ptr fy_std_windows_open_options_v0 options, ptr mut void output) ;

@effects(filesystem)
@link_name("fy_std_windows_file_read_v0")
unsafe extern C fn fy_std_windows_file_result_v0 file_read_raw(ptr mut void file, ptr mut u8 buffer, usize length) ;

@effects(filesystem)
@link_name("fy_std_windows_file_position_v0")
unsafe extern C fn fy_std_windows_file_position_result_v0 file_position_raw(ptr mut void file) ;

@effects(filesystem)
@link_name("fy_std_windows_file_seek_start_v0")
unsafe extern C fn fy_std_windows_file_position_result_v0 file_seek_start_raw(ptr mut void file, usize offset) ;

@effects(filesystem)
@link_name("fy_std_windows_file_write_all_v0")
unsafe extern C fn fy_std_windows_file_result_v0 file_write_all_raw(ptr mut void file, ptr u8 data, usize length) ;

@effects(filesystem)
@link_name("fy_std_windows_file_sync_v0")
unsafe extern C fn fy_std_windows_file_result_v0 file_sync_raw(ptr mut void file) ;

@effects(filesystem)
@link_name("fy_std_windows_file_close_v0")
unsafe extern C fn fy_std_windows_file_result_v0 file_close_raw(ptr mut void file) ;

@effects(filesystem)
@link_name("fy_std_windows_file_drop_v0")
unsafe extern C fn u32 file_drop_raw(ptr mut void file) ;

fn FileError to_error(fy_std_windows_file_result_v0 result) {
    return FileError {
        .status = result.status,
        .os_code = result.os_code,
        .completed = result.count,
    };
}

fn FileError position_error(fy_std_windows_file_position_result_v0 result) {
    return FileError {
        .status = result.status,
        .os_code = result.os_code,
        .completed = 0,
    };
}

@effects(filesystem)
fn i32 discard_closed(File file) {
    return 0;
}

@effects(filesystem)
fn Result<File, FileError> open_with(ref CStr path, fy_std_windows_open_options_v0 options) {
    var File file = File { .handle = null, .open = 0 };
    let fy_std_windows_file_result_v0 result = unsafe {
        file_open_raw(
            path as ptr u8,
            ref options as ptr fy_std_windows_open_options_v0,
            ref mut file as ptr mut void
        )
    };
    return if (result.status == 0) {
        .ok(file)
    } else {
        let i32 discarded = discard_closed(file);
        .err(to_error(result))
    };
}

@effects(filesystem)
pub fn Result<File, FileError> create_truncate(ref CStr path) {
    return open_with(path, fy_std_windows_open_options_v0 {
        .read = 0,
        .write = 1,
        .create = 1,
        .truncate = 1,
    });
}

@effects(filesystem)
pub fn Result<File, FileError> open_read(ref CStr path) {
    return open_with(path, fy_std_windows_open_options_v0 {
        .read = 1,
        .write = 0,
        .create = 0,
        .truncate = 0,
    });
}

@effects(filesystem)
pub fn Result<usize, FileError> write_all(ref mut File file, ref u8[] bytes) {
    let fy_std_windows_file_result_v0 result = unsafe {
        file_write_all_raw(file as ptr mut void, bytes as ptr u8, bytes.len)
    };
    return if (result.status == 0) {
        .ok(result.count)
    } else {
        .err(to_error(result))
    };
}

@effects(filesystem)
pub fn Result<usize, FileError> read(ref mut File file, ref mut u8[] buffer) {
    let fy_std_windows_file_result_v0 result = unsafe {
        file_read_raw(file as ptr mut void, buffer as ptr mut u8, buffer.len)
    };
    return if (result.status == 0) {
        .ok(result.count)
    } else {
        .err(to_error(result))
    };
}

@effects(filesystem)
pub fn Result<usize, FileError> position(ref mut File file) {
    let fy_std_windows_file_position_result_v0 result = unsafe {
        file_position_raw(file as ptr mut void)
    };
    return if (result.status == 0) {
        .ok(result.position)
    } else {
        .err(position_error(result))
    };
}

@effects(filesystem)
pub fn Result<usize, FileError> seek_start(ref mut File file, usize offset) {
    let fy_std_windows_file_position_result_v0 result = unsafe {
        file_seek_start_raw(file as ptr mut void, offset)
    };
    return if (result.status == 0) {
        .ok(result.position)
    } else {
        .err(position_error(result))
    };
}

@effects(filesystem)
pub fn Result<i32, FileError> sync(ref mut File file) {
    let fy_std_windows_file_result_v0 result = unsafe {
        file_sync_raw(file as ptr mut void)
    };
    return if (result.status == 0) {
        .ok(0)
    } else {
        .err(to_error(result))
    };
}

@effects(filesystem)
pub fn Result<i32, FileError> close(File input) {
    var File file = input;
    let fy_std_windows_file_result_v0 result = unsafe {
        file_close_raw(ref mut file as ptr mut void)
    };
    return if (result.status == 0) {
        .ok(0)
    } else {
        .err(to_error(result))
    };
}
