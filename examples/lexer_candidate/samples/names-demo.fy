module names_demo;

fn i64 total(i64 price, i64 fee) {
    let i64 result = price + fee;
    {
        // The initializer sees the outer result; this block owns a new name.
        let i64 result = result + 1;
        result;
    }
    return result;
}
