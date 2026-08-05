module std::fs;

@repr(C)
struct raw_file_result {
    u32 status;
    u32 os_code;
    usize count;
}

@repr(C)
struct raw_file_position_result {
    u32 status;
    u32 os_code;
    usize position;
}

@repr(C)
struct raw_file_size_result {
    u32 status;
    u32 os_code;
    usize size;
}

@repr(C)
struct raw_open_options {
    u8 read;
    u8 write;
    u8 create;
    u8 truncate;
}

/// Reports a stable file status, the target OS code, and bytes completed before failure.
pub struct FileError {
    u32 status;
    u32 os_code;
    usize completed;
}

/// Owns one Windows file handle and closes it automatically at scope exit.
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
unsafe extern C fn raw_file_result file_open_raw(ptr u8 path, ptr raw_open_options options, ptr mut void output) ;

@effects(filesystem)
@link_name("fy_std_windows_file_read_v0")
unsafe extern C fn raw_file_result file_read_raw(ptr mut void file, ptr mut u8 buffer, usize length) ;

@effects(filesystem)
@link_name("fy_std_windows_file_position_v0")
unsafe extern C fn raw_file_position_result file_position_raw(ptr mut void file) ;

@effects(filesystem)
@link_name("fy_std_windows_file_size_v0")
unsafe extern C fn raw_file_size_result file_size_raw(ptr void file) ;

@effects(filesystem)
@link_name("fy_std_windows_file_seek_start_v0")
unsafe extern C fn raw_file_position_result file_seek_start_raw(ptr mut void file, usize offset) ;

@effects(filesystem)
@link_name("fy_std_windows_file_write_all_v0")
unsafe extern C fn raw_file_result file_write_all_raw(ptr mut void file, ptr u8 data, usize length) ;

@effects(filesystem)
@link_name("fy_std_windows_file_sync_v0")
unsafe extern C fn raw_file_result file_sync_raw(ptr mut void file) ;

@effects(filesystem)
@link_name("fy_std_windows_file_close_v0")
unsafe extern C fn raw_file_result file_close_raw(ptr mut void file) ;

@effects(filesystem)
@link_name("fy_std_windows_file_drop_v0")
unsafe extern C fn u32 file_drop_raw(ptr mut void file) ;

fn FileError file_error(raw_file_result result) {
    return FileError {
        .status = result.status,
        .os_code = result.os_code,
        .completed = result.count,
    };
}

fn FileError position_error(raw_file_position_result result) {
    return FileError {
        .status = result.status,
        .os_code = result.os_code,
        .completed = 0,
    };
}

fn FileError size_error(raw_file_size_result result) {
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
fn Result<File, FileError> open_with(ref CStr path, raw_open_options options) {
    var File file = File { .handle = null, .open = 0 };
    let raw_file_result result = unsafe {
        file_open_raw(
            path as ptr u8,
            ref options as ptr raw_open_options,
            ref mut file as ptr mut void
        )
    };
    return if (result.status == 0) {
        .ok(file)
    } else {
        let i32 discarded = discard_closed(file);
        .err(file_error(result))
    };
}

/// Creates a writable file, replacing its previous contents when it already exists.
@effects(filesystem)
pub fn Result<File, FileError> create_truncate(ref CStr path) {
    return open_with(path, raw_open_options {
        .read = 0,
        .write = 1,
        .create = 1,
        .truncate = 1,
    });
}

/// Opens an existing file for reading.
@effects(filesystem)
pub fn Result<File, FileError> open_read(ref CStr path) {
    return open_with(path, raw_open_options {
        .read = 1,
        .write = 0,
        .create = 0,
        .truncate = 0,
    });
}

/// Opens an existing file for reading and writing without truncating it.
@effects(filesystem)
pub fn Result<File, FileError> open_update(ref CStr path) {
    return open_with(path, raw_open_options {
        .read = 1,
        .write = 1,
        .create = 0,
        .truncate = 0,
    });
}

/// Writes every byte or reports how many bytes completed before failure.
@effects(filesystem)
pub fn Result<usize, FileError> write_all(ref mut File file, ref u8[] bytes) {
    let raw_file_result result = unsafe {
        file_write_all_raw(file as ptr mut void, bytes as ptr u8, bytes.len)
    };
    return if (result.status == 0) {
        .ok(result.count)
    } else {
        .err(file_error(result))
    };
}

/// Reads at most buffer.len bytes; a successful zero count is end-of-file.
@effects(filesystem)
pub fn Result<usize, FileError> read(ref mut File file, ref mut u8[] buffer) {
    let raw_file_result result = unsafe {
        file_read_raw(file as ptr mut void, buffer as ptr mut u8, buffer.len)
    };
    return if (result.status == 0) {
        .ok(result.count)
    } else {
        .err(file_error(result))
    };
}

/// Returns the current absolute byte position.
@effects(filesystem)
pub fn Result<usize, FileError> position(ref mut File file) {
    let raw_file_position_result result = unsafe {
        file_position_raw(file as ptr mut void)
    };
    return if (result.status == 0) {
        .ok(result.position)
    } else {
        .err(position_error(result))
    };
}

/// Returns the file length in bytes without changing its current position.
@effects(filesystem)
pub fn Result<usize, FileError> size(ref File file) {
    let raw_file_size_result result = unsafe {
        file_size_raw(file as ptr void)
    };
    return if (result.status == 0) {
        .ok(result.size)
    } else {
        .err(size_error(result))
    };
}

/// Moves to one absolute byte position from the start and returns that position.
@effects(filesystem)
pub fn Result<usize, FileError> seek_start(ref mut File file, usize offset) {
    let raw_file_position_result result = unsafe {
        file_seek_start_raw(file as ptr mut void, offset)
    };
    return if (result.status == 0) {
        .ok(result.position)
    } else {
        .err(position_error(result))
    };
}

/// Synchronizes buffered file contents with the operating system.
@effects(filesystem)
pub fn Result<i32, FileError> sync(ref mut File file) {
    let raw_file_result result = unsafe {
        file_sync_raw(file as ptr mut void)
    };
    return if (result.status == 0) {
        .ok(0)
    } else {
        .err(file_error(result))
    };
}

/// Consumes the file and attempts one close; success and failure both leave it closed.
@effects(filesystem)
pub fn Result<i32, FileError> close(File input) {
    var File file = input;
    let raw_file_result result = unsafe {
        file_close_raw(ref mut file as ptr mut void)
    };
    return if (result.status == 0) {
        .ok(0)
    } else {
        .err(file_error(result))
    };
}

/// Returns the stable file status carried by error.
pub fn u32 error_status(FileError error) {
    return error.status;
}

/// Returns the target operating-system status carried by error.
pub fn u32 error_os_code(FileError error) {
    return error.os_code;
}

/// Returns the bytes completed before error.
pub fn usize error_completed(FileError error) {
    return error.completed;
}
