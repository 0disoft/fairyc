module tests::reference_places;
import tests::hir_nominal as support;
import mir_lower as lower;
import mir_model as m;
import reference_types as refs;
@test fn void projection_keeps_original_owner(){
 let h=support::retained(b"module main;struct S{i32 n;}fn i32 main(){let S[2] a=[S{.n=7};2];let ref i32 p=ref a[1].n;return *p;}");
 let p=switch(lower::lower(h)){case .ok(p):{p}case .err(_):{assert(false);return;}};
 var usize fields=0;var usize indices=0;var usize i=1;while(i<=m::operation_count(ref p)){let o=m::operation_at(ref p,i);if(o.kind==32){fields+=1;assert(refs::valid(o.type) && refs::inner(o.type)==1);}if(o.kind==33){indices+=1;assert(m::local_at(ref p,o.left).binding!=0 && o.right!=0);}i+=1;}assert(fields==1 && indices==1);
}
@test fn void mutable_pointer_store_has_no_result(){
 let h=support::retained(b"module main;fn void edit(ref mut i32 p){*p=9;}fn i32 main(){var i32 n=7;edit(ref mut n);return n;}");
 let p=switch(lower::lower(h)){case .ok(p):{p}case .err(_):{assert(false);return;}};var usize stores=0;var usize i=1;while(i<=m::operation_count(ref p)){let o=m::operation_at(ref p,i);if(o.kind==30){stores+=1;assert(o.result==0 && refs::mutable(o.owner_type));}i+=1;}assert(stores==1);
}
