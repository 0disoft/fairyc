module main;
struct Span { i64 start; i64 end; bool ready; u8 byte; u32 word; usize size; i32 n; }
struct Position { Span span; }
struct Token { Position position; i32 calls; }
fn Span span(i64 start) { return Span { .start=start, .end=start+1, .ready=true, .byte=255, .word=4294967295, .size=9, .n=-2 }; }
fn Token token(i64 start) { return Token { .position=Position { .span=span(start) }, .calls=0 }; }
fn Span next(ref mut Token s) { s.calls+=1; s.position.span.start=99; return span(4000000000); }
fn void edit(ref mut Token s) { s.position.span=next(s); }
fn i32 main() {
 var Token s=token(3);
 var Token copy=s;
 edit(ref mut s);
 assert(s.position.span.start==4000000000 && s.position.span.end==4000000001 && s.calls==1);
 assert(copy.position.span.start==3 && copy.calls==0);
 let Span replacement=span(7);
 ((s).position).span=replacement;
 assert(s.position.span.ready && s.position.span.byte==255 && s.position.span.word==4294967295 && s.position.span.size==9 && s.position.span.n==-2);
 s.position.span.start=8;
 assert(replacement.start==7);
 s.position.span=s.position.span;
 assert(s.position.span.start==8);
 s.position=Position { .span=span(10) };
 copy.position=s.position;
 s.position.span.start=11;
 assert(copy.position.span.start==10);
 s.position={s=token(99);Position { .span=span(12) }};
 assert(s.position.span.start==12 && s.calls==0);
 var Token first=token(20); var Token second=token(30);
 var ref mut Token selected=ref mut first;
 selected.position={selected=ref mut second;Position { .span=span(40) }};
 assert(first.position.span.start==40);
 selected.position.span={selected=ref mut first;span(50)};
 assert(second.position.span.start==50);
 selected.position.span=span(60);
 assert(first.position.span.start==60);
 return 0;
}
