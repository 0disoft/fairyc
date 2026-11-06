module module_state_graph_response_common;

// Separate shared structure and byte ledgers; large bounded wire values must
// not consume the much smaller graph-structure allowance.
pub struct Work{usize structure;usize bytes;}
pub fn Work empty_work(){return Work{.structure=0,.bytes=0};}
pub fn bool charge(ref mut Work work,usize cost){if(work.structure>16777216 || cost>16777216-work.structure){return false;}work.structure+=cost;return true;}
pub fn bool byte_charge(ref mut Work work,usize cost){if(work.bytes>1073741824 || cost>1073741824-work.bytes){return false;}work.bytes+=cost;return true;}
pub struct Header{usize status;usize exit;usize trap;usize cost;usize operations;usize events;usize writes;usize write_bytes;usize payload;}
pub struct Event{usize operation;usize site;usize key_start;usize key_end;usize value_start;usize value_end;u8 found;}
pub struct Write{u32[8] key;bool deleted;usize start;usize end;}
pub struct Read{usize operation;bool found;u32 codec;usize start;usize end;}
// Location is structural evidence only. Only the sealed decoder can return an
// admitted Result after all path, observation and final-write checks succeed.
pub struct Step{usize body;usize index;bool operation;}
