module window_metrics;
pub fn bool valid_dpi(i32 dpi){return dpi>=96 && dpi<=960;}
pub fn i32 scale(i32 value,i32 dpi){return (value*dpi+48)/96;}
pub fn i32 cap(i32 available,i32 logical,i32 dpi){let limit=scale(logical,dpi);return if(available>limit){limit}else{available};}
pub fn i32 center(i32 available,i32 extent){return (available-extent)/2;}
// Raw Win32 navigation remains an event until the Fairy caller chooses policy.
pub fn i32 key_action(i32 key,i32 flags){
    if(key==13 && flags==0){return 1;}if(key==9 && flags==0){return 2;}
    if(key==9 && flags==1){return 3;}if(key==27 && flags==0){return 4;}return 0;
}
