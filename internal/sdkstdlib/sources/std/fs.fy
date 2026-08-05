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
struct raw_file_kind_result {
    u32 status;
    u32 os_code;
    u32 kind;
}

@repr(C)
struct raw_file_id {
    u32 volume_low;
    u32 volume_high;
    u32 file_low_0;
    u32 file_low_1;
    u32 file_high_0;
    u32 file_high_1;
}

@repr(C)
struct raw_open_options {
    u8 read;
    u8 write;
    u8 create;
    u8 truncate;
    u8 exclusive;
    u8 append;
}

/// Reports a stable file status, the target OS code, and bytes completed before failure.
pub struct FileError {
    u32 status;
    u32 os_code;
    usize completed;
}

/// Classifies the object behind one open file handle without exposing target constants.
pub struct FileKind {
    u32 value;
}

/// Identifies one observed file on its Windows volume for bounded equality checks.
pub struct FileId {
    u32 volume_low;
    u32 volume_high;
    u32 file_low_0;
    u32 file_low_1;
    u32 file_high_0;
    u32 file_high_1;
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
@link_name("fy_std_windows_file_read_exact_v0")
unsafe extern C fn raw_file_result file_read_exact_raw(ptr mut void file, ptr mut u8 buffer, usize length) ;

@effects(filesystem)
@link_name("fy_std_windows_file_position_v0")
unsafe extern C fn raw_file_position_result file_position_raw(ptr mut void file) ;

@effects(filesystem)
@link_name("fy_std_windows_file_size_v0")
unsafe extern C fn raw_file_size_result file_size_raw(ptr void file) ;

@effects(filesystem)
@link_name("fy_std_windows_file_kind_v0")
unsafe extern C fn raw_file_kind_result file_kind_raw(ptr void file) ;

@effects(filesystem)
@link_name("fy_std_windows_file_id_v0")
unsafe extern C fn raw_file_result file_id_raw(ptr void file, ptr mut raw_file_id output) ;

@effects(filesystem)
@link_name("fy_std_windows_file_set_len_v0")
unsafe extern C fn raw_file_size_result file_set_len_raw(ptr mut void file, usize size) ;

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

fn FileError kind_error(raw_file_kind_result result) {
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
        .exclusive = 0,
        .append = 0,
    });
}

/// Creates a new writable file and fails without changing an existing file.
@effects(filesystem)
pub fn Result<File, FileError> create_new(ref CStr path) {
    return open_with(path, raw_open_options {
        .read = 0,
        .write = 1,
        .create = 1,
        .truncate = 0,
        .exclusive = 1,
        .append = 0,
    });
}

/// Opens a file for reading and writing, creating it when missing without truncating it.
@effects(filesystem)
pub fn Result<File, FileError> open_or_create(ref CStr path) {
    return open_with(path, raw_open_options {
        .read = 1,
        .write = 1,
        .create = 1,
        .truncate = 0,
        .exclusive = 0,
        .append = 0,
    });
}

/// Opens a write-only append file, creating it when missing.
/// Every underlying write starts at the then-current end of file.
@effects(filesystem)
pub fn Result<File, FileError> open_append(ref CStr path) {
    return open_with(path, raw_open_options {
        .read = 0,
        .write = 1,
        .create = 1,
        .truncate = 0,
        .exclusive = 0,
        .append = 1,
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
        .exclusive = 0,
        .append = 0,
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
        .exclusive = 0,
        .append = 0,
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

/// Fills buffer or reports an incomplete prefix as status 7.
@effects(filesystem)
pub fn Result<usize, FileError> read_exact(ref mut File file, ref mut u8[] buffer) {
    let raw_file_result result = unsafe {
        file_read_exact_raw(file as ptr mut void, buffer as ptr mut u8, buffer.len)
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

/// Returns the target-neutral kind of the object behind this open handle.
@effects(filesystem)
pub fn Result<FileKind, FileError> kind(ref File file) {
    let raw_file_kind_result result = unsafe {
        file_kind_raw(file as ptr void)
    };
    return if (result.status != 0) {
        .err(kind_error(result))
    } else {
        let u32 value = if (result.kind == 1 || result.kind == 2 || result.kind == 3) {
            result.kind
        } else {
            0
        };
        .ok(FileKind { .value = value })
    };
}

/// Reports whether kind names an ordinary disk file.
pub fn bool kind_is_disk(FileKind kind) {
    return kind.value == 1;
}

/// Reports whether kind names a character device such as a console.
pub fn bool kind_is_character(FileKind kind) {
    return kind.value == 2;
}

/// Reports whether kind names a socket, named pipe, or anonymous pipe.
pub fn bool kind_is_pipe(FileKind kind) {
    return kind.value == 3;
}

/// Reports whether the target could not classify kind more specifically.
pub fn bool kind_is_unknown(FileKind kind) {
    return kind.value == 0;
}

/// Observes the opened file's volume and 128-bit target identifier.
/// The value is for bounded equality checks, not persistence or authorization.
@effects(filesystem)
pub fn Result<FileId, FileError> file_id(ref File file) {
    var raw_file_id value = raw_file_id {
        .volume_low = 0,
        .volume_high = 0,
        .file_low_0 = 0,
        .file_low_1 = 0,
        .file_high_0 = 0,
        .file_high_1 = 0,
    };
    let raw_file_result result = unsafe {
        file_id_raw(file as ptr void, ref mut value as ptr mut raw_file_id)
    };
    return if (result.status != 0) {
        .err(file_error(result))
    } else {
        .ok(FileId {
            .volume_low = value.volume_low,
            .volume_high = value.volume_high,
            .file_low_0 = value.file_low_0,
            .file_low_1 = value.file_low_1,
            .file_high_0 = value.file_high_0,
            .file_high_1 = value.file_high_1,
        })
    };
}

/// Reports whether two bounded observations identify the same file on one computer.
pub fn bool file_id_equal(ref FileId left, ref FileId right) {
    return left.volume_low == right.volume_low &&
        left.volume_high == right.volume_high &&
        left.file_low_0 == right.file_low_0 &&
        left.file_low_1 == right.file_low_1 &&
        left.file_high_0 == right.file_high_0 &&
        left.file_high_1 == right.file_high_1;
}

/// Changes the file length without changing its current position.
/// Bytes added by extending a file have unspecified contents until written.
@effects(filesystem)
pub fn Result<usize, FileError> set_len(ref mut File file, usize length) {
    let raw_file_size_result result = unsafe {
        file_set_len_raw(file as ptr mut void, length)
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
