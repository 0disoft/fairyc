module generic_source_tokens;
import generic_source_json as json;
import syntax as syntax;
import std::bytes as bytes;

pub fn usize first(ref syntax::Parsed p,usize start,ref mut json::Ledger work){var usize low=0;var usize high=syntax::token_count(p);
    while(low<high){if(!json::charge(work,1)){return high;}let middle=low+(high-low)/2;if(syntax::token_info(p,middle).start<start){low=middle+1;}else{high=middle;}}return low;
}
pub fn bool text(ref syntax::Parsed p,usize start,usize end,ref mut bytes::Buffer out,ref mut json::Ledger work){if(start>end || end>syntax::source_view(p).len){return json::reject(work,9);}return json::quoted(out,syntax::source_view(p)[start..end],work);}
// Go path AST text joins significant tokens; written type text stays unchanged.
pub fn bool path(ref syntax::Parsed p,usize start,usize end,ref mut bytes::Buffer out,ref mut json::Ledger work){if(start>end || end>syntax::source_view(p).len){return json::reject(work,9);}var usize index=first(p,start,work);var bytes::Buffer normalized=bytes::empty();
    while(index<syntax::token_count(p)){if(!json::charge(work,1)){return false;}let token=syntax::token_info(p,index);if(token.start>=end){break;}if(token.end>end){return json::reject(work,9);}if(token.kind!=2 && !json::raw(ref mut normalized,syntax::source_view(p)[token.start..token.end],work)){return false;}index+=1;}
    return json::quoted(out,bytes::view(ref normalized),work);
}
