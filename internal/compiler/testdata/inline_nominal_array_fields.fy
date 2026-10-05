module main;
struct APage { ZToken[2] tokens; usize count; }
struct Envelope { APage page; }
struct ZToken { Position position; bool active; u8 tint; u32 id; usize size; }
struct Position { i64 value; i32 line; }

fn APage relay(APage page) { return page; }
fn void replace(ref mut APage page, ZToken token) { page.tokens = [token; 2]; }
fn i32 main() {
    let ZToken origin = ZToken { .position = Position { .value = 4000000000, .line = 2 },
        .active = false, .tint = 3, .id = 4, .size = 5 };
    let ZToken changed = ZToken { .position = Position { .value = -6000000000, .line = 7 },
        .active = true, .tint = 9, .id = 12, .size = 10 };
    var APage page = APage { .tokens = [origin, changed], .count = 2 };
    let APage snapshot = page;
    let ZToken selected = page.tokens[0];
    let Option<APage> wrapped = .some(page);
    var ZToken[2] extracted = page.tokens;
    extracted[0].position.value = 42;
    let ZToken[2] before = snapshot.tokens;
    assert(before[0].position.value == 4000000000 && before[1].position.value == -6000000000);
    replace(ref mut page, changed);
    var APage copied = relay(page);
    copied.tokens = [origin; 2];
    let ZToken[2] live = page.tokens;
    assert(live[0].position.value == -6000000000 && live[0].active && live[0].tint == 9);
    assert(live[1].id == 12 && live[1].size == 10 && extracted[0].position.value == 42);
    assert(selected.position.value == 4000000000);
    switch (wrapped) {
        case .some(value): {
            let ZToken[2] saved = value.tokens;
            assert(saved[0].position.value == 4000000000 && !saved[0].active);
        }
        case .none: { assert(false); }
    };
    var Envelope envelope = Envelope { .page = copied };
    let Envelope old = envelope;
    envelope.page.tokens = [changed; 2];
    let ZToken[2] nested = envelope.page.tokens;
    let ZToken[2] independent = old.page.tokens;
    assert(nested[0].position.line == 7 && independent[0].position.line == 2);
    var APage[1] pages = [page];
    pages[0] = copied;
    let APage item = pages[0];
    let ZToken[2] items = item.tokens;
    assert(items[1].position.value == 4000000000);
    return 0;
}
