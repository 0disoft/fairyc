module generic_source_observations;
import generic_source_expression_json as expressions;
import generic_source_tokens as tokens;
import generic_source_json as json;
import syntax as syntax;
import scanner as scanner;
import std::bytes as bytes;
import std::vec as v;

// Original binding order follows written starts. Literal ordering also retains
// parameter-before-result CST children for function declarations and C types.
// Byte-for synthetic AST bindings and literals are not source CST observations.
struct Row{usize start;usize id;usize end;bool token_fact;}
fn void swap(ref mut v::Vec<Row> rows,usize a,usize b){let left=*v::at<Row>(rows,a);let right=*v::at<Row>(rows,b);{let slot=v::at_mut<Row>(rows,a);*slot=right;}{let slot=v::at_mut<Row>(rows,b);*slot=left;}}
fn bool sift(ref mut v::Vec<Row> rows,usize root,usize count,ref mut json::Ledger work){var usize at=root;while(at<count/2){if(!json::charge(work,2)){return false;}var usize child=at*2+1;if(child+1<count && v::at<Row>(rows,child).start<v::at<Row>(rows,child+1).start){child+=1;}if(v::at<Row>(rows,at).start>=v::at<Row>(rows,child).start){return true;}swap(rows,at,child);at=child;}return true;}
fn bool order(ref mut v::Vec<Row> rows,ref mut json::Ledger work){var usize count=v::len<Row>(rows);var usize at=count/2;while(at>0){at-=1;if(!sift(rows,at,count,work)){return false;}}while(count>1){count-=1;swap(rows,0,count);if(!sift(rows,0,count,work)){return false;}}at=1;while(at<v::len<Row>(rows)){if(!json::charge(work,1)){return false;}if(v::at<Row>(rows,at-1).start>=v::at<Row>(rows,at).start){return json::reject(work,9);}at+=1;}return true;}
fn bool push(ref mut v::Vec<Row> rows,Row row,ref mut json::Ledger work){return json::charge(work,1) && if(v::push<Row>(rows,row).is_ok){true}else{json::reject(work,11)};}
fn bool clean(ref syntax::Parsed p,ref mut json::Ledger work){return if(syntax::problem_count(p)==0 && syntax::declaration_count(p)!=0 && syntax::expression_count(p)<=16384 && syntax::type_count(p)<=4096 && syntax::statement_count(p)<=4096){true}else{json::reject(work,9)};}
fn bool collect_bindings(ref syntax::Parsed p,ref mut v::Vec<Row> rows,ref mut json::Ledger work){if(!json::charge(work,1) || !clean(p,work)){return false;}var v::Vec<u8> initializers=v::empty<u8>();var usize index=0;
    while(index<syntax::statement_count(p)){if(!json::charge(work,1)){return false;}if(!v::push<u8>(ref mut initializers,0).is_ok){return json::reject(work,11);}index+=1;}
    index=1;while(index<=syntax::statement_count(p)){if(!json::charge(work,1)){return false;}let s=syntax::statement_at(p,index);if(s.kind==15 && s.first!=0){if(s.first>syntax::statement_count(p)){return json::reject(work,9);}let slot=v::at_mut<u8>(ref mut initializers,s.first-1);*slot=1;}index+=1;}
    index=1;
    while(index<=syntax::statement_count(p)){if(!json::charge(work,1)){return false;}let s=syntax::statement_at(p,index);if(s.kind==2){var usize end=s.end;if(*v::at<u8>(ref initializers,index-1)!=0){if(s.expression==0 || s.expression>syntax::expression_count(p)){return json::reject(work,9);}end=syntax::expression_at(p,s.expression).end;}
        if(s.start>end || end>syntax::source_view(p).len || !push(rows,Row{.start=s.start,.id=index,.end,.token_fact=false},work)){return json::reject(work,9);}}
        index+=1;
    }
    return order(rows,work);
}

fn ref u8[] literal_kind(ref u8[] raw){if(bytes::equal(raw,b"true") || bytes::equal(raw,b"false")){return b"boolean";}if(bytes::equal(raw,b"null")){return b"null";}if(raw.len>=2 && raw[1]==34 && raw[0]==98){return b"byte-string";}if(raw.len>=2 && raw[1]==34 && raw[0]==99){return b"c-string";}if(raw.len!=0 && raw[0]==34){return b"string";}if(raw.len>=2 && raw[0]==48 && raw[1]==120){return b"integer";}var usize index=0;while(index<raw.len){if(raw[index]==46 || raw[index]==101 || raw[index]==69){return b"float";}index+=1;}return b"integer";}
// Children of each C function type and source function visit parameters before
// the earlier-written result. Rotate only that interval, after nested types.
fn bool reverse(ref mut v::Vec<Row> rows,usize first,usize stop,ref mut json::Ledger work){var usize low=first;var usize high=stop;while(high>low && high-low>1){if(!json::charge(work,1)){return false;}high-=1;swap(rows,low,high);low+=1;}return true;}
fn bool result_last(ref mut v::Vec<Row> rows,usize start,usize end,usize limit,ref mut json::Ledger work){if(start>end || end>limit){return json::reject(work,9);}let count=v::len<Row>(rows);var usize first=count;var usize stop=count;var usize last=count;var usize index=0;
    while(index<count){if(!json::charge(work,1)){return false;}let row=*v::at<Row>(rows,index);if(row.start>=start && row.end<=end){if(first==count){first=index;}stop=index+1;}if(row.start>=start && row.end<=limit){last=index+1;}index+=1;}
    if(first==count || last<=stop){return true;}index=first;while(index<stop){let row=*v::at<Row>(rows,index);if(!json::charge(work,1)){return false;}if(row.start<start || row.end>end){return json::reject(work,9);}index+=1;}
    return reverse(rows,first,stop,work) && reverse(rows,stop,last,work) && reverse(rows,first,last,work);
}
fn bool literal_order(ref syntax::Parsed p,ref mut v::Vec<Row> rows,ref mut json::Ledger work){var usize index=1;while(index<=syntax::type_count(p)){if(!json::charge(work,1)){return false;}let n=syntax::type_at(p,index);if(n.kind==15){if(n.left==0 || n.left>=index){return json::reject(work,9);}let result=syntax::type_at(p,n.left);if(!result_last(rows,result.start,result.end,n.end,work)){return false;}}index+=1;}
    index=0;while(index<syntax::declaration_count(p)){if(!json::charge(work,1)){return false;}let d=syntax::declaration_at(p,index);if(d.kind==3 || d.kind==9){if(d.result_type==0 || d.result_type>syntax::type_count(p) || d.entry_start>syntax::entry_count(p) || d.entry_count>syntax::entry_count(p)-d.entry_start){return json::reject(work,9);}let result=syntax::type_at(p,d.result_type);var usize limit=result.end;var usize at=0;
        while(at<d.entry_count){if(!json::charge(work,syntax::entry_count(p)+1)){return false;}let e=syntax::entry_at(p,d.entry_start+at);if(e.kind==1 && e.end>limit){limit=e.end;}at+=1;}if(!result_last(rows,result.start,result.end,limit,work)){return false;}}
        index+=1;
    }return true;
}
fn bool literal_rows(ref syntax::Parsed p,ref mut v::Vec<Row> rows,ref mut json::Ledger work){if(!json::charge(work,1) || !clean(p,work)){return false;}var usize index=1;
    while(index<=syntax::expression_count(p)){if(!json::charge(work,1)){return false;}let n=syntax::expression_at(p,index);if(n.kind==1){if(n.start>n.end || n.end>syntax::source_view(p).len || n.token>=syntax::token_count(p) || !push(rows,Row{.start=n.start,.id=index,.end=n.end,.token_fact=false},work)){return json::reject(work,9);}}index+=1;}
    index=1;while(index<=syntax::type_count(p)){if(!json::charge(work,1)){return false;}let n=syntax::type_at(p,index);if(n.kind==9){let token=tokens::first(p,n.start,work);if(work.code!=0 || token>=syntax::token_count(p) || syntax::token_info(p,token).start!=n.start || syntax::token_info(p,token).end!=n.end){return json::reject(work,9);}if(n.start>n.end || n.end>syntax::source_view(p).len || !push(rows,Row{.start=n.start,.id=token,.end=n.end,.token_fact=true},work)){return json::reject(work,9);}}index+=1;}
    index=0;while(index<syntax::attribute_argument_count(p)){if(!json::charge(work,1)){return false;}let a=syntax::attribute_argument_at(p,index);if(a.kind==2 || a.kind==3){if(a.token>=syntax::token_count(p) || a.start>a.end || a.end>syntax::source_view(p).len || !push(rows,Row{.start=a.start,.id=a.token,.end=a.end,.token_fact=true},work)){return json::reject(work,9);}}index+=1;}
    return order(rows,work) && literal_order(p,rows,work);
}
// Keep only ordered source coordinates between chunks, never accumulated JSON.
pub resource Rows{v::Vec<Row> entries;bool binding_rows;cleanup{}}
pub fn Result<Rows,u32> prepare(ref syntax::Parsed p,bool binding_rows,ref mut json::Ledger work){
    var v::Vec<Row> rows=v::empty<Row>();let ok=if(binding_rows){collect_bindings(p,ref mut rows,work)}else{literal_rows(p,ref mut rows,work)};
    if(!ok){return .err(work.code);}return .ok(Rows{.entries=rows,.binding_rows});
}
pub fn usize count(ref Rows rows){return v::len<Row>(ref rows.entries);}
pub fn bool row(ref Rows rows,ref syntax::Parsed p,usize index,ref mut bytes::Buffer out,ref mut json::Ledger work){
    if(index>=count(rows)){return json::reject(work,9);}if(!json::charge(work,1)){return false;}let item=*v::at<Row>(ref rows.entries,index);
    if(rows.binding_rows){return expressions::binding(p,syntax::statement_at(p,item.id),item.end,0,out,work);}
    let raw=syntax::source_view(p)[item.start..item.end];if(!json::charge(work,raw.len+1)){return false;}
    if(!json::raw(out,b"{\"Raw\":",work) || !json::quoted(out,raw,work) || !json::raw(out,b",\"Kind\":",work) || !json::quoted(out,literal_kind(raw),work) || !json::raw(out,b",\"Cooked\":",work)){return false;}
    if(item.token_fact){if(!json::quoted(out,scanner::token_cooked(syntax::source_stream(p),item.id),work)){return false;}}
    else if(!json::quoted(out,syntax::literal_cooked(p,item.id),work)){return false;}
    return json::raw(out,b"}",work);
}
fn bool list(ref syntax::Parsed p,bool binding_rows,ref mut bytes::Buffer out,ref mut json::Ledger work){
    let rows=switch(prepare(p,binding_rows,work)){case .ok(value):{value}case .err(_):{return false;}};
    if(count(ref rows)==0){return json::raw(out,b"null",work);}if(!json::raw(out,b"[",work)){return false;}var usize index=0;
    while(index<count(ref rows)){if(index!=0 && !json::raw(out,b",",work)){return false;}if(!row(ref rows,p,index,out,work)){return false;}index+=1;}return json::raw(out,b"]",work);
}
pub fn bool bindings(ref syntax::Parsed p,ref mut bytes::Buffer out,ref mut json::Ledger work){return list(p,true,out,work);}
pub fn bool literals(ref syntax::Parsed p,ref mut bytes::Buffer out,ref mut json::Ledger work){return list(p,false,out,work);}

pub fn Result<bytes::Buffer,u32> encode(ref syntax::Parsed p,bool binding_rows,ref mut usize work){var json::Ledger ledger=json::Ledger{.used=*work,.code=0};var bytes::Buffer out=bytes::empty();let ok=if(binding_rows){bindings(p,ref mut out,ref mut ledger)}else{literals(p,ref mut out,ref mut ledger)};*work=ledger.used;if(!ok){return .err(ledger.code);}return .ok(out);}
