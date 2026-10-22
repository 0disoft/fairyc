module main;
resource FirstLeaf { i32 value; cleanup {} }
resource LastLeaf { i32 value; cleanup {} }
resource TemporaryLeaf { i32 value; cleanup {} }
resource Box {
    FirstLeaf first;
    LastLeaf last;
    cleanup { let TemporaryLeaf temporary = TemporaryLeaf { .value = 3 }; }
}
fn i32 main() {
    let Box owner = Box { .first = FirstLeaf { .value = 1 }, .last = LastLeaf { .value = 2 } };
    return 42;
}
