module main;
import points as points;
import std::io as io;

@effects(console)
fn i32 main() {
    let points::Point origin = points::Point { .x = 4000000000, .y = 7 };
    var points::Point[4] polygon = [origin; 4];
    polygon[0].y = 9;
    points::add_x(ref mut polygon[0].x, 10);
    points::add_x(ref mut polygon[3].x, -10);
    let ref i64 first_x = ref polygon[0].x;
    assert(*first_x == 4000000010);
    let ref mut points::Point[] vertices = ref mut polygon;
    points::translate(vertices[1..3], 10, -2);
    let i64 total = points::sum_x(vertices);
    assert(total == 16000000020 && vertices[0].x == 4000000010);
    assert(vertices[0].y == 9 && vertices[3].x == 3999999990);
    assert(vertices[1].y == 5 && vertices[3].y == 7);
    var usize count = 0;
    for (let points::Point point in polygon) {
        assert(point.x >= 3999999990);
        count += 1;
    }
    if (!io::write("vertices=").is_ok || !io::write_usize(count).is_ok
        || !io::write(" sum_x=").is_ok || !io::write_i64(total).is_ok
        || !io::write("\n").is_ok) { return 1; }
    return 0;
}
