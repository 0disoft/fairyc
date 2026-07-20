module examples::windows_file::windows_file;

@repr(C)
struct fy_std_windows_file_result_v0 {
    status: u32;
    os_code: u32;
    count: usize;
}

@repr(C)
struct fy_std_windows_open_options_v0 {
    read: u8;
    write: u8;
    create: u8;
    truncate: u8;
}

pub struct FileError {
    status: u32;
    os_code: u32;
    completed: usize;
}

@effects(filesystem)
pub resource File {
    handle: *mut void;
    open: u8;

    drop fn(self: &mut Self) {
        let dropped: u32 = unsafe {
            file_drop_raw(&mut self as *mut void)
        };
    }
}

@effects(filesystem)
@link_name("fy_std_windows_file_open_v0")
unsafe extern C fn file_open_raw(
    path: *u8,
    options: *fy_std_windows_open_options_v0,
    output: *mut void
) -> fy_std_windows_file_result_v0;

@effects(filesystem)
@link_name("fy_std_windows_file_read_v0")
unsafe extern C fn file_read_raw(
    file: *mut void,
    buffer: *mut void,
    length: usize
) -> fy_std_windows_file_result_v0;

@effects(filesystem)
@link_name("fy_std_windows_file_write_all_v0")
unsafe extern C fn file_write_all_raw(
    file: *mut void,
    data: *u8,
    length: usize
) -> fy_std_windows_file_result_v0;

@effects(filesystem)
@link_name("fy_std_windows_file_sync_v0")
unsafe extern C fn file_sync_raw(file: *mut void) -> fy_std_windows_file_result_v0;

@effects(filesystem)
@link_name("fy_std_windows_file_close_v0")
unsafe extern C fn file_close_raw(file: *mut void) -> fy_std_windows_file_result_v0;

@effects(filesystem)
@link_name("fy_std_windows_file_drop_v0")
unsafe extern C fn file_drop_raw(file: *mut void) -> u32;

fn to_error(result: fy_std_windows_file_result_v0) -> FileError {
    FileError {
        status: result.status,
        os_code: result.os_code,
        completed: result.count,
    }
}

@effects(filesystem)
fn discard_closed(file: File) -> i32 {
    0
}

@effects(filesystem)
fn open_with(
    path: &CStr,
    options: fy_std_windows_open_options_v0
) -> Result<File, FileError> {
    var file: File = File { handle: null, open: 0 };
    let result: fy_std_windows_file_result_v0 = unsafe {
        file_open_raw(
            path as *u8,
            &options as *fy_std_windows_open_options_v0,
            &mut file as *mut void
        )
    };
    if result.status == 0 {
        Result::Ok(file)
    } else {
        let discarded: i32 = discard_closed(file);
        Result::Err(to_error(result))
    }
}

@effects(filesystem)
pub fn create_truncate(path: &CStr) -> Result<File, FileError> {
    open_with(path, fy_std_windows_open_options_v0 {
        read: 0,
        write: 1,
        create: 1,
        truncate: 1,
    })
}

@effects(filesystem)
pub fn open_read(path: &CStr) -> Result<File, FileError> {
    open_with(path, fy_std_windows_open_options_v0 {
        read: 1,
        write: 0,
        create: 0,
        truncate: 0,
    })
}

@effects(filesystem)
pub fn write_all(file: &mut File, bytes: &[u8]) -> Result<usize, FileError> {
    let result: fy_std_windows_file_result_v0 = unsafe {
        file_write_all_raw(file as *mut void, bytes as *u8, bytes.len)
    };
    if result.status == 0 {
        Result::Ok(result.count)
    } else {
        Result::Err(to_error(result))
    }
}

@effects(filesystem)
pub fn read_16(file: &mut File, buffer: &mut [u8; 16]) -> Result<usize, FileError> {
    let result: fy_std_windows_file_result_v0 = unsafe {
        file_read_raw(file as *mut void, buffer as *mut void, 16)
    };
    if result.status == 0 {
        Result::Ok(result.count)
    } else {
        Result::Err(to_error(result))
    }
}

@effects(filesystem)
pub fn sync(file: &mut File) -> Result<i32, FileError> {
    let result: fy_std_windows_file_result_v0 = unsafe {
        file_sync_raw(file as *mut void)
    };
    if result.status == 0 {
        Result::Ok(0)
    } else {
        Result::Err(to_error(result))
    }
}

@effects(filesystem)
pub fn close(mut file: File) -> Result<i32, FileError> {
    let result: fy_std_windows_file_result_v0 = unsafe {
        file_close_raw(&mut file as *mut void)
    };
    if result.status == 0 {
        Result::Ok(0)
    } else {
        Result::Err(to_error(result))
    }
}
