module literals;

const DECIMAL: i32 = 1_000;
const HEXADECIMAL: u32 = 0xDEAD_BEEF;
const BINARY: u8 = 0b1010_0101;
const FRACTION: f64 = 1_024.5e-2;
const TEXT: &str = "line\n\u{03BB}";
const BYTES: &[u8] = b"\x00A";
const C_NAME: &CStr = c"fairy";
