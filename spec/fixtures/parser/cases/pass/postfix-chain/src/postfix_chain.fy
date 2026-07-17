module postfix_chain;

fn read(source: Source) -> Result<i32, Error> {
  let value = source.fetch(1, 2).items[0].value?;
  return Result::Ok(value);
}
