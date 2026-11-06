module transaction;

@contract_uid("example.native-state-dynamic")
state {
    cell bytes blob;
    cell bytes selected_blob;
    cell text label;
    cell text selected_label;
    map bytes bytes blobs;
    map text text labels;
}

@effects(storage_write)
fn void seed() {
    storage::write("blob", b"\x00\xffAB\x1a\r\n");
    storage::write("label", "가나다");
    return;
}

@effects(storage_read, storage_write)
fn void transaction() {
    seed();
    let bytes key = storage::read("blob");
    let text label = storage::read("label");
    storage::write("blobs", key, key);
    storage::write("labels", label, label);
    let original = storage::borrow("blob");
    let window = storage::view(original, 1, 5);
    let nested = storage::view(window, 1, 3);
    let saved = storage::copy(nested);
    storage::write("blob", b"replaced");
    let borrowed_text = storage::borrow("label");
    let scalar = storage::view(borrowed_text, 3, 3);
    let saved_text = storage::copy(scalar);
    storage::write("label", "교체");
    let previous = storage::read("blobs", key);
    if (storage::found(previous)) {
        storage::write("selected_blob", saved);
    } else {
        storage::reject();
    }
    storage::write("selected_label", saved_text);
    storage::read("selected_blob");
    storage::read("selected_label");
    storage::commit();
}
