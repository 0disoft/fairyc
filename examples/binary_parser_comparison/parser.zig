const std = @import("std");

extern "c" fn fopen([*:0]const u8, [*:0]const u8) ?*anyopaque;
extern "c" fn fseek(*anyopaque, c_long, c_int) c_int;
extern "c" fn ftell(*anyopaque) c_long;
extern "c" fn fread([*]u8, usize, usize, *anyopaque) usize;
extern "c" fn fclose(*anyopaque) c_int;
extern "c" fn printf([*:0]const u8, ...) c_int;
extern fn parser_ticks() usize;
extern fn parser_frequency() usize;

fn decode(input: []const u8) error{ TruncatedHeader, TruncatedPayload }!usize {
    var offset: usize = 0;
    var sum: usize = 0;
    while (offset < input.len) {
        if (input.len - offset < 2) return error.TruncatedHeader;
        const length = @as(usize, input[offset]) + @as(usize, input[offset + 1]) * 256;
        offset += 2;
        if (input.len - offset < length) return error.TruncatedPayload;
        const end = offset + length;
        while (offset < end) : (offset += 1) {
            sum += input[offset];
        }
    }
    return sum;
}

fn load(allocator: std.mem.Allocator) ![]u8 {
    const file = fopen("input.bin", "rb") orelse return error.Input;
    defer _ = fclose(file);
    if (fseek(file, 0, 2) != 0) return error.Input;
    const size = ftell(file);
    if (size < 0 or size > 67108864 or fseek(file, 0, 0) != 0) return error.Input;
    const input = try allocator.alloc(u8, @intCast(size));
    errdefer allocator.free(input);
    if (fread(input.ptr, 1, input.len, file) != input.len) return error.Input;
    return input;
}

pub fn main() u8 {
    const allocator = std.heap.page_allocator;
    const buffer = load(allocator) catch return 4;
    defer allocator.free(buffer);
    const input: []const u8 = buffer;
    const start = parser_ticks();
    const result = decode(input);
    const end = parser_ticks();
    const frequency = parser_frequency();
    if (start == 0 or end < start or frequency == 0) return 5;
    const sum = result catch |err| return switch (err) {
        error.TruncatedHeader => 2,
        error.TruncatedPayload => 3,
    };
    _ = printf("%zu %zu %zu\n", sum, end - start, frequency);
    return 0;
}
