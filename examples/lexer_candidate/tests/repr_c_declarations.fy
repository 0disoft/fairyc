module tests::repr_c_declarations;
import repr_c_declarations as repr_c;
import c_abi_types as c_types;
import pointer_types as pointers;
import syntax as syntax;
import scanner as scanner;
import std::bytes as bytes;

fn syntax::Parsed parsed(ref u8[] input){var bytes::Buffer b=bytes::empty();assert(bytes::append(ref mut b,input).is_ok);let s=switch(scanner::collect(b,false,32768,1048576,256,16777216)){case .ok(s):{s}case .err(_):{assert(false);return parsed(b"module a;");}};let p=switch(syntax::parse(s,512)){case .ok(p):{p}case .err(_):{assert(false);return parsed(b"module a;");}};assert(syntax::problem_count(ref p)==0);return p;}
fn usize struct_at(ref syntax::Parsed p,usize ordinal){var usize i=0;var usize seen=0;while(i<syntax::declaration_count(p)){if(syntax::declaration_at(p,i).kind==4){if(seen==ordinal){return i;}seen+=1;}i+=1;}assert(false);return 0;}
fn repr_c::Info read_struct(ref u8[] input,usize ordinal){let p=parsed(input);return repr_c::read(ref p,struct_at(ref p,ordinal));}
fn u32 struct_code(ref u8[] input){return read_struct(input,0).code;}
fn bool struct_layout(ref u8[] input){let p=parsed(input);return repr_c::layout(ref p,struct_at(ref p,0));}

fn void admitted_marker_and_value_lane(){
    let p=parsed(b"module m;@repr(C)struct Point{i32 x;i32 y;c::int z;u8 f;ptr i32 p;ptr mut void o;}");
    let usize s=struct_at(ref p,0);let repr_c::Info info=repr_c::read(ref p,s);
    assert(info.repr_c && info.code==0);
    assert(bytes::equal(syntax::source_view(ref p)[info.start..info.end],b"@repr(C)"));
    assert(repr_c::layout(ref p,s));
    let d=syntax::declaration_at(ref p,s);assert(d.field_count==6);
    assert(c_types::resolve(ref p,syntax::field_at(ref p,d.field_start+2).type_root)==132);
    assert(bytes::equal(syntax::type_text(ref p,syntax::field_at(ref p,d.field_start+4).type_root),b"ptr i32"));
    let address=pointers::resolve(ref p,syntax::field_at(ref p,d.field_start+4).type_root);assert(address!=0 && pointers::valid(address));
    let opaque=pointers::resolve(ref p,syntax::field_at(ref p,d.field_start+5).type_root);assert(opaque==1048591 && pointers::mutable(opaque));
}
fn void plain_struct_without_marker_is_not_repr_c(){
    let p=parsed(b"module m;struct Plain{i32 x;}");
    let usize s=struct_at(ref p,0);let repr_c::Info info=repr_c::read(ref p,s);
    assert(!info.repr_c && info.code==0);
    let d=syntax::declaration_at(ref p,s);assert(info.start==d.start && info.end==d.end);
    assert(repr_c::layout(ref p,s));
}
fn void malformed_attributes_are_rejected(){
    assert(struct_code(b"module m;@repr(C)@repr(C)struct S{i32 x;}")==5106);
    assert(struct_code(b"module m;@repr(C,mu)struct S{i32 x;}")==5106);
    assert(struct_code(b"module m;@repr(D)struct S{i32 x;}")==5106);
    assert(struct_code(b"module m;@repr struct S{i32 x;}")==5106);
    assert(struct_code(b"module m;@repr(1)struct S{i32 x;}")==5106);
    assert(struct_code(b"module m;@repr(c::C)struct S{i32 x;}")==5106);
    assert(struct_code(b"module m;@effects(console)struct S{i32 x;}")==5106);
    assert(struct_layout(b"module m;@repr(C,mu)struct S{i32 x;}")); // layout ignores the marker; only read rejects it
}
fn void empty_and_unsupported_value_lanes_are_rejected(){
    let empty=read_struct(b"module m;@repr(C)struct E{}",0);assert(!empty.repr_c && empty.code==5102);
    assert(!struct_layout(b"module m;@repr(C)struct E{}"));
    assert(struct_code(b"module m;@repr(C)struct S{ref i32 r;}")==5102);
    assert(struct_code(b"module m;@repr(C)struct S{Option<i32> o;}")==5102);
    assert(struct_code(b"module m;@repr(C)struct S{i32[3] a;}")==5102);
    let nested=read_struct(b"module m;struct Inner{i32 n;}@repr(C)struct S{i32 n;Inner i;}",1);assert(!nested.repr_c && nested.code==5102);
    let generic=read_struct(b"module m;@repr(C)struct Pair<T>{i32 a;}",0);assert(!generic.repr_c && generic.code==5106);
    assert(struct_code(b"module m;struct Pair<T>{i32 a;}")==0);
}
fn void non_c_identifiers_and_duplicate_fields_are_rejected(){
    assert(struct_code(b"module m;@repr(C)struct S{i32 main;}")==5106);
    assert(struct_code(b"module m;@repr(C)struct S{i32 fy_x;}")==5106);
    assert(struct_code(b"module m;@repr(C)struct main{i32 x;}")==5106);
    assert(struct_code(b"module m;@repr(C)struct S{i32 int32_t;}")==5106);
    assert(struct_code(b"module m;@repr(C)struct S{i32 x;i32 x;}")==5106);
    assert(!struct_layout(b"module m;@repr(C)struct S{i32 x;i32 x;}"));
}
fn void non_struct_declarations_are_ignored(){
    let p=parsed(b"module m;");let info=repr_c::read(ref p,0);assert(!info.repr_c && info.code==0 && !repr_c::layout(ref p,0));
    let q=parsed(b"module m;fn i32 f(){return 0;}");let fn_info=repr_c::read(ref q,1);assert(!fn_info.repr_c && fn_info.code==0);
}
@test fn void repr_c_declaration_native_contract(){
    admitted_marker_and_value_lane();
    plain_struct_without_marker_is_not_repr_c();
    malformed_attributes_are_rejected();
    empty_and_unsupported_value_lanes_are_rejected();
    non_c_identifiers_and_duplicate_fields_are_rejected();
    non_struct_declarations_are_ignored();
}
