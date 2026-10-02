module highlighting;

import std::bytes as bytes;

/// Preview syntax colors, outline, brackets and nested comments in Zed.
const i64 MAX_TOTAL = 9_223_372_036_854_775_807;

struct Entry {
  i64 amount;
}

enum Status {
  Ready;
  Failed { i32 code; };
}

fn i64 identity(ref i64 value) {
  return *value;
}

@effects(filesystem, console)
fn void preview() {
  var bytes::Buffer input = bytes::empty();
  let Result<bytes::Buffer, bytes::BufferError> wrapped = .ok(input);
  var i64[4] totals = [0; 4];
  var u8[8] data = [0; 8];
  let ref mut u8[] part = ref mut data[1..4];
  part[0] = 0x41;
  let ref i64[] shared = ref totals[..];
  let i64 result = identity(ref totals[0]);
  let message = "한국어 // string, not a comment: \n";
  let entry = Entry { .amount = result };
  let status = .Ready;
  /* outer comment
     /* nested comment */ still a comment */
  discard bytes::len(data);
  discard bytes::get<i64>(data);
  discard bytes::Buffer;
  discard entry.show();
  discard shared;
  discard message;
  discard entry;
  discard status;
  discard wrapped;
}
