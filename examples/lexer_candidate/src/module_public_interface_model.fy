module module_public_interface_model;
import std::bytes as bytes;
import std::vec as v;

// Source-independent, owned public facts. Text/range indices are storage
// coordinates only; encoders publish their referenced values, never indices.
pub struct Text{usize start;usize end;}
pub struct Range{usize first;usize count;}
pub struct Module{Text path;Range items;}
pub struct Field{Text name;Text type;}
pub struct Variant{Text name;Range fields;}
pub struct Item{Text kind;Text name;bool repr_c;bool unsafe_call;bool must_use;Text abi;Text link;Text owner;Range effects;Range abilities;Range parameters;Text result;Range fields;Range variants;}
pub resource Descriptor{bytes::Buffer names;v::Vec<Text> lists;v::Vec<Field> fields;v::Vec<Variant> variants;v::Vec<Item> items;v::Vec<Module> modules;cleanup{}}
pub fn Descriptor assemble(bytes::Buffer names,v::Vec<Text> lists,v::Vec<Field> fields,v::Vec<Variant> variants,v::Vec<Item> items,v::Vec<Module> modules){return Descriptor{.names,.lists,.fields,.variants,.items,.modules};}
pub fn usize text_bytes(ref Descriptor p){return bytes::len(ref p.names);}
pub fn bool valid_text(ref Descriptor p,Text t){return t.start<=t.end && t.end<=text_bytes(p);}
pub fn ref u8[] text(ref Descriptor p,Text t){assert(valid_text(p,t));return bytes::view(ref p.names)[t.start..t.end];}
pub fn bool valid_range(Range r,usize count){return r.first<=count && r.count<=count-r.first;}
pub fn usize module_count(ref Descriptor p){return v::len<Module>(ref p.modules);}
pub fn usize item_count(ref Descriptor p){return v::len<Item>(ref p.items);}
pub fn usize list_count(ref Descriptor p){return v::len<Text>(ref p.lists);}
pub fn usize field_count(ref Descriptor p){return v::len<Field>(ref p.fields);}
pub fn usize variant_count(ref Descriptor p){return v::len<Variant>(ref p.variants);}
pub fn Module module_at(ref Descriptor p,usize id){return *v::at<Module>(ref p.modules,id);}
pub fn Item item_at(ref Descriptor p,usize id){return *v::at<Item>(ref p.items,id);}
pub fn Text list_at(ref Descriptor p,usize id){return *v::at<Text>(ref p.lists,id);}
pub fn Field field_at(ref Descriptor p,usize id){return *v::at<Field>(ref p.fields,id);}
pub fn Variant variant_at(ref Descriptor p,usize id){return *v::at<Variant>(ref p.variants,id);}
