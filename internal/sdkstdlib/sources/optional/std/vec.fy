module std::vec;

/// Owns a contiguous initialized prefix of moved elements.
pub resource Vec<T> {
    ptr mut void data;
    usize bytes;
    usize alignment;
    usize length;
    usize element_size;
    usize element_alignment;

    cleanup {
        while (this.length > 0) {
            let T value = take_slot<T>(unsafe { pop_slot(ref mut this as ptr mut void, this.element_size) });
        }
        unsafe { release(ref mut this as ptr mut void); };
    }
}

@link_name("fy_private_vector_initialize_v0")
unsafe extern C fn u32 initialize_raw(ptr mut void storage, usize size, usize alignment);
@link_name("fy_private_vector_release_empty_v0")
unsafe extern C fn u32 release_raw(ptr mut void storage);
@link_name("fy_private_vector_reserve_v0")
unsafe extern C fn u32 reserve_raw(ptr mut void storage, usize capacity);
@link_name("fy_private_vector_prepare_push_v0")
unsafe extern C fn u32 prepare_raw(ptr mut void storage, usize size, ptr mut void output);
@link_name("fy_private_vector_commit_push_v0")
unsafe extern C fn u32 commit_raw(ptr mut void storage, ptr mut void slot, usize size);
@link_name("fy_private_vector_pop_slot_v0")
unsafe extern C fn ptr mut void pop_slot_raw(ptr mut void storage, usize size);
// Keep raw checks shared instead of cloning them for every element type.
fn void initialize(ptr mut void storage, usize size, usize alignment) {
    assert(unsafe { initialize_raw(storage, size, alignment) } == 0);
}
fn void commit(ptr mut void storage, ptr mut void slot, usize size) {
    assert(unsafe { commit_raw(storage, slot, size) } == 0);
}
fn void release(ptr mut void storage) {
    assert(unsafe { release_raw(storage) } == 0);
}
fn ptr mut void nonnull(ptr mut void slot) {
    assert(slot != null);
    return slot;
}
fn ptr mut void pop_slot(ptr mut void storage, usize size) {
    return nonnull(unsafe { pop_slot_raw(storage, size) });
}
fn T take_slot<T>(ptr mut void slot) { return unsafe { storage_take<T>(slot) }; }
fn i32 write_slot<T>(ptr mut void slot, T value) { return unsafe { storage_put<T>(slot, value); 1 }; }

/// Constructs an empty vector without allocating.
pub fn Vec<T> empty<T>() {
    var value = Vec<T> { .data = null, .bytes = 0, .alignment = 0, .length = 0, .element_size = 0, .element_alignment = 0 };
    unsafe { initialize(ref mut value as ptr mut void, storage_size<T>(), storage_alignment<T>()); };
    return value;
}

/// Returns the number of initialized elements.
pub fn usize len<T>(ref Vec<T> value) { return value.length; }

/// Borrows the initialized byte prefix without copying or allocating.
/// The view and derived aliases keep the vector borrowed until their last use.
pub fn ref u8[] view_u8(ref Vec<u8> value) {
    return unsafe { storage_borrow<u8[]>(value, value.data, value.length) };
}

/// Borrows the initialized byte prefix exclusively without copying or allocating.
/// Structural changes and other owner access wait until the last retained use.
pub fn ref mut u8[] view_mut_u8(ref mut Vec<u8> value) {
    return unsafe { storage_borrow_mut<u8[]>(value, value.data, value.length) };
}

/// Borrows the initialized i64 prefix without copying or allocating.
/// Indices and slice bounds count elements. Mutation and move are forbidden
/// until the last use of this view or any derived subview.
pub fn ref i64[] view_i64(ref Vec<i64> value) {
    return unsafe { storage_borrow<i64[]>(value, value.data, value.length) };
}

/// Borrows the initialized i64 prefix exclusively without allocating or copying.
/// Structural changes and other owner access wait until the last retained use.
pub fn ref mut i64[] view_mut_i64(ref mut Vec<i64> value) {
    return unsafe { storage_borrow_mut<i64[]>(value, value.data, value.length) };
}

/// Borrows the initialized i32 prefix without copying or allocating.
/// Derived views keep the vector borrowed through their last use.
pub fn ref i32[] view_i32(ref Vec<i32> value) {
    return unsafe { storage_borrow<i32[]>(value, value.data, value.length) };
}

/// Borrows the initialized i32 prefix exclusively; lengths count elements.
/// Structural changes and other owner access wait for the last retained use.
pub fn ref mut i32[] view_mut_i32(ref mut Vec<i32> value) {
    return unsafe { storage_borrow_mut<i32[]>(value, value.data, value.length) };
}

/// Borrows the initialized u32 prefix without copying or allocating.
/// Derived views keep the vector borrowed through their last use.
pub fn ref u32[] view_u32(ref Vec<u32> value) {
    return unsafe { storage_borrow<u32[]>(value, value.data, value.length) };
}

/// Borrows the initialized u32 prefix exclusively; lengths count elements.
/// Structural changes and other owner access wait for the last retained use.
pub fn ref mut u32[] view_mut_u32(ref mut Vec<u32> value) {
    return unsafe { storage_borrow_mut<u32[]>(value, value.data, value.length) };
}

/// Borrows the initialized usize prefix without copying or allocating.
/// Derived views keep the vector borrowed through their last use.
pub fn ref usize[] view_usize(ref Vec<usize> value) {
    return unsafe { storage_borrow<usize[]>(value, value.data, value.length) };
}

/// Borrows the initialized usize prefix exclusively; lengths count elements.
/// Structural changes and other owner access wait for the last retained use.
pub fn ref mut usize[] view_mut_usize(ref mut Vec<usize> value) {
    return unsafe { storage_borrow_mut<usize[]>(value, value.data, value.length) };
}

/// Borrows the initialized prefix of an integer or scalar-field struct vector without copying.
/// The exact nominal type is preserved; structural mutation waits for last use.
pub fn ref T[] view<T>(ref Vec<T> value) {
    return unsafe { storage_borrow<T[]>(value, value.data, value.length) };
}

/// Borrows the initialized prefix of an integer or plain struct vector exclusively.
/// Whole-element replacement is allowed; owner mutation waits for the last use.
pub fn ref mut T[] view_mut<T>(ref mut Vec<T> value) {
    return unsafe { storage_borrow_mut<T[]>(value, value.data, value.length) };
}

/// Borrows an initialized element without moving it. An out-of-range index traps.
/// The vector cannot be mutated or moved while the returned reference is live.
pub fn ref T at<T>(ref Vec<T> value, usize index) {
    assert(index < value.length);
    return unsafe { storage_borrow<T>(value, nonnull(storage_element_address<T>(value.data, index))) };
}

/// Exclusively borrows an initialized element without moving it. An out-of-range index traps.
/// The vector cannot be borrowed again, mutated or moved while the reference is live.
pub fn ref mut T at_mut<T>(ref mut Vec<T> value, usize index) {
    assert(index < value.length);
    return unsafe { storage_borrow_mut<T>(value, nonnull(storage_element_address<T>(value.data, index))) };
}

/// Returns how many elements fit without growing the allocation.
pub fn usize capacity<T>(ref Vec<T> value) { return value.bytes / value.element_size; }

/// Reports whether no elements are initialized.
pub fn bool is_empty<T>(ref Vec<T> value) { return value.length == 0; }

/// Grows capacity without consuming elements; an error leaves the vector unchanged.
pub fn Result<usize, u32> reserve<T>(ref mut Vec<T> value, usize minimum_capacity) {
    let status = unsafe { reserve_raw(value as ptr mut void, minimum_capacity) };
    return if (status == 0) { .ok(value.bytes / value.element_size) } else { .err(status) };
}

/// Moves one element into the vector and returns the new length. On allocation
/// failure the original element is returned and the vector remains unchanged.
pub fn Result<usize, T> push<T>(ref mut Vec<T> value, T element) {
    var ptr mut void slot = null;
    let status = unsafe { prepare_raw(value as ptr mut void, value.element_size, ref mut slot as ptr mut void) };
    return if (status == 0) {
        assert(slot != null);
        assert(write_slot<T>(slot, element) == 1);
        unsafe { commit(value as ptr mut void, slot, value.element_size); };
        .ok(value.length)
    } else { .err(element) };
}

/// Removes the last element and transfers ownership to the caller.
pub fn Option<T> pop<T>(ref mut Vec<T> value) {
    return if (value.length == 0) { .none } else {
        .some(take_slot<T>(unsafe { pop_slot(value as ptr mut void, value.element_size) }))
    };
}

/// Destroys the tail in reverse order and returns the number of removed elements.
/// Retains capacity. A length at or above the current length changes nothing.
pub fn usize truncate<T>(ref mut Vec<T> value, usize new_length) {
    let count = value.length;
    while (value.length > new_length) {
        let T element = take_slot<T>(unsafe { pop_slot(value as ptr mut void, value.element_size) });
    }
    return count - value.length;
}

/// Destroys all elements in reverse order while retaining capacity.
pub fn usize clear<T>(ref mut Vec<T> value) {
    return truncate<T>(value, 0);
}
