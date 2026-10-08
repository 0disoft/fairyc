module array_identity;

// Extended rank-one scalar arrays occupy a closed namespace below tagged IDs.
// Legacy small-array identities are unchanged. The separate composite-array
// namespace encodes one owned child key and a bounded length, never a resource.
pub fn bool large_flat(u32 type){return type>=33554953 && type<=33587206 && (type-33554432)%8>=1 && (type-33554432)%8<=6;}
pub fn bool flat(u32 type){return type>=1033 && type<=1542 && (type-1024)%8>=1 && (type-1024)%8<=6 || large_flat(type);}
pub fn u32 element(u32 type){assert(flat(type));return (type-if(large_flat(type)){33554432}else{1024})%8;}
pub fn usize length(u32 type){assert(flat(type));return ((type-if(large_flat(type)){33554432}else{1024})/8) as usize;}
pub fn u32 count_id(usize count){assert(count<=4096);var usize n=count;var u32 out=0;while(n>0){n-=1;out+=1;}return out;}
pub fn u32 make(u32 element,usize count){return if(element<1 || element>6 || count<1 || count>4096){0}else{if(count<=64){1024}else{33554432}+count_id(count)*8+element};}
