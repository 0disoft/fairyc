module main;
struct Span { i32 start; i64 end; }
struct Token { bool active; Span span; }
fn Token make(i32 start){return Token{.active=true,.span=Span{.start=start,.end=4000000000}};}
fn usize count(ref Token[] tokens){var usize n=0;for(let ref Token item in tokens){if(item.active){n+=1;}}return n;}
fn Token[2] produce(){return [make(1),make(2)];}
fn Token[2] copied(Token[2] values){return values;}
fn Option<Token[1]> tagged(){return .some([make(7)]);}
fn ref Token first(ref Token[] values){for(let ref Token item in values){return item;}return ref values[0];}
fn void early(ref mut Token[] values){for(let ref mut Token item in values){item.active=false;return;}}
fn i32 main(){
 var Token[3] data=[make(1),make(2),make(3)];
 assert(data.len==3 && data[1].span.start==2);
 var Token copy=data[0];copy.span.start=9;assert(data[0].span.start==1);
 data[0]=make(4);assert(data[0].span.start==4);
 {let ref mut Token item=ref mut data[0];item.span.start=5;}
 assert(data[0].span.start==5);
 let ref Token[] view=ref data;
 assert(count(view)==3 && count(view[3..])==0 && first(view).span.start==5);
 var usize visits=0;
 for(let Token item in data){assert(item.active);visits+=1;}
 assert(visits==3);
 for(let ref Token item in data){assert(item.span.end==4000000000);}
 visits=0;
 for(let ref mut Token item in data){item.span.start+=1;visits+=1;if(visits==1){continue;}if(visits==2){break;}}
 assert(visits==2 && data[0].span.start==6 && data[1].span.start==3 && data[2].span.start==3);
 let ref mut Token[] edit=ref mut data;
 {let ref mut Token[] child=edit[1..];for(let ref mut Token item in child){item.span.end+=1;}}
 assert(edit[1].span.end==4000000001 && edit[0].span.end==4000000000);
 early(edit);assert(!edit[0].active && edit[1].active);
 visits=0;
 for(let Token before in edit){if(visits==0){edit[1]=make(8);}if(visits==1){assert(before.span.start==8);}visits+=1;}
 let Token template=make(10);
 var Token[2] repeated=[template;2];
 {let ref mut Token item=ref mut repeated[0];item.span.start=99;}
 assert(repeated[1].span.start==10 && template.span.start==10);
 let Token[2] original=produce();var Token[2] independent=copied(original);
 {let ref mut Token item=ref mut independent[0];item.span.start=20;}
 assert(original[0].span.start==1 && independent[0].span.start==20);
 switch(tagged()){case .some(items):{assert(items[0].span.start==7);}case .none:{assert(false);}};
 return 0;
}
