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
            let ptr mut void slot = unsafe { pop_slot_raw(ref mut this as ptr mut void, this.element_size) };
            assert(slot != null);
            let T value = take_slot<T>(slot);
        }
        let u32 released = unsafe { release_raw(ref mut this as ptr mut void) };
        assert(released == 0);
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
fn T take_slot<T>(ptr mut void slot) { return unsafe { storage_take<T>(slot) }; }
fn i32 write_slot<T>(ptr mut void slot, T value) { return unsafe { storage_put<T>(slot, value); 1 }; }

/// Constructs an empty vector without allocating.
pub fn Vec<T> empty<T>() {
    var Vec<T> value = Vec<T> { .data = null, .bytes = 0, .alignment = 0, .length = 0, .element_size = 0, .element_alignment = 0 };
    assert(unsafe { initialize_raw(ref mut value as ptr mut void, storage_size<T>(), storage_alignment<T>()) } == 0);
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

/// Borrows an initialized element without moving it. An out-of-range index traps.
/// The vector cannot be mutated or moved while the returned reference is live.
pub fn ref T at<T>(ref Vec<T> value, usize index) {
    assert(index < value.length);
    let ptr mut void slot = unsafe { storage_element_address<T>(value.data, index) };
    assert(slot != null);
    return unsafe { storage_borrow<T>(value, slot) };
}

/// Exclusively borrows an initialized element without moving it. An out-of-range index traps.
/// The vector cannot be borrowed again, mutated or moved while the reference is live.
pub fn ref mut T at_mut<T>(ref mut Vec<T> value, usize index) {
    assert(index < value.length);
    let ptr mut void slot = unsafe { storage_element_address<T>(value.data, index) };
    assert(slot != null);
    return unsafe { storage_borrow_mut<T>(value, slot) };
}

/// Returns how many elements fit without growing the allocation.
pub fn usize capacity<T>(ref Vec<T> value) { return value.bytes / value.element_size; }

/// Reports whether no elements are initialized.
pub fn bool is_empty<T>(ref Vec<T> value) { return value.length == 0; }

/// Grows capacity without consuming elements; an error leaves the vector unchanged.
pub fn Result<usize, u32> reserve<T>(ref mut Vec<T> value, usize minimum_capacity) {
    let u32 status = unsafe { reserve_raw(value as ptr mut void, minimum_capacity) };
    return if (status == 0) { .ok(value.bytes / value.element_size) } else { .err(status) };
}

/// Moves one element into the vector and returns the new length. On allocation
/// failure the original element is returned and the vector remains unchanged.
pub fn Result<usize, T> push<T>(ref mut Vec<T> value, T element) {
    var ptr mut void slot = null;
    let u32 status = unsafe { prepare_raw(value as ptr mut void, value.element_size, ref mut slot as ptr mut void) };
    return if (status == 0) {
        assert(slot != null);
        assert(write_slot<T>(slot, element) == 1);
        assert(unsafe { commit_raw(value as ptr mut void, slot, value.element_size) } == 0);
        .ok(value.length)
    } else { .err(element) };
}

/// Removes the last element and transfers ownership to the caller.
pub fn Option<T> pop<T>(ref mut Vec<T> value) {
    return if (value.length == 0) { .none } else {
        let ptr mut void slot = unsafe { pop_slot_raw(value as ptr mut void, value.element_size) };
        assert(slot != null);
        .some(take_slot<T>(slot))
    };
}

/// Destroys the tail in reverse order and returns the number of removed elements.
/// Retains capacity. A length at or above the current length changes nothing.
pub fn usize truncate<T>(ref mut Vec<T> value, usize new_length) {
    let usize count = value.length;
    while (value.length > new_length) {
        let ptr mut void slot = unsafe { pop_slot_raw(value as ptr mut void, value.element_size) };
        assert(slot != null);
        let T element = take_slot<T>(slot);
    }
    return count - value.length;
}

/// Destroys all elements in reverse order while retaining capacity.
pub fn usize clear<T>(ref mut Vec<T> value) {
    return truncate<T>(value, 0);
}
