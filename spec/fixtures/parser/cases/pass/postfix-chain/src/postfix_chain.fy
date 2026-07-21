module postfix_chain;

fn Result<i32, Error> read(Source source) {
  let value = source.fetch(1, 2).items[0].value?;
  return .ok(value);
}
