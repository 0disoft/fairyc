module main;
struct Position { i64 x; i32 y; }
struct Point { Position position; bool active; u8 tint; u32 id; usize count; }

fn i32 main() {
    let Point origin = Point { .position = Position { .x = 4000000000, .y = 2 },
        .active = false, .tint = 3, .id = 4, .count = 5 };
    var Point[2] points = [origin; 2];
    let Point before = points[0];
    points[0].position.x = 6000000000;
    points[0].position.y += 7;
    points[0].active = true;
    points[0].tint *= 2;
    points[0].id |= 8;
    points[0].count <<= 1;
    assert(points[0].position.x == 6000000000 && points[0].position.y == 9);
    assert(points[0].active && points[0].tint == 6 && points[0].id == 12 && points[0].count == 10);
    assert(before.position.x == 4000000000 && !before.active);
    assert(points[1].position.x == 4000000000 && points[1].position.y == 2);
    points[1].position = Position { .x = -5000000000, .y = 8 };
    assert(points[1].position.x == -5000000000 && points[1].position.y == 8);

    var usize index = 0;
    var usize calls = 0;
    var usize order = 0;
    ((points)[{ calls += 1; order = 1; index }]).position.x += {
        assert(order == 1);
        order = 2;
        index = 1;
        20
    };
    assert(calls == 1 && order == 2 && points[0].position.x == 6000000020);
    assert(points[1].position.x == -5000000000);
    points[{ calls += 1; 1 }].position.y = { calls += 1; 17 };
    assert(calls == 3 && points[1].position.y == 17);
    points[points.len - 1].active = true;
    let usize length = points.len;
    for (var usize i = 0; i < length; i += 1) { points[i].position.x += 1; }
    let ref Point[] view = ref points;
    assert(view[0].position.x == 6000000021 && view[1].position.x == -4999999999);
    assert(view[1].active);
    return 0;
}
