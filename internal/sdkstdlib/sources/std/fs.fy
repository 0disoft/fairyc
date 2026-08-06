module std::fs;

import std::bytes as bytes;

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
struct raw_file_attribute_result {
    u32 status;
    u32 os_code;
    u32 enabled;
}

@repr(C)
struct raw_file_buffer_result {
    u32 file_status;
    u32 os_code;
    u32 allocation_status;
    usize count;
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

/// Reports the independent file and allocation statuses from an owned-buffer read.
pub struct ReadAppendError {
    u32 file_status;
    u32 os_code;
    u32 allocation_status;
}

/// Classifies the object behind one open file handle without exposing target constants.
pub enum FileKind {
    Unknown;
    Disk;
    Character;
    Pipe;
}

/// Classifies one observed filesystem path without exposing target attributes.
pub enum PathKind {
    File;
    Directory;
    Other;
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
@link_name("fy_std_windows_file_remove_v0")
unsafe extern C fn raw_file_result file_remove_raw(ptr u8 path) ;

@effects(filesystem)
@link_name("fy_std_windows_directory_create_v0")
unsafe extern C fn raw_file_result directory_create_raw(ptr u8 path) ;

@effects(filesystem)
@link_name("fy_std_windows_directory_ensure_v0")
unsafe extern C fn raw_file_result directory_create_all_raw(ptr u8 path) ;

@effects(filesystem)
@link_name("fy_std_windows_directory_remove_v0")
unsafe extern C fn raw_file_result directory_remove_raw(ptr u8 path) ;

@effects(filesystem)
@link_name("fy_std_windows_directory_rename_v0")
unsafe extern C fn raw_file_result directory_rename_raw(ptr u8 source, ptr u8 destination) ;

@effects(filesystem)
@link_name("fy_std_windows_file_rename_v0")
unsafe extern C fn raw_file_result file_rename_raw(ptr u8 source, ptr u8 destination) ;

@effects(filesystem)
@link_name("fy_std_windows_file_copy_bytes_v0")
unsafe extern C fn raw_file_result file_copy_bytes_raw(ptr u8 source, ptr u8 destination) ;

@effects(filesystem)
@link_name("fy_std_windows_file_read_v0")
unsafe extern C fn raw_file_result file_read_raw(ptr mut void file, ptr mut u8 buffer, usize length) ;

@effects(filesystem)
@link_name("fy_std_windows_file_read_exact_v0")
unsafe extern C fn raw_file_result file_read_exact_raw(ptr mut void file, ptr mut u8 buffer, usize length) ;

@effects(filesystem)
@link_name("fy_std_windows_file_read_append_v0")
unsafe extern C fn raw_file_buffer_result file_read_append_raw(ptr mut void file, ptr mut void buffer, usize maximum) ;

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
@link_name("fy_std_windows_file_read_only_v0")
unsafe extern C fn raw_file_attribute_result file_read_only_raw(ptr void file) ;

@effects(filesystem)
@link_name("fy_std_windows_file_path_kind_v0")
unsafe extern C fn raw_file_kind_result file_path_kind_raw(ptr u8 path) ;

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
@link_name("fy_std_windows_file_seek_current_v0")
unsafe extern C fn raw_file_position_result file_seek_current_raw(ptr mut void file, i32 offset) ;

@effects(filesystem)
@link_name("fy_std_windows_file_seek_end_v0")
unsafe extern C fn raw_file_position_result file_seek_end_raw(ptr mut void file, i32 offset) ;

@effects(filesystem)
@link_name("fy_std_windows_file_write_v0")
unsafe extern C fn raw_file_result file_write_raw(ptr mut void file, ptr u8 data, usize length) ;

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

fn FileError attribute_error(raw_file_attribute_result result) {
    return FileError {
        .status = result.status,
        .os_code = result.os_code,
        .completed = 0,
    };
}

fn ReadAppendError read_append_error(raw_file_buffer_result result) {
    return ReadAppendError {
        .file_status = result.file_status,
        .os_code = result.os_code,
        .allocation_status = result.allocation_status,
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

/// Removes one existing file path without closing or changing any open File owner.
@effects(filesystem)
pub fn Result<i32, FileError> remove_file(ref CStr path) {
    let raw_file_result result = unsafe {
        file_remove_raw(path as ptr u8)
    };
    return if (result.status == 0) {
        .ok(0)
    } else {
        .err(file_error(result))
    };
}

/// Creates exactly one new directory without creating missing parents.
@effects(filesystem)
pub fn Result<i32, FileError> create_dir(ref CStr path) {
    let raw_file_result result = unsafe {
        directory_create_raw(path as ptr u8)
    };
    return if (result.status == 0) {
        .ok(0)
    } else {
        .err(file_error(result))
    };
}

/// Creates every missing directory component and accepts an existing directory.
/// A later failure preserves any prefix directories created by this call.
@effects(filesystem)
pub fn Result<i32, FileError> create_dir_all(ref CStr path) {
    let raw_file_result result = unsafe {
        directory_create_all_raw(path as ptr u8)
    };
    return if (result.status == 0) {
        .ok(0)
    } else {
        .err(file_error(result))
    };
}

/// Removes exactly one existing empty directory without recursion.
@effects(filesystem)
pub fn Result<i32, FileError> remove_dir(ref CStr path) {
    let raw_file_result result = unsafe {
        directory_remove_raw(path as ptr u8)
    };
    return if (result.status == 0) {
        .ok(0)
    } else {
        .err(file_error(result))
    };
}

/// Renames one existing directory without replacing the destination or copying across volumes.
@effects(filesystem)
pub fn Result<i32, FileError> rename_dir(ref CStr source, ref CStr destination) {
    let raw_file_result result = unsafe {
        directory_rename_raw(source as ptr u8, destination as ptr u8)
    };
    return if (result.status == 0) {
        .ok(0)
    } else {
        .err(file_error(result))
    };
}

/// Renames one existing file without replacing the destination or copying across volumes.
@effects(filesystem)
pub fn Result<i32, FileError> rename_file(ref CStr source, ref CStr destination) {
    let raw_file_result result = unsafe {
        file_rename_raw(source as ptr u8, destination as ptr u8)
    };
    return if (result.status == 0) {
        .ok(0)
    } else {
        .err(file_error(result))
    };
}

/// Copies one existing file's bytes into a newly created destination without replacing it.
/// A later failure preserves the completed destination prefix reported by FileError.
@effects(filesystem)
pub fn Result<usize, FileError> copy_file_bytes(ref CStr source, ref CStr destination) {
    let raw_file_result result = unsafe {
        file_copy_bytes_raw(source as ptr u8, destination as ptr u8)
    };
    return if (result.status == 0) {
        .ok(result.count)
    } else {
        .err(file_error(result))
    };
}

/// Performs at most one underlying write and returns its exact completed byte count.
@effects(filesystem)
pub fn Result<usize, FileError> write(ref mut File file, ref u8[] bytes) {
    let raw_file_result result = unsafe {
        file_write_raw(file as ptr mut void, bytes as ptr u8, bytes.len)
    };
    return if (result.status == 0) {
        .ok(result.count)
    } else {
        .err(file_error(result))
    };
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

/// Reads at most maximum bytes once and appends only the initialized result to buffer.
/// End-of-file succeeds with zero and leaves the initialized prefix unchanged.
@effects(filesystem)
pub fn Result<usize, ReadAppendError> read_append(
    ref mut File file,
    ref mut bytes::Buffer buffer,
    usize maximum
) {
    let raw_file_buffer_result result = unsafe {
        file_read_append_raw(file as ptr mut void, buffer as ptr mut void, maximum)
    };
    return if (result.file_status == 0 && result.allocation_status == 0) {
        .ok(result.count)
    } else {
        .err(read_append_error(result))
    };
}

/// Reads from the current position through end-of-file and appends every initialized byte.
/// A failure preserves the prefix appended by earlier completed reads.
@effects(filesystem)
pub fn Result<usize, ReadAppendError> read_to_end(
    ref mut File file,
    ref mut bytes::Buffer buffer
) {
    var usize total = 0;
    var usize maximum = 65536;
    while (true) {
        let usize count = read_append(file, buffer, maximum)?;
        total += count;
        if (count == 0) {
            return .ok(total);
        }
        if (count < maximum) {
            maximum -= count;
        } else {
            maximum = 65536;
        }
    }
    return .ok(total);
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
    } else if (result.kind == 1) {
        .ok(.Disk)
    } else if (result.kind == 2) {
        .ok(.Character)
    } else if (result.kind == 3) {
        .ok(.Pipe)
    } else {
        .ok(.Unknown)
    };
}

/// Reports whether the object behind this open handle currently has read-only metadata.
@effects(filesystem)
pub fn Result<bool, FileError> is_read_only(ref File file) {
    let raw_file_attribute_result result = unsafe {
        file_read_only_raw(file as ptr void)
    };
    return if (result.status == 0) {
        .ok(result.enabled != 0)
    } else {
        .err(attribute_error(result))
    };
}

/// Observes whether one bounded path currently names a file, directory, or other object.
/// This snapshot is not a persistent identity, authorization decision, or race-free precheck.
@effects(filesystem)
pub fn Result<PathKind, FileError> path_kind(ref CStr path) {
    let raw_file_kind_result result = unsafe {
        file_path_kind_raw(path as ptr u8)
    };
    return if (result.status != 0) {
        .err(kind_error(result))
    } else if (result.kind == 1) {
        .ok(.File)
    } else if (result.kind == 2) {
        .ok(.Directory)
    } else {
        .ok(.Other)
    };
}

/// Reports whether kind names an ordinary disk file.
pub fn bool kind_is_disk(FileKind kind) {
    return switch (kind) {
        case .Unknown : { false }
        case .Disk : { true }
        case .Character : { false }
        case .Pipe : { false }
    };
}

/// Reports whether kind names a character device such as a console.
pub fn bool kind_is_character(FileKind kind) {
    return switch (kind) {
        case .Unknown : { false }
        case .Disk : { false }
        case .Character : { true }
        case .Pipe : { false }
    };
}

/// Reports whether kind names a socket, named pipe, or anonymous pipe.
pub fn bool kind_is_pipe(FileKind kind) {
    return switch (kind) {
        case .Unknown : { false }
        case .Disk : { false }
        case .Character : { false }
        case .Pipe : { true }
    };
}

/// Reports whether the target could not classify kind more specifically.
pub fn bool kind_is_unknown(FileKind kind) {
    return switch (kind) {
        case .Unknown : { true }
        case .Disk : { false }
        case .Character : { false }
        case .Pipe : { false }
    };
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

/// Moves by one signed byte delta from the current position and returns the absolute position.
@effects(filesystem)
pub fn Result<usize, FileError> seek_current(ref mut File file, i32 offset) {
    let raw_file_position_result result = unsafe {
        file_seek_current_raw(file as ptr mut void, offset)
    };
    return if (result.status == 0) {
        .ok(result.position)
    } else {
        .err(position_error(result))
    };
}

/// Moves by one signed byte delta from the current end of file and returns the absolute position.
/// Moving beyond the end changes only the cursor until a later write or length change.
@effects(filesystem)
pub fn Result<usize, FileError> seek_end(ref mut File file, i32 offset) {
    let raw_file_position_result result = unsafe {
        file_seek_end_raw(file as ptr mut void, offset)
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

/// Returns the stable file status from an owned-buffer read failure, or zero otherwise.
pub fn u32 read_append_error_file_status(ReadAppendError error) {
    return error.file_status;
}

/// Returns the target operating-system status from an owned-buffer read failure.
pub fn u32 read_append_error_os_code(ReadAppendError error) {
    return error.os_code;
}

/// Returns the stable allocation status from an owned-buffer read failure, or zero otherwise.
pub fn u32 read_append_error_allocation_status(ReadAppendError error) {
    return error.allocation_status;
}
