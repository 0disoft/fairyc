module generic_source_documentation;
import generic_source_json as json;
import generic_source_tokens as tokens;
import syntax as syntax;
import scanner as scanner;
import std::bytes as bytes;

pub struct Span{bool present;usize start;usize end;}
fn Span none(){return Span{.present=false,.start=0,.end=0};}
fn bool valid(ref scanner::Stream p,scanner::TokenInfo t){return t.start<=t.end && t.end<=scanner::source_view(p).len;}
fn bool separator(ref scanner::Stream p,usize index,ref mut json::Ledger work){let t=scanner::token_info(p,index);if(t.kind!=2 || t.trivia!=2 || !valid(p,t)){return false;}let value=scanner::source_view(p)[t.start..t.end];if(!json::charge(work,value.len+1)){return false;}if(value.len==0){return false;}var usize at=0;
    if(value[0]==10){at=1;}else if(value.len>=2 && value[0]==13 && value[1]==10){at=2;}else{return false;}
    while(at<value.len){if(value[at]!=32 && value[at]!=9){return false;}at+=1;}return true;
}
fn bool line(ref scanner::Stream p,usize index,ref mut json::Ledger work){let t=scanner::token_info(p,index);if(t.kind!=2 || t.trivia!=5 || !valid(p,t) || t.end-t.start<3){return false;}let source=scanner::source_view(p);if(!json::charge(work,4)){return false;}if(!bytes::equal(source[t.start..t.start+3],b"///")){return false;}var usize at=t.start;
    while(at>0 && source[at-1]!=10){if(!json::charge(work,1)){return false;}at-=1;}
    if(at==0 && source.len>=3 && bytes::equal(source[..3],b"\xef\xbb\xbf")){at=3;}
    while(at<t.start){if(!json::charge(work,1)){return false;}if(source[at]!=32 && source[at]!=9){return false;}at+=1;}return true;
}
// Exact lexer attachment fact, without retaining or interpreting comment prose.
pub fn Span attached(ref scanner::Stream p,usize declaration_token,ref mut json::Ledger work){
    if(declaration_token<=1 || declaration_token>=scanner::token_count(p)){return none();}let d=scanner::token_info(p,declaration_token);if(d.kind==2 || !valid(p,d)){return none();}
    var usize separating=declaration_token-1;var usize document=separating-1;
    if(!separator(p,separating,work) || scanner::token_info(p,separating).end!=d.start || !line(p,document,work) || scanner::token_info(p,document).end!=scanner::token_info(p,separating).start){return none();}
    var usize start=scanner::token_info(p,document).start;let end=scanner::token_info(p,document).end;
    while(document>=2){separating=document-1;let previous=separating-1;if(!separator(p,separating,work) || !line(p,previous,work) || scanner::token_info(p,previous).end!=scanner::token_info(p,separating).start || scanner::token_info(p,separating).end!=scanner::token_info(p,document).start){break;}start=scanner::token_info(p,previous).start;document=previous;}
    return Span{.present=true,.start,.end};
}
pub fn Span declaration(ref syntax::Parsed p,usize index,ref mut json::Ledger work){if(index>=syntax::declaration_count(p) || syntax::problem_count(p)!=0){discard json::reject(work,9);return none();}let d=syntax::declaration_at(p,index);
    if(d.kind!=3 && d.kind!=4 && d.kind!=6 && d.kind!=9 && d.kind!=11){return none();}
    let token=tokens::first(p,d.start,work);if(work.code!=0){return none();}if(token>=syntax::token_count(p) || syntax::token_info(p,token).start!=d.start){discard json::reject(work,9);return none();}return attached(syntax::source_stream(p),token,work);
}
