module transaction;

@contract_uid("example.native-state-borrows")
state {
    cell u64 source;
    cell u64 target;
    cell bool enabled;
    map u64 u64 values;
}

@effects(storage_read)
fn void inspect_source() {
    storage::read("source");
    return;
}

@effects(storage_write)
fn void replace_source() {
    storage::write("source", 99);
    return;
}

@effects(storage_read, storage_write)
fn void transaction() {
    storage::write("source", 42);
    let borrowed = storage::borrow("source");
    let view = storage::view(borrowed, 0, 8);
    let nested = storage::view(view, 0, 8);
    inspect_source();
    let saved = storage::copy(nested);
    replace_source();
    let enabled = storage::read("enabled");
    if (enabled) {
        storage::write("target", saved);
    } else {
        storage::write("values", 7, saved);
    }
    storage::commit();
}
