module module_state_graph_response_wire;
import module_state_graph_response_common as common;

// Framing facts only. The sealed decoder owns all graph/path admission.
// Every attempted parse costs one structure unit, even malformed input. Reject
// either overdrawn ledger before doing any work; byte charges precede reads.
fn bool enter(ref mut common::Work work){
    if(!common::charge(work,0) || !common::byte_charge(work,0)){return false;}
    return common::charge(work,1);
}

// Caller has validated at..at+width and width<=8 (64-bit native usize).
pub fn usize number(ref u8[] data,usize at,usize width){
    var usize value=0;var usize i=0;
    while(i<width){value=value*256+(data[at+i] as usize);i+=1;}return value;
}
// Caller has validated exactly 32 accessible bytes. No borrowed view escapes.
pub fn u32[8] key(ref u8[] raw){
    var u32[8] result=[0;8];var usize i=0;
    while(i<8){let at=i*4;
        result[i]=(raw[at] as u32)<<24 | (raw[at+1] as u32)<<16 | (raw[at+2] as u32)<<8 | (raw[at+3] as u32);
        i+=1;
    }return result;
}

pub fn Result<common::Header,u32> header(ref u8[] output,ref u8[] request,ref u8[] request_digest,ref mut common::Work work){
    if(!enter(work)){return .err(1002);}
    if(output.len<186 || output.len>67108864 || request.len<152 || request_digest.len!=32){return .err(1001);}
    // Read the output header (186), request prefix (112), and digest (32).
    if(!common::byte_charge(work,330)){return .err(1002);}
    let out_magic=b"FYGROUT1";let in_magic=b"FYGRIN01";var usize i=0;
    while(i<8){if(output[i]!=out_magic[i] || request[i]!=in_magic[i]){return .err(1001);}i+=1;}
    i=8;while(i<112){if(output[i]!=request[i]){return .err(1001);}i+=1;}
    i=0;while(i<32){if(output[112+i]!=request_digest[i]){return .err(1001);}i+=1;}
    let h=common::Header{.status=output[144] as usize,.exit=output[145] as usize,
        .trap=number(output,146,4),.cost=number(output,150,8),.operations=number(output,158,8),
        .events=number(output,166,4),.writes=number(output,170,4),
        .write_bytes=number(output,174,8),.payload=number(output,182,4)};
    if(h.status>11 || h.exit>2 || h.payload!=output.len-186){return .err(1001);}
    // Divide before multiplying untrusted counts, and subtract only after the
    // first minimum fits. No count sum or multiplication can overflow.
    if(h.events>h.payload/21){return .err(1001);}
    if(h.writes>(h.payload-h.events*21)/37){return .err(1001);}
    if((h.status!=0 || h.exit!=0) && (h.writes!=0 || h.write_bytes!=0)){return .err(1001);}
    if(h.status!=0 && (h.exit!=0 || h.trap!=0)){return .err(1001);}
    return .ok(h);
}

// Failure leaves the cursor unchanged, but never refunds either work ledger.
pub fn Result<common::Event,u32> event(ref u8[] output,ref mut usize cursor,ref mut common::Work work){
    if(!enter(work)){return .err(1002);}let at=*cursor;
    if(at>output.len || output.len-at<21){return .err(1001);}
    if(!common::byte_charge(work,21)){return .err(1002);}
    let key_size=number(output,at+12,4);let value_size=number(output,at+16,4);let start=at+21;
    if(key_size>output.len-start){return .err(1001);}let middle=start+key_size;
    if(value_size>output.len-middle){return .err(1001);}let end=middle+value_size;
    let e=common::Event{.operation=number(output,at,8),.site=number(output,at+8,4),
        .key_start=start,.key_end=middle,.value_start=middle,.value_end=end,.found=output[at+20]};
    *cursor=end;return .ok(e);
}

pub fn Result<common::Write,u32> write(ref u8[] output,ref mut usize cursor,ref mut common::Work work){
    if(!enter(work)){return .err(1002);}let at=*cursor;
    if(at>output.len || output.len-at<37){return .err(1001);}
    if(!common::byte_charge(work,37)){return .err(1002);}
    let size=number(output,at+33,4);let start=at+37;
    if(output[at+32]>1 || size>output.len-start){return .err(1001);}let end=start+size;
    let w=common::Write{.key=key(output[at..at+32]),.deleted=output[at+32]==1,.start,.end};
    *cursor=end;return .ok(w);
}
