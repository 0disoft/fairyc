module points;
pub struct Point { i64 x; i64 y; }

pub fn void add_x(ref mut i64 x, i64 amount) { *x = *x + amount; }

pub fn void translate(ref mut Point[] points, i64 dx, i64 dy) {
    for (let ref mut Point point in points) {
        point.x += dx;
        point.y += dy;
    }
}

pub fn i64 sum_x(ref Point[] points) {
    var i64 total = 0;
    for (let ref Point point in points) { total += point.x; }
    return total;
}
