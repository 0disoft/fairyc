module hir_slices;
import hir_model as h;
import hir_state as state;
import syntax as syntax;
import std::bytes as bytes;
import std::vec as v;

// Normalize a fixed-array borrow plus optional subrange without rewriting AST.
// The address operation continues to select original named/field storage.
pub fn usize borrow(syntax::Expression source,h::Type type,usize owner,bool exclusive,ref mut v::Vec<h::Expression> expressions,ref mut state::Context c){
    var h::Expression e=state::typed_expression(.Borrow,type,.None,source.start,source.end);e.left=owner;e.boolean=exclusive;
    return state::push_expression(expressions,e,c);
}
pub fn h::Expression bounds(syntax::Expression source,h::Type type,usize owner,usize low,usize high,ref mut v::Vec<h::Expression> expressions,ref mut bytes::Buffer payload,ref mut state::Context c){
    var h::Expression e=state::typed_expression(.Slice,type,.None,source.start,source.end);e.left=owner;e.right=low;e.alternative=high;e.boolean=high==0;
    if(low==0){
        let usize start=bytes::len(payload);if(!bytes::append_byte(payload,48).is_ok){discard state::fail(c,3,source.start,source.end);return e;}
        var h::Expression zero=state::typed_expression(.Integer,.Usize,.None,source.start,source.end);zero.payload_start=start;zero.payload_end=start+1;
        e.right=state::push_expression(expressions,zero,c);
    }
    if(high==0){var h::Expression length=state::typed_expression(.Length,.Usize,.None,source.start,source.end);length.left=owner;e.alternative=state::push_expression(expressions,length,c);}
    return e;
}
