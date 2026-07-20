module examples::windows_file::main;

import examples::windows_file::windows_file as fs;

@effects(filesystem)
fn write_then_close(mut file: fs::File) -> i32 {
    let bytes: &[u8] = b"FairyFile";
    let operation_code: i32 = match fs::write_all(&mut file, bytes) {
        Result::Ok(written) => if written == 9 {
            match fs::sync(&mut file) {
                Result::Ok(_) => 0,
                Result::Err(_) => 12,
            }
        } else {
            11
        },
        Result::Err(_) => 10,
    };
    let close_code: i32 = match fs::close(file) {
        Result::Ok(_) => 0,
        Result::Err(_) => 13,
    };
    if operation_code != 0 {
        operation_code
    } else {
        close_code
    }
}

@effects(filesystem)
fn read_then_close(mut file: fs::File) -> i32 {
    var buffer: [u8; 16] = [
        0, 0, 0, 0, 0, 0, 0, 0,
        0, 0, 0, 0, 0, 0, 0, 0,
    ];
    let operation_code: i32 = match fs::read(&mut file, &mut buffer) {
        Result::Ok(read) => if read == 9 &&
            buffer[0] == 70 && buffer[1] == 97 && buffer[2] == 105 &&
            buffer[3] == 114 && buffer[4] == 121 && buffer[5] == 70 &&
            buffer[6] == 105 && buffer[7] == 108 && buffer[8] == 101 {
            match fs::read(&mut file, &mut buffer) {
                Result::Ok(eof) => if eof == 0 { 0 } else { 23 },
                Result::Err(_) => 22,
            }
        } else {
            21
        },
        Result::Err(_) => 20,
    };
    let close_code: i32 = match fs::close(file) {
        Result::Ok(_) => 0,
        Result::Err(_) => 24,
    };
    if operation_code != 0 {
        operation_code
    } else {
        close_code
    }
}

@effects(filesystem)
fn implicit_drop_probe(file: fs::File) -> i32 {
    0
}

@effects(filesystem)
fn main() -> i32 {
    let data_path: &CStr = c"fairyc-fy-file-data.tmp";
    let drop_path: &CStr = c"fairyc-fy-file-drop.tmp";
    let write_code: i32 = match fs::create_truncate(data_path) {
        Result::Ok(file) => write_then_close(file),
        Result::Err(_) => 30,
    };
    if write_code != 0 {
        write_code
    } else {
        let read_code: i32 = match fs::open_read(data_path) {
            Result::Ok(file) => read_then_close(file),
            Result::Err(_) => 31,
        };
        if read_code != 0 {
            read_code
        } else {
            let drop_code: i32 = match fs::create_truncate(drop_path) {
                Result::Ok(file) => implicit_drop_probe(file),
                Result::Err(_) => 32,
            };
            if drop_code != 0 {
                drop_code
            } else {
                match fs::create_truncate(drop_path) {
                    Result::Ok(file) => match fs::close(file) {
                        Result::Ok(_) => 42,
                        Result::Err(_) => 34,
                    },
                    Result::Err(_) => 33,
                }
            }
        }
    }
}
