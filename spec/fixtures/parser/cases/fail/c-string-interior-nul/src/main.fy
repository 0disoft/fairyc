module main;

fn main() -> &CStr {
  let name = c"fairy\0c";
  return name;
}
