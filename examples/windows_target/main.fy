module examples::windows_target::main;

@link_name("fairyc_unused_windows_long")
unsafe extern C fn unused_windows_long(value: c::long) -> c::long;

@link_name("fy_std_text_utf8_valid_v0")
unsafe extern C fn utf8_valid(data: *u8, length: usize) -> u32;

@repr(C)
struct system_heap_block {
    data: *mut void;
    size: usize;
    alignment: usize;
}

@link_name("fy_std_system_heap_allocate_v0")
unsafe extern C fn allocate(
    size: usize,
    alignment: usize,
    output: *mut system_heap_block
) -> u32;

@link_name("fy_std_system_heap_deallocate_v0")
unsafe extern C fn deallocate(block: *mut system_heap_block) -> u32;

fn main() -> i32 {
    var block: system_heap_block = system_heap_block {
        data: null,
        size: 0,
        alignment: 0,
    };
    let allocated: u32 = unsafe {
        allocate(8, 8, &mut block as *mut system_heap_block)
    };
    let allocation_valid: bool = block.size == 8 && block.alignment == 8;
    let deallocated: u32 = unsafe {
        deallocate(&mut block as *mut system_heap_block)
    };
    let deallocation_valid: bool = block.size == 0 && block.alignment == 0;
    if allocated == 0 && allocation_valid && deallocated == 0 && deallocation_valid &&
        unsafe { utf8_valid(null, 0) } == 1 {
        42
    } else {
        1
    }
}
