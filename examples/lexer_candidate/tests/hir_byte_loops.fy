module tests::hir_byte_loops;
import tests::hir_nominal as fixture;
import hir_model as h;
import mir_lower as lower;
import mir_verify as verify;
import std::bytes as bytes;

@test
fn void owned_byte_iteration_survives_parser_and_targets_loop_exits() {
    let ref u8[] source=b"module main;fn usize count(ref u8[] data){var usize result=0;for(let u8 byte in data){if(byte==0){continue;}result+=byte as usize;if(byte==9){break;}}return result;}fn i32 main(){return 7;}";
    let h::Program p=fixture::retained(source);
    assert(bytes::equal(h::source_view(ref p),source));
    var usize loop=0;var usize loops=0;var usize exits=0;var usize i=1;
    while(i<=h::statement_count(ref p)) {
        let s=h::statement_at(ref p,i);
        if(h::statement_code(s.kind)==6) {
            loop=i;loops+=1;
            let body=h::block_at(ref p,s.body);
            let item=h::statement_at(ref p,h::edge_at(ref p,body.edge_start));
            let value=h::expression_at(ref p,item.expression);
            assert(h::statement_code(item.kind)==1 && h::expression_code(value.kind)==13 && h::type_code(value.type)==3);
            assert(bytes::equal(h::binding_text(ref p,item.binding),b"byte"));
            assert(h::binding_code(h::binding_at(ref p,item.binding).kind)==3);
            let initial=h::statement_at(ref p,s.initial);
            assert(h::binding_at(ref p,initial.binding).scope==s.scope);
            assert(bytes::equal(h::integer_text(ref p,initial.expression),b"0"));
            assert(h::operator_code(h::statement_at(ref p,s.post).operator)==1);
        }
        i+=1;
    }
    assert(loops==1 && loop!=0);
    i=1;while(i<=h::statement_count(ref p)) {
        let s=h::statement_at(ref p,i);let u32 kind=h::statement_code(s.kind);
        if(kind==7 || kind==8){assert(s.loop_target==loop);exits+=1;}
        i+=1;
    }
    assert(exits==2);
    let mir=switch(lower::lower(p)){case .ok(value):{value}case .err(_):{assert(false);return;}};
    assert(verify::check(ref mir).code==0);
}
