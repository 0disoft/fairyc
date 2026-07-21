module precedence;

fn Result<i32, Error> calculate(Result<i32, Error> input) {
  let a = 1 + 2 * 3;
  let b = (1 + 2) * 3;
  let c = 8 >> 1 + 1;
  let d = 1 | 2 ^ 3 & 4;
  let e = true || false && false;
  let f = input?;
  return .ok(a + b + c + d + f);
}
