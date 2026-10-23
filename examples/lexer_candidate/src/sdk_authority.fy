module sdk_authority;
import sha256 as hash;
import std::bytes as bytes;

// Source pins select the exact bundled language/20, runtime/0 SDK bytes.
// Names, claimed digests and user AST flags cannot grant storage authority.
// A changed SDK requires an intentional source-table update, not a fallback.
pub fn usize module_id(ref u8[] name){
    return if(bytes::equal(name,b"std::alloc")){1}else if(bytes::equal(name,b"std::bytes")){2}else if(bytes::equal(name,b"std::fs")){3}else if(bytes::equal(name,b"std::io")){4}else if(bytes::equal(name,b"std::num")){5}else if(bytes::equal(name,b"std::process")){6}else if(bytes::equal(name,b"std::text")){7}else if(bytes::equal(name,b"std::vec")){8}else{0};
}
pub fn bool reserved(ref u8[] name){return bytes::equal(name,b"std") || name.len>=5 && bytes::equal(name[..5],b"std::");}
fn bool matches(usize id,ref u8[] source){
    return if(id==1){hash::matches(source,b"61f50fa03239eee7b669c315659e8c9d07982117857cf50fe7f5a2b6f35a82c2")}else if(id==2){hash::matches(source,b"4740a506e93349fb002c383967dcc44d182d20f09f2094b83b5464a4ab252982")}else if(id==3){hash::matches(source,b"243b63642e2c26220322f82905dbc85c35e8b4cdf04556b9aca92be4b4783a2f")}else if(id==4){hash::matches(source,b"4b89c3aeb8b4de4887395d29723095d364c5660c6b773c5696fb1f65f5365f48")}else if(id==5){hash::matches(source,b"09aa0d45f075aa4212049527ded7bb48d99ceb6ffed56b310e267421bae20bb3")}else if(id==6){hash::matches(source,b"9680a1481df4f761f481d5a039af7bdbefeb312d1091961194f8e7bc0681ae24")}else if(id==7){hash::matches(source,b"cccd47e090fda11c9da063ffb321489dd8753fe9996855fc432833e05a79e695")}else if(id==8){hash::matches(source,b"0ae4c395bf98c39079ad620bde035b3b590d51417864802e698edc069281db5b")}else{false};
}
pub fn bool authentic(ref u8[] name,ref u8[] source){let usize id=module_id(name);return id!=0 && matches(id,source);}
// Canonical field transport of the authenticated Vec template. Element identity
// remains separate; no element can change or supply this storage representation.
pub fn u32 vector_field(usize index){return if(index==0){1048591}else if(index<6){5}else{0};}
