module hir_slices;
import hir_model as h;
import hir_state as state;
import syntax as syntax;
import std::bytes as bytes;
import std::vec as v;

// Normalize a fixed-array borrow plus optional subrange without rewriting AST.
// The address operation continues to select original named/field storage.
pub fn usize borrow(syntax::Expression source,h::Type type,usize owner,bool exclusive,ref mut v::Vec<h::Expression> expressions,ref mut state::Context c){
    return state::push_expression(expressions,h::Expression{.kind=.Borrow,.type,.operator=.None,.start=source.start,.end=source.end,.left=owner,.right=0,.alternative=0,.block=0,.binding=0,.edge_start=0,.edge_count=0,.payload_start=0,.payload_end=0,.boolean=exclusive},c);
}
pub fn h::Expression bounds(syntax::Expression source,h::Type type,usize owner,usize low,usize high,ref mut v::Vec<h::Expression> expressions,ref mut bytes::Buffer payload,ref mut state::Context c){
    var h::Expression e=h::Expression{.kind=.Slice,.type,.operator=.None,.start=source.start,.end=source.end,.left=owner,.right=low,.alternative=high,.block=0,.binding=0,.edge_start=0,.edge_count=0,.payload_start=0,.payload_end=0,.boolean=high==0};
    if(low==0){
        let usize start=bytes::len(payload);if(!bytes::append_byte(payload,48).is_ok){discard state::fail(c,3,source.start,source.end);return e;}
        e.right=state::push_expression(expressions,h::Expression{.kind=.Integer,.type=.Usize,.operator=.None,.start=source.start,.end=source.end,.left=0,.right=0,.alternative=0,.block=0,.binding=0,.edge_start=0,.edge_count=0,.payload_start=start,.payload_end=start+1,.boolean=false},c);
    }
    if(high==0){e.alternative=state::push_expression(expressions,h::Expression{.kind=.Length,.type=.Usize,.operator=.None,.start=source.start,.end=source.end,.left=owner,.right=0,.alternative=0,.block=0,.binding=0,.edge_start=0,.edge_count=0,.payload_start=0,.payload_end=0,.boolean=false},c);}
    return e;
}
