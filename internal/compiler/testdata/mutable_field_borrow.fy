module main;
struct Span {i64 start;i64 end;bool ready;u8 byte;u32 word;i32 n;usize size;}
struct Position {Span span;}
struct Token {Position position;i64 sibling;}
fn Token make(){return Token{.position=Position{.span=Span{.start=4000000000,.end=9,.ready=false,.byte=1,.word=2,.n=3,.size=4}},.sibling=7};}
fn void normalize(ref mut Span span){span.start+=2;span.end=12;}
fn ref mut Span project(ref mut Token token){return ref mut token.position.span;}
fn Option<ref mut Span> tagged(ref mut Token token){return .some(project(token));}
fn i64 read(ref Span span){return span.start;}
fn Result<ref mut Span,u32> result(ref mut Token token){return .ok(project(token));}
fn ref mut i64 scalar(ref mut Span span){return ref mut span.start;}
fn i32 main(){
 var Token token=make();var Token copy=token;
 let ref mut Span direct=ref mut ((token).position).span;
 normalize(direct);
 let ref mut i64 value=scalar(direct);let ref mut i64 alias=value;
 *alias=*alias+3;assert(*value==4000000005);
 assert(read(direct)==4000000005);
 direct.ready=true;
 let ref mut u8 byte=ref mut direct.byte;*byte=8;
 let ref mut u32 word=ref mut direct.word;*word=20;
 let ref mut i32 n=ref mut direct.n;*n=30;
 let ref mut usize size=ref mut direct.size;*size=40;
 let ref mut bool ready=ref mut direct.ready;*ready=false;
 assert(direct.start==4000000005 && direct.byte==8 && direct.word==20 && direct.n==30 && direct.size==40 && !direct.ready);
 normalize(project(ref mut token));
 switch(tagged(ref mut token)){case .some(span):{span.end=18;}case .none:{} };
 switch(result(ref mut token)){case .ok(span):{span.end=19;}case .err(_):{assert(false);}};
 assert(token.position.span.start==4000000007 && token.position.span.end==19 && token.sibling==7);
 assert(copy.position.span.start==4000000000 && copy.position.span.end==9);
 let ref mut i64 sibling=ref mut token.sibling;*sibling=11;assert(*sibling==11);
 var Token other=make();var ref mut Token selected=ref mut token;
 let ref mut Span chosen=project(selected);chosen.end=21;
 selected=ref mut other;normalize(project(selected));
 assert(token.position.span.end==21 && other.position.span.start==4000000002);
 for (var i32 i=0;i<2;i+=1){let ref mut Span loop=project(ref mut other);loop.end+=1;}
 assert(other.position.span.end==14);
 return 42;
}
