module module_inline_packages;
import module_inline_declarations as declarations;
import module_inline_identity as identity;
import module_signatures as signatures;
import module_public_type_text as coordinates;
import generic_source_json as json;
import project_cache_encoding as budget;
import syntax as syntax;
import sha256 as sha;
import std::bytes as bytes;
import std::vec as v;

struct Row{usize first;usize end;bool loop;u32[8] hash;}
pub fn Result<bytes::Buffer,u32> text(u32[8] hash,ref u8[] prefix){var bytes::Buffer out=bytes::empty();if(!bytes::append(ref mut out,prefix).is_ok){return .err(11);}let digits=b"0123456789abcdef";var usize i=0;while(i<8){var u32 word=hash[i];var usize j=0;while(j<8){if(!bytes::append_byte(ref mut out,digits[((word>>28)&15) as usize]).is_ok){return .err(11);}word=(word&268435455)<<4;j+=1;}i+=1;}return .ok(out);}
// Called only after complete source/body checking. No caller-provided class is
// admitted, and public-interface v5 remains independent of these body bytes.
pub fn Result<u32[8],u32> derive(ref v::Vec<syntax::Parsed> sources,ref signatures::Program metadata,ref coordinates::Coordinates table,usize owner,ref mut usize work){
    var v::Vec<Row> rows=v::empty<Row>();var bytes::Buffer names=bytes::empty();var usize m=0;
    while(m<v::len<syntax::Parsed>(sources)){if(!budget::charge(work,1)){return .err(10);}if(signatures::module_owner(metadata,m)==owner){let p=v::at<syntax::Parsed>(sources,m);var usize d=0;while(d<syntax::declaration_count(p)){if(!budget::charge(work,1)){return .err(10);}let info=declarations::read(p,d);if(info.code!=0){return .err(9);}if(info.annotated){let first=bytes::len(ref names);let path_decl=syntax::declaration_at(p,0);var usize t=0;while(t<syntax::token_count(p)){let token=syntax::token_info(p,t);if(token.start>=path_decl.name_end){break;}if(token.start>=path_decl.name_start && token.kind!=2){let raw=syntax::source_view(p)[token.start..token.end];if(!budget::charge(work,raw.len+1)){return .err(10);}if(!bytes::append(ref mut names,raw).is_ok){return .err(11);}}t+=1;}let decl=syntax::declaration_at(p,d);if(!bytes::append(ref mut names,b"::").is_ok || !bytes::append(ref mut names,syntax::source_view(p)[decl.name_start..decl.name_end]).is_ok || !v::push<Row>(ref mut rows,Row{.first,.end=bytes::len(ref names),.loop=info.loop,.hash=info.hash}).is_ok){return .err(11);}}d+=1;}}m+=1;}
    if(v::len<Row>(ref rows)==0){return .ok([0;8]);}var usize i=1;while(i<v::len<Row>(ref rows)){let row=*v::at<Row>(ref rows,i);var usize at=i;while(at>0){let prev=*v::at<Row>(ref rows,at-1);if(!budget::charge(work,row.end-row.first+prev.end-prev.first+1)){return .err(10);}if(!budget::before(bytes::view(ref names)[row.first..row.end],bytes::view(ref names)[prev.first..prev.end])){break;}let cell=v::at_mut<Row>(ref mut rows,at);*cell=prev;at-=1;}let cell=v::at_mut<Row>(ref mut rows,at);*cell=row;i+=1;}
    var usize coordinate=coordinates::coordinate_count(table);i=0;while(i<coordinates::coordinate_count(table)){if(coordinates::coordinate_at(table,i).owner==owner){if(coordinate!=coordinates::coordinate_count(table)){return .err(9);}coordinate=i;}i+=1;}if(coordinate==coordinates::coordinate_count(table) || !coordinates::valid_coordinate(coordinates::coordinate_text(table,coordinate))){return .err(9);}
    var bytes::Buffer out=bytes::empty();var json::Ledger ledger=json::empty();if(!json::raw(ref mut out,b"{\"Version\":\"fairyc.inline-bodies/v1\",\"Owner\":",ref mut ledger) || !json::quoted(ref mut out,coordinates::coordinate_text(table,coordinate),ref mut ledger) || !json::raw(ref mut out,b",\"Entries\":[",ref mut ledger)){return .err(ledger.code);}i=0;
    while(i<v::len<Row>(ref rows)){let row=*v::at<Row>(ref rows,i);let name=bytes::view(ref names)[row.first..row.end];if(i>0){let previous=*v::at<Row>(ref rows,i-1);if(bytes::equal(name,bytes::view(ref names)[previous.first..previous.end])){return .err(9);}if(!json::raw(ref mut out,b",",ref mut ledger)){return .err(ledger.code);}}let body=switch(text(row.hash,if(row.loop){b"fyinlinefn2-sha256-"}else{b"fyinlinefn1-sha256-"})){case .ok(p):{p}case .err(code):{return .err(code);}};if(!json::raw(ref mut out,b"{\"Name\":",ref mut ledger) || !json::quoted(ref mut out,name,ref mut ledger) || !json::raw(ref mut out,b",\"Body\":",ref mut ledger) || !json::quoted(ref mut out,bytes::view(ref body),ref mut ledger) || !json::raw(ref mut out,b"}",ref mut ledger)){return .err(ledger.code);}i+=1;}
    if(!json::raw(ref mut out,b"]}",ref mut ledger)){return .err(ledger.code);}if(!budget::charge(work,ledger.used+bytes::len(ref out)+1)){return .err(10);}return .ok(sha::digest(bytes::view(ref out)));
}
