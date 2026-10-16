module c_abi_types;
import syntax as syntax;
import std::bytes as bytes;
// Semantic identities are independent of target availability. The C emitter
// proves each target ABI before using its spelling; no Fairy literals/operators.
pub fn bool valid(u32 type){return type>=128 && type<=143;}
pub fn u32 named(ref u8[] text){return if(bytes::equal(text,b"c::bool")){128} else if(bytes::equal(text,b"c::char")){129} else if(bytes::equal(text,b"c::double")){130} else if(bytes::equal(text,b"c::float")){131} else if(bytes::equal(text,b"c::int")){132} else if(bytes::equal(text,b"c::long")){133} else if(bytes::equal(text,b"c::longlong")){134} else if(bytes::equal(text,b"c::schar")){135} else if(bytes::equal(text,b"c::short")){136} else if(bytes::equal(text,b"c::size_t")){137} else if(bytes::equal(text,b"c::ssize_t")){138} else if(bytes::equal(text,b"c::uchar")){139} else if(bytes::equal(text,b"c::uint")){140} else if(bytes::equal(text,b"c::ulong")){141} else if(bytes::equal(text,b"c::ulonglong")){142} else if(bytes::equal(text,b"c::ushort")){143} else{0};}
pub fn u32 resolve(ref syntax::Parsed p,usize root){return if(root!=0 && syntax::type_at(p,root).kind==1){named(syntax::type_text(p,root))}else{0};}
pub fn ref u8[] spelling(u32 type){
    assert(valid(type));let ref u8[] names=b"_Bool|char|double|float|int|long|long long|signed char|short|size_t|ssize_t|unsigned char|unsigned int|unsigned long|unsigned long long|unsigned short|";
    var u32 id=128;var usize start=0;while(id<type){while(names[start]!=124){start+=1;}start+=1;id+=1;}
    var usize end=start;while(names[end]!=124){end+=1;}return names[start..end];
}
pub fn bool cast(u32 source,u32 target){return source==1 && target==132 || source==132 && target==1;}

// Exact subset of hir.ValuePreservingIntegerCast; target C ABI proof is unchanged.
pub fn bool allowed(u32 source,u32 target){return cast(source,target) || source==3 && (target==1 || target==2 || target==4 || target==5) || source==1 && target==2 || source==4 && (target==2 || target==5);}
