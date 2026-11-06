module linux_fetch_deadline;
@effects(clock) @link_name("fairyc_private_linux_monotonic_ms_v1") unsafe extern C fn u32 now(ptr mut usize output);
pub struct Deadline{usize started;}
pub fn usize duration(){return 120000;}
pub fn usize remaining_at(usize started,usize current){return if(current<started || current-started>=duration()){0}else{duration()-(current-started)};}
@effects(clock) pub fn Result<Deadline,u32> start(){var usize current=0;let code=unsafe{now(ref mut current as ptr mut usize)};return if(code==0){.ok(Deadline{.started=current})}else{.err(code)};}
@effects(clock) pub fn usize remaining(Deadline value){var usize current=0;let code=unsafe{now(ref mut current as ptr mut usize)};return if(code==0){remaining_at(value.started,current)}else{0};}
@effects(clock) pub fn bool expired(Deadline value){return remaining(value)==0;}
