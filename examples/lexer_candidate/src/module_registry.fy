module module_registry;
import syntax as syntax;
import scanner as scanner;
import type_state as state;
import std::bytes as bytes;
import std::vec as v;

// One bounded, manifest-ordered source set, without packages or filesystem lookup.
// Names are scanner-normalized so comments/trivia do not change identity.
pub struct Problem { u32 code; usize module_index; usize start; usize end; }
pub struct Module { usize start; usize end; }
pub struct Import { usize module_index; usize alias_start; usize alias_end; usize path_start; usize path_end; usize start; usize end; }
pub struct Item { usize module_index; usize declaration; usize start; usize end; bool public; u32 result; usize parameter_start; usize parameter_count; }
pub struct Lookup { u32 code; usize item; }
pub resource Registry { bytes::Buffer spellings; v::Vec<Module> modules; v::Vec<Import> imports; v::Vec<Item> items; v::Vec<u32> parameters; cleanup { } }
pub fn Registry empty() { return Registry { .spellings = bytes::empty(), .modules = v::empty<Module>(), .imports = v::empty<Import>(), .items = v::empty<Item>(), .parameters = v::empty<u32>() }; }
pub fn Problem ok() { return Problem { .code = 0, .module_index = 0, .start = 0, .end = 0 }; }
fn Problem fail(u32 code, usize module_index, usize start, usize end) { return Problem { .code, .module_index, .start, .end }; }
pub fn usize module_count(ref Registry registry) { return v::len<Module>(ref registry.modules); }
pub fn Item item_at(ref Registry registry, usize index) { return *v::at<Item>(ref registry.items,index); }
pub fn u32 parameter_at(ref Registry registry, usize index) { return *v::at<u32>(ref registry.parameters,index); }

// 11 allocation, 10 limit, 9 unsupported. Input is a parser-accepted path,
// an import declaration, or an actual name expression, not substitute source.
pub fn u32 normalize(ref u8[] source, ref mut bytes::Buffer result) {
    var bytes::Buffer input = bytes::empty();
    if (!bytes::append(ref mut input,source).is_ok) { return 11; }
    let scanner::Stream stream = switch(scanner::collect(input,false,32768,1048576,256,16777216)) {
        case .ok(value): { value } case .err(_): { return 11; }
    };
    if (scanner::diagnostic_count(ref stream) != 0) { return 9; }
    var usize index = 0;
    while (index < scanner::token_count(ref stream)) {
        let scanner::TokenInfo token = scanner::token_info(ref stream,index);
        let ref u8[] text = scanner::token_raw(ref stream,index);
        if (token.kind != 2 && token.kind != 1) {
            if (bytes::equal(text,b"as") || bytes::equal(text,b";")) { break; }
            if (!bytes::equal(text,b"module") && !bytes::equal(text,b"import")) {
                if (token.kind != 3 && !bytes::equal(text,b"::")) { return 9; }
                if (!bytes::append(result,text).is_ok) { return 11; }
            }
        }
        index += 1;
    }
    return if (bytes::len(result) == 0) { 9 } else { 0 };
}
fn bool same(ref Registry registry, usize start, usize end, ref u8[] name) { return bytes::equal(bytes::view(ref registry.spellings)[start..end],name); }
fn bool append(ref mut bytes::Buffer spellings, ref u8[] text) {
    if (text.len > 1048576 - bytes::len(spellings)) { return false; }
    return bytes::append(spellings,text).is_ok;
}
fn bool scalar(ref syntax::Parsed parsed, usize root, bool result) {
    if (root == 0 || syntax::type_at(parsed,root).kind != 1) { return false; }
    let ref u8[] text = syntax::type_text(parsed,root);
    return bytes::equal(text,b"i32") || bytes::equal(text,b"i64") || bytes::equal(text,b"u8") || bytes::equal(text,b"u32") || bytes::equal(text,b"usize") || bytes::equal(text,b"bool") || (result && bytes::equal(text,b"void"));
}
fn bool spelling_same(ref bytes::Buffer spellings, usize start, usize end, ref u8[] name) { return bytes::equal(bytes::view(spellings)[start..end],name); }
// Construct locally, then publish one complete owner. No partial registry escapes.
pub fn Result<Registry,Problem> build(ref v::Vec<syntax::Parsed> sources) {
    var bytes::Buffer spellings = bytes::empty();
    var v::Vec<Module> module_entries = v::empty<Module>(); var v::Vec<Import> import_entries = v::empty<Import>();
    var v::Vec<Item> item_entries = v::empty<Item>(); var v::Vec<u32> parameters = v::empty<u32>();
    var usize module_index = 0;
    while (module_index < v::len<syntax::Parsed>(sources)) {
    let ref syntax::Parsed parsed = v::at<syntax::Parsed>(sources,module_index);
    if (module_index >= 32) { return .err(fail(10,module_index,0,0)); }
    if (syntax::problem_count(parsed) != 0 || syntax::declaration_count(parsed) == 0) { return .err(fail(9,module_index,0,0)); }
    let syntax::Declaration first = syntax::declaration_at(parsed,0);
    if (first.kind != 1) { return .err(fail(9,module_index,first.start,first.end)); }
    let ref u8[] source = syntax::source_view(parsed);
    var bytes::Buffer path = bytes::empty();
    let u32 normalized = normalize(source[first.name_start..first.name_end],ref mut path);
    if (normalized != 0) { return .err(fail(normalized,module_index,first.name_start,first.name_end)); }
    var usize index = 0;
    while (index < module_index) {
        let Module previous = *v::at<Module>(ref module_entries,index);
        if (spelling_same(ref spellings,previous.start,previous.end,bytes::view(ref path))) { return .err(fail(1,module_index,first.name_start,first.name_end)); }
        index += 1;
    }
    let usize start = bytes::len(ref spellings);
    if (!append(ref mut spellings,bytes::view(ref path)) || !v::push<Module>(ref mut module_entries,Module { .start, .end = bytes::len(ref spellings) }).is_ok) { return .err(fail(11,module_index,first.start,first.end)); }
    index = 1; var usize function_count = 0; var usize import_count = 0;
    while (index < syntax::declaration_count(parsed)) {
        let syntax::Declaration declaration = syntax::declaration_at(parsed,index);
        let ref u8[] name = source[declaration.name_start..declaration.name_end];
        if (declaration.kind != 2 && declaration.kind != 3) { return .err(fail(9,module_index,declaration.start,declaration.end)); }
        if (bytes::equal(name,b"source_location")) { return .err(fail(9,module_index,declaration.name_start,declaration.name_end)); }
        var usize previous = 0;
        while (previous < v::len<Import>(ref import_entries)) {
            let Import entry = *v::at<Import>(ref import_entries,previous);
            if (entry.module_index == module_index && spelling_same(ref spellings,entry.alias_start,entry.alias_end,name)) { return .err(fail(if (declaration.kind == 2) { 7 } else { 2 },module_index,declaration.name_start,declaration.name_end)); }
            previous += 1;
        }
        previous = 0;
        while (previous < v::len<Item>(ref item_entries)) {
            let Item entry = *v::at<Item>(ref item_entries,previous);
            if (entry.module_index == module_index && spelling_same(ref spellings,entry.start,entry.end,name)) { return .err(fail(2,module_index,declaration.name_start,declaration.name_end)); }
            previous += 1;
        }
        let usize name_start = bytes::len(ref spellings);
        if (!append(ref mut spellings,name)) { return .err(fail(11,module_index,declaration.name_start,declaration.name_end)); }
        let usize name_end = bytes::len(ref spellings);
        if (declaration.kind == 2) {
            import_count += 1;
            if (import_count > 128 || declaration.public || declaration.attribute_count != 0) { return .err(fail(if (import_count > 128) { 10 } else { 9 },module_index,declaration.start,declaration.end)); }
            var bytes::Buffer target = bytes::empty();
            let u32 status = normalize(source[declaration.start..declaration.end],ref mut target);
            if (status != 0) { return .err(fail(status,module_index,declaration.start,declaration.end)); }
            let usize path_start = bytes::len(ref spellings);
            if (!append(ref mut spellings,bytes::view(ref target))) { return .err(fail(11,module_index,declaration.start,declaration.end)); }
            let Import entry = Import { .module_index, .alias_start = name_start, .alias_end = name_end, .path_start, .path_end = bytes::len(ref spellings), .start = declaration.start, .end = declaration.end };
            if (!v::push<Import>(ref mut import_entries,entry).is_ok) { return .err(fail(11,module_index,declaration.start,declaration.end)); }
        } else {
            function_count += 1;
            if (function_count > 128) { return .err(fail(10,module_index,declaration.start,declaration.end)); }
            if (declaration.owner_end != 0 || declaration.unsafe_end != 0 || declaration.attribute_count != 0 || declaration.generic_parameters != 0 || !scalar(parsed,declaration.result_type,true)) { return .err(fail(9,module_index,declaration.start,declaration.end)); }
            let usize parameter_start = v::len<u32>(ref parameters);
            var usize e = declaration.entry_start;
            while (e < declaration.entry_start + declaration.entry_count) {
                let syntax::Entry parameter = syntax::entry_at(parsed,e);
                if (parameter.kind == 1) {
                    if (!scalar(parsed,parameter.type_root,false)) { return .err(fail(9,module_index,parameter.type_start,parameter.type_end)); }
                    if (!v::push<u32>(ref mut parameters,state::primitive(parsed,parameter.type_root)).is_ok) { return .err(fail(11,module_index,parameter.start,parameter.end)); }
                }
                e += 1;
            }
            let usize parameter_count = v::len<u32>(ref parameters) - parameter_start;
            if (parameter_count > 128) { return .err(fail(10,module_index,declaration.start,declaration.end)); }
            let Item entry = Item { .module_index, .declaration = index + 1, .start = name_start, .end = name_end, .public = declaration.public, .result = state::primitive(parsed,declaration.result_type), .parameter_start, .parameter_count };
            if (!v::push<Item>(ref mut item_entries,entry).is_ok) { return .err(fail(11,module_index,declaration.start,declaration.end)); }
        }
        index += 1;
    }
    module_index += 1;
    }
    let Registry registry = Registry { .spellings, .modules = module_entries, .imports = import_entries, .items = item_entries, .parameters };
    let Problem problem = validate(ref registry);
    if (problem.code != 0) { return .err(problem); }
    return .ok(registry);
}
fn usize target(ref Registry registry, Import entry) {
    let ref u8[] path = bytes::view(ref registry.spellings)[entry.path_start..entry.path_end];
    var usize index = 0;
    while (index < module_count(registry)) { let Module module_index = *v::at<Module>(ref registry.modules,index); if (same(registry,module_index.start,module_index.end,path)) { return index + 1; } index += 1; }
    return 0;
}
pub fn Problem validate(ref Registry registry) {
    var usize index = 0;
    while (index < v::len<Import>(ref registry.imports)) { let Import entry = *v::at<Import>(ref registry.imports,index); if (target(registry,entry) == 0) { return fail(3,entry.module_index,entry.start,entry.end); } index += 1; }
    return ok();
}
// Lookup IDs are one-based. Only an explicit alias::item is source-visible.
pub fn Lookup lookup(ref Registry registry, usize module_index, ref u8[] qualified) {
    var bytes::Buffer normalized = bytes::empty();
    let u32 status = normalize(qualified,ref mut normalized);
    if (status != 0) { return Lookup { .code = status, .item = 0 }; }
    let ref u8[] name = bytes::view(ref normalized);
    var usize split = 0;
    while (split < name.len && name[split] != 58) { split += 1; }
    if (split == name.len || split + 2 >= name.len) { return Lookup { .code = 9, .item = 0 }; }
    var usize index = 0; var usize destination = 0;
    while (index < v::len<Import>(ref registry.imports)) { let Import entry = *v::at<Import>(ref registry.imports,index); if (entry.module_index == module_index && same(registry,entry.alias_start,entry.alias_end,name[..split])) { destination = target(registry,entry); break; } index += 1; }
    if (destination == 0) { return Lookup { .code = 4, .item = 0 }; }
    index = 0;
    while (index < v::len<Item>(ref registry.items)) { let Item entry = item_at(registry,index); if (entry.module_index == destination - 1 && same(registry,entry.start,entry.end,name[split + 2..])) { return Lookup { .code = if (entry.public) { 0 } else { 6 }, .item = index + 1 }; } index += 1; }
    return Lookup { .code = 5, .item = 0 };
}
