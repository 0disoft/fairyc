module tests::static_operation_declarations;
import static_operation_declarations as operations;
import scanner as scanner;
import syntax as syntax;
import std::bytes as bytes;
fn syntax::Parsed parsed(ref u8[] raw){let b=switch(bytes::copy(raw)){case .ok(p):{p}case .err(_):{assert(false);return parsed(raw);}};let s=switch(scanner::collect(b,false,65536,1048576,256,16777216)){case .ok(p):{p}case .err(_):{assert(false);return parsed(raw);}};return switch(syntax::parse(s,512)){case .ok(p):{assert(syntax::problem_count(ref p)==0);p}case .err(_):{assert(false);parsed(raw)}};}
@test fn void exact_providers_and_missing_tuple(){let p=parsed(b"module root;@interface pub fn T transform<T>(T x){}@implements(transform,i32) fn i32 twice(i32 y){return y*2;}@implements(transform,i64) fn i64 triple(i64 z){return z*3;}");var usize work=0;
assert(operations::operation(ref p,2).code==0);
assert(operations::operation(ref p,2).parameters==1);
assert(operations::provider(ref p,3).kind==2);
assert(operations::provider(ref p,3).scalar==1);
assert(operations::provider(ref p,3).operation==2);
assert(operations::provider(ref p,3).parameters==1);
assert(operations::provider(ref p,3).code==0);
assert(operations::provider(ref p,4).code==0);
assert(operations::validate(ref p,ref mut work)==0);
assert(switch(operations::select(ref p,2,1,ref mut work)){case .ok(id):{id==3}case .err(_):{false}});assert(switch(operations::select(ref p,2,2,ref mut work)){case .ok(id):{id==4}case .err(_):{false}});assert(switch(operations::select(ref p,2,6,ref mut work)){case .ok(_):{false}case .err(code):{code==4502}});work=1048576;assert(operations::validate(ref p,ref mut work)==4503);}
fn void rejects(ref u8[] raw){let p=parsed(raw);var usize work=0;assert(operations::validate(ref p,ref mut work)==4501);}
@test fn void malformed_operation_and_provider_reject(){
rejects(b"module root;@interface(1) fn T op<T>(){}");
rejects(b"module root;@interface @interface fn T op<T>(){}");
rejects(b"module root;@interface unsafe fn T op<T>(){}");
rejects(b"module root;@interface fn T op<T>(T x){return x;}");
rejects(b"module root;@interface fn T op<T>(i32 x){}");
rejects(b"module root;@interface fn T op<T>(T a,T b,T c,T d,T e){}");
rejects(b"module root;@interface fn T op<T>(T x){}@implements(op,i32) fn i64 wrong(i64 x){return x;}");
rejects(b"module root;@implements(other,i32) fn i32 orphan(i32 x){return x;}");
rejects(b"module root;@interface fn T op<T>(T x){}@implements(op,i32) unsafe fn i32 wrong(i32 x){return x;}");
rejects(b"module root;@interface fn T op<T>(T x){}@implements(op,i32) fn i32 a(i32 x){return x;}@implements(op,i32) fn i32 b(i32 x){return x;}");
rejects(b"module root;@interface fn T op<T>(){}@implements(op,void) fn void wrong(){}");
}

@test fn void all_scalar_and_parameter_boundaries(){var usize i=0;while(i<13){let name=if(i==0){b"bool"}else if(i==1){b"u8"}else if(i==2){b"i32"}else if(i==3){b"i64"}else if(i==4){b"u32"}else if(i==5){b"usize"}else if(i==6){b"i8"}else if(i==7){b"i16"}else if(i==8){b"isize"}else if(i==9){b"u16"}else if(i==10){b"u64"}else if(i==11){b"f32"}else{b"f64"};var bytes::Buffer raw=bytes::empty();assert(bytes::append(ref mut raw,b"module root;@interface fn T op<T>(T x){}@implements(op,").is_ok && bytes::append(ref mut raw,name).is_ok && bytes::append(ref mut raw,b") fn ").is_ok && bytes::append(ref mut raw,name).is_ok && bytes::append(ref mut raw,b" pass(").is_ok && bytes::append(ref mut raw,name).is_ok && bytes::append(ref mut raw,b" x){return x;}").is_ok);let p=parsed(bytes::view(ref raw));var usize work=0;assert(operations::validate(ref p,ref mut work)==0 && switch(operations::select(ref p,2,operations::scalar(name),ref mut work)){case .ok(id):{id==3}case .err(_):{false}});i+=1;}
let zero=parsed(b"module root;@interface fn T op<T>(){}@implements(op,i32) fn i32 make(){return 42;}");var usize work=0;assert(operations::validate(ref zero,ref mut work)==0 && operations::operation(ref zero,2).parameters==0);
let four=parsed(b"module root;@interface fn T op<T>(T a,T b,T c,T d){}@implements(op,i32) fn i32 make(i32 a,i32 b,i32 c,i32 d){var i32 sum=a+b;return sum+c+d;}");work=0;assert(operations::validate(ref four,ref mut work)==0 && operations::operation(ref four,2).parameters==4);
}
