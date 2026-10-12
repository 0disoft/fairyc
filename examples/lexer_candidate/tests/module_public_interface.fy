module tests::module_public_interface;
import module_public_interface as public_interface;
import module_public_interface_model as model;
import module_state_layout as state;
import std::bytes as bytes;
import std::vec as v;
fn model::Text text(ref mut bytes::Buffer names,ref u8[] value){let start=bytes::len(names);assert(bytes::append(names,value).is_ok);return model::Text{.start,.end=bytes::len(names)};}
fn state::Plan layouts(){var bytes::Buffer names=bytes::empty();assert(bytes::append(ref mut names,b"example.empty").is_ok);var v::Vec<state::Layout> rows=v::empty<state::Layout>();assert(v::push<state::Layout>(ref mut rows,state::Layout{.owner=7,.module_index=2,.declaration=13,.start=0,.end=0,.uid_start=0,.uid_end=13,.field_start=0,.field_count=0,.hash=[3782939016,37667965,801448020,270976210,1576068715,3511805801,3605122185,3274707774]}).is_ok);return state::assemble(names,rows,v::empty<state::Field>());}
fn model::Descriptor fixture(u32 mode){var bytes::Buffer names=bytes::empty();if(mode==1){assert(bytes::append(ref mut names,b"private body and local numbering").is_ok);}var v::Vec<model::Text> lists=v::empty<model::Text>();var v::Vec<model::Field> fields=v::empty<model::Field>();var v::Vec<model::Variant> variants=v::empty<model::Variant>();var v::Vec<model::Item> items=v::empty<model::Item>();var v::Vec<model::Module> modules=v::empty<model::Module>();
let kind0=text(ref mut names,b"enum");
let name0=text(ref mut names,b"Kind");
let abi0=text(ref mut names,b"");
let link0=text(ref mut names,b"");
let owner0=text(ref mut names,b"");
let result0=text(ref mut names,b"");
let first_effects0=v::len<model::Text>(ref lists);
let effects0=model::Range{.first=first_effects0,.count=0};
let first_abilities0=v::len<model::Text>(ref lists);
let abilities0=model::Range{.first=first_abilities0,.count=0};
let first_parameters0=v::len<model::Text>(ref lists);
let parameters0=model::Range{.first=first_parameters0,.count=0};
let first_fields0=v::len<model::Field>(ref fields);
let fields0=model::Range{.first=first_fields0,.count=0};
let first_variants0=v::len<model::Variant>(ref variants);
let variant0_0=text(ref mut names,b"Off");
let first_payloadvariant0_0=v::len<model::Field>(ref fields);
let payloadvariant0_0=model::Range{.first=first_payloadvariant0_0,.count=0};
assert(v::push<model::Variant>(ref mut variants,model::Variant{.name=variant0_0,.fields=payloadvariant0_0}).is_ok);
let variant0_1=text(ref mut names,b"Pair");
let first_payloadvariant0_1=v::len<model::Field>(ref fields);
let payloadvariant0_1_n0=text(ref mut names,b"value");
let payloadvariant0_1_t0=text(ref mut names,b"i64");
assert(v::push<model::Field>(ref mut fields,model::Field{.name=payloadvariant0_1_n0,.type=payloadvariant0_1_t0}).is_ok);
let payloadvariant0_1_n1=text(ref mut names,b"node");
let payloadvariant0_1_t1=text(ref mut names,b"self::alpha::Packet");
assert(v::push<model::Field>(ref mut fields,model::Field{.name=payloadvariant0_1_n1,.type=payloadvariant0_1_t1}).is_ok);
let payloadvariant0_1=model::Range{.first=first_payloadvariant0_1,.count=2};
assert(v::push<model::Variant>(ref mut variants,model::Variant{.name=variant0_1,.fields=payloadvariant0_1}).is_ok);
let variants0=model::Range{.first=first_variants0,.count=2};
var item0=model::Item{.kind=kind0,.name=name0,.abi=abi0,.link=link0,.owner=owner0,.result=result0,.effects=effects0,.abilities=abilities0,.parameters=parameters0,.fields=fields0,.variants=variants0,.repr_c=false,.unsafe_call=false,.must_use=false};
if(mode==12){item0.name.end=1048577;}
assert(v::push<model::Item>(ref mut items,item0).is_ok);
let kind1=text(ref mut names,b"foreign-function");
let name1=text(ref mut names,b"packet");
let abi1=text(ref mut names,b"C");
let link1=text(ref mut names,b"fixture_packet");
let owner1=text(ref mut names,b"");
let result1=text(ref mut names,b"self::alpha::Packet");
let first_effects1=v::len<model::Text>(ref lists);
let effects1_0=text(ref mut names,b"console");assert(v::push<model::Text>(ref mut lists,effects1_0).is_ok);
let effects1_1=text(ref mut names,b"filesystem");assert(v::push<model::Text>(ref mut lists,effects1_1).is_ok);
let effects1=model::Range{.first=first_effects1,.count=2};
let first_abilities1=v::len<model::Text>(ref lists);
let abilities1=model::Range{.first=first_abilities1,.count=0};
let first_parameters1=v::len<model::Text>(ref lists);
let parameters1_0=text(ref mut names,b"c::uint");assert(v::push<model::Text>(ref mut lists,parameters1_0).is_ok);
let parameters1_1=text(ref mut names,b"ptr mut void");assert(v::push<model::Text>(ref mut lists,parameters1_1).is_ok);
let parameters1=model::Range{.first=first_parameters1,.count=2};
let first_fields1=v::len<model::Field>(ref fields);
let fields1=model::Range{.first=first_fields1,.count=0};
let first_variants1=v::len<model::Variant>(ref variants);
let variants1=model::Range{.first=first_variants1,.count=0};
var item1=model::Item{.kind=kind1,.name=name1,.abi=abi1,.link=link1,.owner=owner1,.result=result1,.effects=effects1,.abilities=abilities1,.parameters=parameters1,.fields=fields1,.variants=variants1,.repr_c=false,.unsafe_call=true,.must_use=true};
if(mode==8){item1.abilities=model::Range{.first=0,.count=1};}
assert(v::push<model::Item>(ref mut items,item1).is_ok);
let kind2=text(ref mut names,b"function");
let name2=text(ref mut names,b"view");
let abi2=text(ref mut names,b"");
let link2=text(ref mut names,b"");
let owner2=text(ref mut names,b"self::alpha::Owner");
let result2=text(ref mut names,b"Option<ref self::alpha::Packet>");
let first_effects2=v::len<model::Text>(ref lists);
let effects2_0=text(ref mut names,b"clock");assert(v::push<model::Text>(ref mut lists,effects2_0).is_ok);
let effects2_1=text(ref mut names,b"console");assert(v::push<model::Text>(ref mut lists,effects2_1).is_ok);
let effects2=model::Range{.first=first_effects2,.count=2};
let first_abilities2=v::len<model::Text>(ref lists);
let abilities2=model::Range{.first=first_abilities2,.count=0};
let first_parameters2=v::len<model::Text>(ref lists);
let parameters2_0=text(ref mut names,b"ref mut self::alpha::Owner");assert(v::push<model::Text>(ref mut lists,parameters2_0).is_ok);
let parameters2_1=text(ref mut names,b"Result<i64,u32>");assert(v::push<model::Text>(ref mut lists,parameters2_1).is_ok);
let parameters2_2=text(ref mut names,b"u8[]");assert(v::push<model::Text>(ref mut lists,parameters2_2).is_ok);
let parameters2=model::Range{.first=first_parameters2,.count=3};
let first_fields2=v::len<model::Field>(ref fields);
let fields2=model::Range{.first=first_fields2,.count=0};
let first_variants2=v::len<model::Variant>(ref variants);
let variants2=model::Range{.first=first_variants2,.count=0};
var item2=model::Item{.kind=kind2,.name=name2,.abi=abi2,.link=link2,.owner=owner2,.result=result2,.effects=effects2,.abilities=abilities2,.parameters=parameters2,.fields=fields2,.variants=variants2,.repr_c=false,.unsafe_call=true,.must_use=true};
if(mode==2){item2.must_use=false;}if(mode==5){item2.parameters.first=32768;}
assert(v::push<model::Item>(ref mut items,item2).is_ok);
let kind3=text(ref mut names,b"resource");
let name3=text(ref mut names,b"Owner");
let abi3=text(ref mut names,b"");
let link3=text(ref mut names,b"");
let owner3=text(ref mut names,b"");
let result3=text(ref mut names,b"");
let first_effects3=v::len<model::Text>(ref lists);
let effects3_0=text(ref mut names,b"console");assert(v::push<model::Text>(ref mut lists,effects3_0).is_ok);
let effects3_1=text(ref mut names,b"filesystem");assert(v::push<model::Text>(ref mut lists,effects3_1).is_ok);
let effects3=model::Range{.first=first_effects3,.count=2};
let first_abilities3=v::len<model::Text>(ref lists);
let abilities3_0=text(ref mut names,b"drop");assert(v::push<model::Text>(ref mut lists,abilities3_0).is_ok);
let abilities3_1=text(ref mut names,if(mode==6){b"key"}else{b"store"});assert(v::push<model::Text>(ref mut lists,abilities3_1).is_ok);
let abilities3_2=text(ref mut names,if(mode==6){b"store"}else{b"key"});assert(v::push<model::Text>(ref mut lists,abilities3_2).is_ok);
let abilities3=model::Range{.first=first_abilities3,.count=3};
let first_parameters3=v::len<model::Text>(ref lists);
let parameters3=model::Range{.first=first_parameters3,.count=0};
let first_fields3=v::len<model::Field>(ref fields);
let fields3=model::Range{.first=first_fields3,.count=0};
let first_variants3=v::len<model::Variant>(ref variants);
let variants3=model::Range{.first=first_variants3,.count=0};
var item3=model::Item{.kind=kind3,.name=name3,.abi=abi3,.link=link3,.owner=owner3,.result=result3,.effects=effects3,.abilities=abilities3,.parameters=parameters3,.fields=fields3,.variants=variants3,.repr_c=false,.unsafe_call=false,.must_use=false};
if(mode==4){item3.kind=kind2;item3.name=name2;item3.abilities=model::Range{.first=0,.count=0};}if(mode==7){item3.abilities.count=0;}if(mode==9){item3.abilities.first=first_abilities3+2;item3.abilities.count=1;}
assert(v::push<model::Item>(ref mut items,item3).is_ok);
let kind4=text(ref mut names,b"struct");
let name4=text(ref mut names,b"Packet");
let abi4=text(ref mut names,b"");
let link4=text(ref mut names,b"");
let owner4=text(ref mut names,b"");
let result4=text(ref mut names,b"");
let first_effects4=v::len<model::Text>(ref lists);
let effects4=model::Range{.first=first_effects4,.count=0};
let first_abilities4=v::len<model::Text>(ref lists);
let abilities4=model::Range{.first=first_abilities4,.count=0};
let first_parameters4=v::len<model::Text>(ref lists);
let parameters4=model::Range{.first=first_parameters4,.count=0};
let first_fields4=v::len<model::Field>(ref fields);
let fields4_n0=text(ref mut names,b"tag");
let fields4_t0=text(ref mut names,b"u32");
assert(v::push<model::Field>(ref mut fields,model::Field{.name=fields4_n0,.type=fields4_t0}).is_ok);
let fields4_n1=text(ref mut names,b"payload");
let fields4_t1=text(ref mut names,b"u8[16]");
assert(v::push<model::Field>(ref mut fields,model::Field{.name=fields4_n1,.type=fields4_t1}).is_ok);
let fields4=model::Range{.first=first_fields4,.count=2};
let first_variants4=v::len<model::Variant>(ref variants);
let variants4=model::Range{.first=first_variants4,.count=0};
var item4=model::Item{.kind=kind4,.name=name4,.abi=abi4,.link=link4,.owner=owner4,.result=result4,.effects=effects4,.abilities=abilities4,.parameters=parameters4,.fields=fields4,.variants=variants4,.repr_c=true,.unsafe_call=false,.must_use=false};
assert(v::push<model::Item>(ref mut items,item4).is_ok);
let alpha=text(ref mut names,if(mode==3){b"zeta"}else{b"alpha"});let zeta=text(ref mut names,if(mode==3 || mode==10){b"alpha"}else{b"zeta"});if(mode!=11){assert(v::push<model::Module>(ref mut modules,model::Module{.path=alpha,.items=model::Range{.first=0,.count=5}}).is_ok);assert(v::push<model::Module>(ref mut modules,model::Module{.path=zeta,.items=model::Range{.first=5,.count=0}}).is_ok);}return model::assemble(names,lists,fields,variants,items,modules);}
fn bytes::Buffer encoded(ref state::Plan layouts,ref model::Descriptor p){var usize work=0;return switch(public_interface::encode(layouts,7,p,ref mut work)){case .ok(b):{b}case .err(_):{assert(false);return encoded(layouts,p);}};}
fn bytes::Buffer hash(ref state::Plan layouts,ref model::Descriptor p){var usize work=0;return switch(public_interface::hash(layouts,7,p,ref mut work)){case .ok(b):{b}case .err(_):{assert(false);return hash(layouts,p);}};}
@test
fn void complete_v5_bytes_and_hash_match_actual_go_descriptor(){let layouts=layouts();let p=fixture(0);let b=encoded(ref layouts,ref p);assert(bytes::equal(bytes::view(ref b),golden()));let h=hash(ref layouts,ref p);assert(bytes::equal(bytes::view(ref h),golden_hash()));let no_state=state::empty();let no=encoded(ref no_state,ref p);assert(bytes::equal(bytes::view(ref no),without_state()));let nh=hash(ref no_state,ref p);assert(bytes::equal(bytes::view(ref nh),without_state_hash()));}
@test
fn void public_descriptor_storage_coordinates_are_not_interface_facts(){let layouts=layouts();let p=fixture(0);let shifted=fixture(1);let before=hash(ref layouts,ref p);let after=hash(ref layouts,ref shifted);assert(bytes::equal(bytes::view(ref before),bytes::view(ref after)));let changed=fixture(2);let different=hash(ref layouts,ref changed);assert(!bytes::equal(bytes::view(ref before),bytes::view(ref different)));}
@test
fn void interface_rejects_unsorted_duplicate_and_malformed_public_descriptors(){let layouts=layouts();var u32 mode=3;while(mode<=12){let p=fixture(mode);var usize work=0;assert(switch(public_interface::encode(ref layouts,7,ref p,ref mut work)){case .ok(_):{false}case .err(code):{code==9}});mode+=1;}}
@test
fn void interface_hash_honors_the_complete_shared_work_budget(){let layouts=layouts();let p=fixture(0);var usize cost=0;assert(switch(public_interface::hash(ref layouts,7,ref p,ref mut cost)){case .ok(_):{true}case .err(_):{false}});assert(cost>0 && cost<1048576);var usize work=1048576-cost;assert(switch(public_interface::hash(ref layouts,7,ref p,ref mut work)){case .ok(_):{true}case .err(_):{false}});assert(work==1048576);work=1048576-cost+1;assert(switch(public_interface::hash(ref layouts,7,ref p,ref mut work)){case .ok(_):{false}case .err(code):{code==10}});assert(work<=1048576);}

fn ref u8[] golden(){return b"\x66\x61\x69\x72\x79\x63\x2e\x70\x75\x62\x6c\x69\x63\x2d\x69\x6e\x74\x65\x72\x66\x61\x63\x65\x00\x05\x00\x00\x00\x00\x00\x00\x00\x01\x00\x00\x00\x00\x00\x00\x00\x0d\x65\x78\x61\x6d\x70\x6c\x65\x2e\x65\x6d\x70\x74\x79\xe1\x7b\x11\x88\x02\x3e\xc4\x7d\x2f\xc5\x20\x54\x10\x26\xc4\xd2\x5d\xf0\xe6\x6b\xd1\x51\xe7\x69\xd6\xe1\xcc\x89\xc3\x30\x13\x3e\x00\x00\x00\x00\x00\x00\x00\x02\x00\x00\x00\x00\x00\x00\x00\x05\x61\x6c\x70\x68\x61\x00\x00\x00\x00\x00\x00\x00\x05\x00\x00\x00\x00\x00\x00\x00\x04\x65\x6e\x75\x6d\x00\x00\x00\x00\x00\x00\x00\x04\x4b\x69\x6e\x64\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x02\x00\x00\x00\x00\x00\x00\x00\x03\x4f\x66\x66\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x04\x50\x61\x69\x72\x00\x00\x00\x00\x00\x00\x00\x02\x00\x00\x00\x00\x00\x00\x00\x05\x76\x61\x6c\x75\x65\x00\x00\x00\x00\x00\x00\x00\x03\x69\x36\x34\x00\x00\x00\x00\x00\x00\x00\x04\x6e\x6f\x64\x65\x00\x00\x00\x00\x00\x00\x00\x13\x73\x65\x6c\x66\x3a\x3a\x61\x6c\x70\x68\x61\x3a\x3a\x50\x61\x63\x6b\x65\x74\x00\x00\x00\x00\x00\x00\x00\x10\x66\x6f\x72\x65\x69\x67\x6e\x2d\x66\x75\x6e\x63\x74\x69\x6f\x6e\x00\x00\x00\x00\x00\x00\x00\x06\x70\x61\x63\x6b\x65\x74\x00\x01\x01\x00\x00\x00\x00\x00\x00\x00\x01\x43\x00\x00\x00\x00\x00\x00\x00\x0e\x66\x69\x78\x74\x75\x72\x65\x5f\x70\x61\x63\x6b\x65\x74\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x02\x00\x00\x00\x00\x00\x00\x00\x07\x63\x6f\x6e\x73\x6f\x6c\x65\x00\x00\x00\x00\x00\x00\x00\x0a\x66\x69\x6c\x65\x73\x79\x73\x74\x65\x6d\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x02\x00\x00\x00\x00\x00\x00\x00\x07\x63\x3a\x3a\x75\x69\x6e\x74\x00\x00\x00\x00\x00\x00\x00\x0c\x70\x74\x72\x20\x6d\x75\x74\x20\x76\x6f\x69\x64\x00\x00\x00\x00\x00\x00\x00\x13\x73\x65\x6c\x66\x3a\x3a\x61\x6c\x70\x68\x61\x3a\x3a\x50\x61\x63\x6b\x65\x74\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x08\x66\x75\x6e\x63\x74\x69\x6f\x6e\x00\x00\x00\x00\x00\x00\x00\x04\x76\x69\x65\x77\x00\x01\x01\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x12\x73\x65\x6c\x66\x3a\x3a\x61\x6c\x70\x68\x61\x3a\x3a\x4f\x77\x6e\x65\x72\x00\x00\x00\x00\x00\x00\x00\x02\x00\x00\x00\x00\x00\x00\x00\x05\x63\x6c\x6f\x63\x6b\x00\x00\x00\x00\x00\x00\x00\x07\x63\x6f\x6e\x73\x6f\x6c\x65\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x03\x00\x00\x00\x00\x00\x00\x00\x1a\x72\x65\x66\x20\x6d\x75\x74\x20\x73\x65\x6c\x66\x3a\x3a\x61\x6c\x70\x68\x61\x3a\x3a\x4f\x77\x6e\x65\x72\x00\x00\x00\x00\x00\x00\x00\x0f\x52\x65\x73\x75\x6c\x74\x3c\x69\x36\x34\x2c\x75\x33\x32\x3e\x00\x00\x00\x00\x00\x00\x00\x04\x75\x38\x5b\x5d\x00\x00\x00\x00\x00\x00\x00\x1f\x4f\x70\x74\x69\x6f\x6e\x3c\x72\x65\x66\x20\x73\x65\x6c\x66\x3a\x3a\x61\x6c\x70\x68\x61\x3a\x3a\x50\x61\x63\x6b\x65\x74\x3e\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x08\x72\x65\x73\x6f\x75\x72\x63\x65\x00\x00\x00\x00\x00\x00\x00\x05\x4f\x77\x6e\x65\x72\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x02\x00\x00\x00\x00\x00\x00\x00\x07\x63\x6f\x6e\x73\x6f\x6c\x65\x00\x00\x00\x00\x00\x00\x00\x0a\x66\x69\x6c\x65\x73\x79\x73\x74\x65\x6d\x00\x00\x00\x00\x00\x00\x00\x03\x00\x00\x00\x00\x00\x00\x00\x04\x64\x72\x6f\x70\x00\x00\x00\x00\x00\x00\x00\x05\x73\x74\x6f\x72\x65\x00\x00\x00\x00\x00\x00\x00\x03\x6b\x65\x79\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x06\x73\x74\x72\x75\x63\x74\x00\x00\x00\x00\x00\x00\x00\x06\x50\x61\x63\x6b\x65\x74\x01\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x02\x00\x00\x00\x00\x00\x00\x00\x03\x74\x61\x67\x00\x00\x00\x00\x00\x00\x00\x03\x75\x33\x32\x00\x00\x00\x00\x00\x00\x00\x07\x70\x61\x79\x6c\x6f\x61\x64\x00\x00\x00\x00\x00\x00\x00\x06\x75\x38\x5b\x31\x36\x5d\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x04\x7a\x65\x74\x61\x00\x00\x00\x00\x00\x00\x00\x00";}
fn ref u8[] without_state(){return b"\x66\x61\x69\x72\x79\x63\x2e\x70\x75\x62\x6c\x69\x63\x2d\x69\x6e\x74\x65\x72\x66\x61\x63\x65\x00\x05\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x02\x00\x00\x00\x00\x00\x00\x00\x05\x61\x6c\x70\x68\x61\x00\x00\x00\x00\x00\x00\x00\x05\x00\x00\x00\x00\x00\x00\x00\x04\x65\x6e\x75\x6d\x00\x00\x00\x00\x00\x00\x00\x04\x4b\x69\x6e\x64\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x02\x00\x00\x00\x00\x00\x00\x00\x03\x4f\x66\x66\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x04\x50\x61\x69\x72\x00\x00\x00\x00\x00\x00\x00\x02\x00\x00\x00\x00\x00\x00\x00\x05\x76\x61\x6c\x75\x65\x00\x00\x00\x00\x00\x00\x00\x03\x69\x36\x34\x00\x00\x00\x00\x00\x00\x00\x04\x6e\x6f\x64\x65\x00\x00\x00\x00\x00\x00\x00\x13\x73\x65\x6c\x66\x3a\x3a\x61\x6c\x70\x68\x61\x3a\x3a\x50\x61\x63\x6b\x65\x74\x00\x00\x00\x00\x00\x00\x00\x10\x66\x6f\x72\x65\x69\x67\x6e\x2d\x66\x75\x6e\x63\x74\x69\x6f\x6e\x00\x00\x00\x00\x00\x00\x00\x06\x70\x61\x63\x6b\x65\x74\x00\x01\x01\x00\x00\x00\x00\x00\x00\x00\x01\x43\x00\x00\x00\x00\x00\x00\x00\x0e\x66\x69\x78\x74\x75\x72\x65\x5f\x70\x61\x63\x6b\x65\x74\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x02\x00\x00\x00\x00\x00\x00\x00\x07\x63\x6f\x6e\x73\x6f\x6c\x65\x00\x00\x00\x00\x00\x00\x00\x0a\x66\x69\x6c\x65\x73\x79\x73\x74\x65\x6d\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x02\x00\x00\x00\x00\x00\x00\x00\x07\x63\x3a\x3a\x75\x69\x6e\x74\x00\x00\x00\x00\x00\x00\x00\x0c\x70\x74\x72\x20\x6d\x75\x74\x20\x76\x6f\x69\x64\x00\x00\x00\x00\x00\x00\x00\x13\x73\x65\x6c\x66\x3a\x3a\x61\x6c\x70\x68\x61\x3a\x3a\x50\x61\x63\x6b\x65\x74\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x08\x66\x75\x6e\x63\x74\x69\x6f\x6e\x00\x00\x00\x00\x00\x00\x00\x04\x76\x69\x65\x77\x00\x01\x01\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x12\x73\x65\x6c\x66\x3a\x3a\x61\x6c\x70\x68\x61\x3a\x3a\x4f\x77\x6e\x65\x72\x00\x00\x00\x00\x00\x00\x00\x02\x00\x00\x00\x00\x00\x00\x00\x05\x63\x6c\x6f\x63\x6b\x00\x00\x00\x00\x00\x00\x00\x07\x63\x6f\x6e\x73\x6f\x6c\x65\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x03\x00\x00\x00\x00\x00\x00\x00\x1a\x72\x65\x66\x20\x6d\x75\x74\x20\x73\x65\x6c\x66\x3a\x3a\x61\x6c\x70\x68\x61\x3a\x3a\x4f\x77\x6e\x65\x72\x00\x00\x00\x00\x00\x00\x00\x0f\x52\x65\x73\x75\x6c\x74\x3c\x69\x36\x34\x2c\x75\x33\x32\x3e\x00\x00\x00\x00\x00\x00\x00\x04\x75\x38\x5b\x5d\x00\x00\x00\x00\x00\x00\x00\x1f\x4f\x70\x74\x69\x6f\x6e\x3c\x72\x65\x66\x20\x73\x65\x6c\x66\x3a\x3a\x61\x6c\x70\x68\x61\x3a\x3a\x50\x61\x63\x6b\x65\x74\x3e\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x08\x72\x65\x73\x6f\x75\x72\x63\x65\x00\x00\x00\x00\x00\x00\x00\x05\x4f\x77\x6e\x65\x72\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x02\x00\x00\x00\x00\x00\x00\x00\x07\x63\x6f\x6e\x73\x6f\x6c\x65\x00\x00\x00\x00\x00\x00\x00\x0a\x66\x69\x6c\x65\x73\x79\x73\x74\x65\x6d\x00\x00\x00\x00\x00\x00\x00\x03\x00\x00\x00\x00\x00\x00\x00\x04\x64\x72\x6f\x70\x00\x00\x00\x00\x00\x00\x00\x05\x73\x74\x6f\x72\x65\x00\x00\x00\x00\x00\x00\x00\x03\x6b\x65\x79\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x06\x73\x74\x72\x75\x63\x74\x00\x00\x00\x00\x00\x00\x00\x06\x50\x61\x63\x6b\x65\x74\x01\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x02\x00\x00\x00\x00\x00\x00\x00\x03\x74\x61\x67\x00\x00\x00\x00\x00\x00\x00\x03\x75\x33\x32\x00\x00\x00\x00\x00\x00\x00\x07\x70\x61\x79\x6c\x6f\x61\x64\x00\x00\x00\x00\x00\x00\x00\x06\x75\x38\x5b\x31\x36\x5d\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x04\x7a\x65\x74\x61\x00\x00\x00\x00\x00\x00\x00\x00";}
fn ref u8[] golden_hash(){return b"fyif5-sha256-fbdbc11691c494534fe5f6e8c5662c3e24a95332be2c367ecca0fcd0f8d2f333";}
fn ref u8[] without_state_hash(){return b"fyif5-sha256-4214d399cc6ebe486c724acec8665c2946973d1ca5d028027f38d51896c7b506";}
