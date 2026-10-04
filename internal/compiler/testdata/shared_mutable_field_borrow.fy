module main;
struct Span {i64 start;i64 end;bool ready;u8 byte;u32 word;i32 n;usize size;}
struct Position {Span span;}
struct Token {Position position;i64 sibling;}
fn Token make(){return Token{.position=Position{.span=Span{.start=4000000000,.end=9,.ready=true,.byte=1,.word=2,.n=3,.size=4}},.sibling=7};}
fn i64 read(ref Span s){return s.start;}
fn ref Span project(ref mut Token t){return ref ((t).position).span;}
fn ref i64 number(ref mut Token t){return ref t.position.span.start;}
fn Option<ref Span> optional(ref mut Token t){return .some(project(t));}
fn Result<ref i64,u32> result(ref mut Token t){return .ok(number(t));}
fn void inspect(ref mut Token t){
 let ref Span span=ref t.position.span;
 let ref i64 start=ref t.position.span.start;
 let ref i64 end=ref span.end;
 assert(read(span)==4000000000 && *start==4000000000 && *end==9);
 assert(t.sibling==7 && t.position.span.start==4000000000);
 let ref bool ready=ref t.position.span.ready;assert(*ready);
 let ref u8 byte=ref t.position.span.byte;assert(*byte==1);
 let ref u32 word=ref t.position.span.word;assert(*word==2);
 let ref i32 n=ref t.position.span.n;assert(*n==3);
 let ref usize size=ref t.position.span.size;assert(*size==4);
 t.position.span.start+=2;
 let ref Span returned=project(t);let ref Span alias=returned;
 assert(read(alias)==4000000002);
 t.position.span.start+=3;
 switch(optional(t)){case .some(s):{assert(s.start==4000000005);}case .none:{assert(false);}};
 t.position.span.start+=1;
 switch(result(t)){case .ok(value):{assert(*value==4000000006);}case .err(_):{assert(false);}};
 t.position.span.end=20;
}
fn i32 main(){
 var Token t=make();var Token copy=t;inspect(ref mut t);
 assert(t.position.span.start==4000000006 && t.position.span.end==20 && copy.position.span.start==4000000000);
 var Token other=make();var ref mut Token selected=ref mut t;
 let ref i64 chosen=number(selected);
 selected=ref mut other;selected.position.span.start+=1;assert(*chosen==4000000006);let ref Span direct=ref selected.position.span;assert(direct.start==4000000001);
 selected.position.span.start+=1;
 let ref mut Span mutable=ref mut selected.position.span;let ref i64 child=ref mutable.start;assert(*child==4000000002);mutable.start+=1;
 assert(other.position.span.start==4000000003);
 for(var i32 i=0;i<2;i+=1){let ref Span loop=project(ref mut other);assert(loop.start==4000000003);}
 other.sibling=11;
 return 42;
}
