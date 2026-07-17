module precedence;

fn calculate(input: Result<i32, Error>) -> Result<i32, Error> {
  let a = 1 + 2 * 3;
  let b = (1 + 2) * 3;
  let c = 8 >> 1 + 1;
  let d = 1 | 2 ^ 3 & 4;
  let e = true || false && false;
  let f = input?;
  return Result::Ok(a + b + c + d + f);
}
