module native_runtime;
import native_runtime_contract as contract;
import std::bytes as bytes;

// Allocation-free selection; the packet owner retains the diagnostic span.
// Component IDs and order belong to native_runtime_contract. Host: 0 none,
// 1 process, 2 process followed by rooted file. No C asset bytes live here.
pub struct Plan {u32 components;bool vectors;u32 scalar_exit;u32 host;}
// 101 target-specific, 102 unknown std, 104 unknown private Windows host.
pub struct Problem {u32 kind;usize start;usize end;}
fn bool at(ref u8[] data,usize start,ref u8[] prefix){return start<=data.len && prefix.len<=data.len-start && bytes::equal(data[start..start+prefix.len],prefix);}
fn bool word(u8 c){return c>=48 && c<=57 || c>=65 && c<=90 || c>=97 && c<=122 || c==95;}
// Go regexp's ASCII \s excludes vertical tab.
fn bool space(u8 c){return c==9 || c==10 || c==12 || c==13 || c==32;}
fn bool before(ref u8[] a,ref u8[] b){var usize i=0;while(i<a.len && i<b.len){if(a[i]!=b[i]){return a[i]<b[i];}i+=1;}return a.len<b.len;}
pub fn bool contains(ref Plan plan,u32 id){return id>=1 && id<=14 && (plan.components & (1<<(id-1)))!=0;}
// Non-stdlib foreign names can be classified directly by the closed contract;
// header scanning intentionally matches only the same prefixes as Go.
pub fn u32 symbol_kind(ref u8[] name){return contract::classify(name);}
pub fn Result<Plan,Problem> select(ref u8[] header,bool trap_control){
    var Plan plan=Plan{.components=0,.vectors=false,.scalar_exit=if(trap_control){128}else{100},.host=0};
    var Problem failure=Problem{.kind=0,.start=0,.end=0};var usize i=0;
    while(i<header.len){
        if(at(header,i,b"fy_private_vector_")){plan.vectors=true;}
        if((i==0 || !word(header[i-1])) && at(header,i,b"fy_std_")){
            var usize end=i+7;while(end<header.len && word(header[end])){end+=1;}var usize call=end;while(call<header.len && space(header[call])){call+=1;}
            if(end>i+7 && call<header.len && header[call]==40){let name=header[i..end];let u32 kind=contract::classify(name);
                if(kind<=14){plan.components=plan.components | contract::closure(kind);}
                else if(failure.kind==0 || before(name,header[failure.start..failure.end])){failure=Problem{.kind,.start=i,.end};}
            }
        }i+=1;
    }
    // The Go bridge sorts/deduplicates std symbols before validating any host.
    if(failure.kind!=0){return .err(failure);}if(plan.vectors){plan.components=plan.components | 64;}
    i=0;while(i<header.len){
        if((i==0 || !word(header[i-1])) && at(header,i,b"fairyc_private_windows_")){
            var usize end=i+23;while(end<header.len && word(header[end])){end+=1;}var usize call=end;while(call<header.len && space(header[call])){call+=1;}
            if(end>i+23 && call<header.len && header[call]==40){
                // WindowsSource trims SP/TAB/CR/LF, but not form feed.
                var usize trimmed=call;while(trimmed>end && (header[trimmed-1]==32 || header[trimmed-1]==9 || header[trimmed-1]==10 || header[trimmed-1]==13)){trimmed-=1;}
                let u32 host=contract::host(header[i..trimmed]);if(host==0){return .err(Problem{.kind=104,.start=i,.end=trimmed});}if(host>plan.host){plan.host=host;}
                i=call;
            }
        }i+=1;
    }return .ok(plan);
}
// Exact bridge diagnostic bytes for spans produced by select. Std spans are
// ASCII identifiers; host spans may retain form feed, matching WindowsSource.
pub fn bool diagnostic(ref u8[] header,Problem problem,ref mut bytes::Buffer output){
    if(problem.start>problem.end || problem.end>header.len){return false;}let name=header[problem.start..problem.end];
    if(problem.kind==104){return bytes::append(output,b"unknown private Windows host symbol ").is_ok && bytes::append(output,name).is_ok;}
    if(problem.kind==101){return bytes::append(output,b"Stage 0 standard-library symbol \"").is_ok && bytes::append(output,name).is_ok && bytes::append(output,b"\" has no x86_64-windows-msvc source contract").is_ok;}
    if(problem.kind==102){return bytes::append(output,b"unrecognized Stage 0 standard-library symbol \"").is_ok && bytes::append(output,name).is_ok && bytes::append(output,b"\"").is_ok;}return false;
}
