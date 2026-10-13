module generic_scalar_arguments;
import std::bytes as bytes;

// Executable type graphs are capped at 4096 nodes. These seven source-only
// argument handles are outside that domain and never become graph records.
// A use as a value type must reject before layout, Flow or IR publication.
pub fn bool descriptor(usize id){return id>4096 && id<=4103;}
pub fn ref u8[] name(usize id){assert(descriptor(id));if(id==4097){return b"i8";}if(id==4098){return b"i16";}if(id==4099){return b"isize";}if(id==4100){return b"u16";}if(id==4101){return b"u64";}if(id==4102){return b"f32";}return b"f64";}
pub fn usize named(ref u8[] spelling){var usize id=4097;while(id<=4103){if(bytes::equal(spelling,name(id))){return id;}id+=1;}return 0;}
