module mir_model;
import array_types as arrays;
import literal_types as literals;
import std::bytes as bytes;
import std::vec as v;
import hir_shapes as shapes;

// Private mutable-local scalar MIR. IDs are one-based; zero is absent.
// Ranges and edge/signature indexes are zero-based. No HIR tree is retained.
// Scalar type tags match hir_model: 1 i32, 2 i64, 3 u8, 4 u32, 5 usize,
// 6 bool, 7 void. Operators match hir_model's exhaustive operator_code.
pub struct Problem { u32 code; usize start; usize end; }
pub struct Signature { u32 type; }
pub struct Target { usize binding; usize module_index; usize declaration; u32 result; usize signature_start; usize signature_count; usize start; usize end; bool external; }
pub struct Function { usize target; usize entry; usize local_start; usize local_count; usize block_start; usize block_count; usize parameter_start; usize parameter_count; usize start; usize end; bool public; }
pub struct Local { usize function; usize binding; u32 type; bool parameter; bool mutable; usize start; usize end; }
// Operation kinds: 1 constant, 2 copy/assignment, 3 unary, 4 binary, 5 check,
// 6 construct (declaration-order operands), 7 field read, 8 functional update.
// 9 array construct (boolean means repeat once), 10 indexed read, 11 array
// update (one value edge), 12 checked index, 13 array length. Checked indexes
// freeze usize operands and trap bounds before loads or RHS/store effects.
// 18 shared named u8-array view, 19 checked byte slice (one end-bound edge).
// 14 static literal descriptor, 15 view length/empty (boolean selects empty),
// 16 checked view index (left frozen descriptor, right raw usize), 17 byte read.
// A check defines no local. It must end its block immediately before a matching
// checked terminator. Actual arithmetic occurs only in the success block.
pub struct Operation { usize function; usize block; u32 kind; usize result; u32 type; u32 owner_type; u32 operator; usize left; usize right; usize next; usize payload_start; usize payload_end; bool boolean; usize start; usize end; usize argument_start; usize argument_count; }
// Terminators: 1 return, 2 goto, 3 bool branch, 4 checked switch, 5 call,
// 6 trap, 7 unreachable. Calls have one normal successor, no unwind edge.
// Checked switches name check Operation ID plus frozen local operands.
// Trap mask: 1 integer-overflow, 2 division-by-zero, 4 signed-division-overflow,
// 8 invalid-shift, 16 assertion-failed. Trap blocks have no successor/cleanup.
pub struct Block { usize function; usize first; usize last; u32 kind; usize value; usize left; usize right; u32 operator; u32 type; usize target; usize alternative; usize check; usize callee; usize destination; usize argument_start; usize argument_count; u32 trap_mask; usize start; usize end; }
pub resource Program {
    shapes::Table nominal;
    bytes::Buffer source; bytes::Buffer payload;
    v::Vec<Target> targets; v::Vec<Signature> signatures;
    v::Vec<Function> functions; v::Vec<Local> locals; v::Vec<Block> blocks;
    v::Vec<Operation> operations; v::Vec<usize> edges;
    usize module_index; bool success;
    cleanup { }
}
// Construction caps: 32768 source bytes, 512 functions/targets, 16384 locals,
// 16384 blocks, 32768 operations, 65536 edges/signatures, 688128 payload bytes.
// 1 invalid input, 2 budget, 3 allocation, 4 malformed/unsupported graph.
pub fn usize target_count(ref Program p) { assert(p.success); return v::len<Target>(ref p.targets); }
pub fn usize signature_count(ref Program p) { assert(p.success); return v::len<Signature>(ref p.signatures); }
pub fn usize function_count(ref Program p) { assert(p.success); return v::len<Function>(ref p.functions); }
pub fn usize local_count(ref Program p) { assert(p.success); return v::len<Local>(ref p.locals); }
pub fn usize block_count(ref Program p) { assert(p.success); return v::len<Block>(ref p.blocks); }
pub fn usize operation_count(ref Program p) { assert(p.success); return v::len<Operation>(ref p.operations); }
pub fn usize edge_count(ref Program p) { assert(p.success); return v::len<usize>(ref p.edges); }
pub fn bool has_views(ref Program p) { var usize i=1;while(i<=local_count(p)){if(local_at(p,i).type==11){return true;}i+=1;}i=1;while(i<=operation_count(p)){if(operation_at(p,i).kind>=18){return true;}i+=1;}return false; }
pub fn bool has_literals(ref Program p) { var usize i = 1; while (i <= local_count(p)) { if (literals::valid(local_at(p,i).type)) { return true; } i += 1; } i = 1; while (i <= target_count(p)) { if (literals::valid(target_at(p,i).result)) { return true; } i += 1; } return false; }
pub fn bool has_arrays(ref Program p) {
    var usize i = 1; while (i <= local_count(p)) { if (arrays::valid(local_at(p,i).type)) { return true; } i += 1; }
    i = 1; while (i <= target_count(p)) { if (arrays::valid(target_at(p,i).result)) { return true; } i += 1; }
    i = 1; while (i <= shapes::field_count(nominal(p))) { if (arrays::valid(shapes::field_at(nominal(p),i).type)) { return true; } i += 1; } return false;
}
pub fn ref shapes::Table nominal(ref Program p) { assert(p.success); return ref p.nominal; }
pub fn usize module_index(ref Program p) { assert(p.success); return p.module_index; }
pub fn Target target_at(ref Program p, usize id) { assert(id > 0 && id <= target_count(p)); return *v::at<Target>(ref p.targets,id - 1); }
pub fn Function function_at(ref Program p, usize id) { assert(id > 0 && id <= function_count(p)); return *v::at<Function>(ref p.functions,id - 1); }
pub fn Local local_at(ref Program p, usize id) { assert(id > 0 && id <= local_count(p)); return *v::at<Local>(ref p.locals,id - 1); }
pub fn Block block_at(ref Program p, usize id) { assert(id > 0 && id <= block_count(p)); return *v::at<Block>(ref p.blocks,id - 1); }
pub fn Operation operation_at(ref Program p, usize id) { assert(id > 0 && id <= operation_count(p)); return *v::at<Operation>(ref p.operations,id - 1); }
pub fn u32 signature_at(ref Program p, usize index) { assert(index < signature_count(p)); return v::at<Signature>(ref p.signatures,index).type; }
pub fn usize edge_at(ref Program p, usize index) { assert(index < edge_count(p)); return *v::at<usize>(ref p.edges,index); }
pub fn ref u8[] source_view(ref Program p) { assert(p.success); return bytes::view(ref p.source); }
pub fn ref u8[] target_text(ref Program p, usize id) { let Target t = target_at(p,id); return source_view(p)[t.start..t.end]; }
pub fn ref u8[] literal_text(ref Program p, usize id) { let Operation o = operation_at(p,id); return bytes::view(ref p.payload)[o.payload_start..o.payload_end]; }

// This is a checked construction boundary, not the future complete MIR verifier.
// It rejects unfinished blocks, malformed links and ownership/range drift.
pub fn Result<Program,Problem> assemble(bytes::Buffer source, bytes::Buffer payload, v::Vec<Target> targets, v::Vec<Signature> signatures, v::Vec<Function> functions, v::Vec<Local> locals, v::Vec<Block> blocks, v::Vec<Operation> operations, v::Vec<usize> edges, usize module_index) {
    return assemble_nominal(source,payload,targets,signatures,functions,locals,blocks,operations,edges,module_index,shapes::empty());
}
pub fn Result<Program,Problem> assemble_nominal(bytes::Buffer source, bytes::Buffer payload, v::Vec<Target> targets, v::Vec<Signature> signatures, v::Vec<Function> functions, v::Vec<Local> locals, v::Vec<Block> blocks, v::Vec<Operation> operations, v::Vec<usize> edges, usize module_index, shapes::Table nominal) {
    let usize ns = bytes::len(ref source); let usize np = bytes::len(ref payload);
    let usize nt = v::len<Target>(ref targets); let usize ng = v::len<Signature>(ref signatures); let usize nf = v::len<Function>(ref functions);
    let usize nl = v::len<Local>(ref locals); let usize nb = v::len<Block>(ref blocks); let usize no = v::len<Operation>(ref operations); let usize ne = v::len<usize>(ref edges);
    if (ns > 32768 || np > 688128 || nt > 512 || nf > 512 || nl > 16384 || nb > 16384 || no > 32768 || ne > 65536 || ng > 65536) { return .err(Problem { .code = 2, .start = 0, .end = ns }); }
    if (!shapes::valid(ref nominal,ns,1152)) { return .err(Problem { .code = 4, .start = 0, .end = ns }); }
    var usize i = 0;
    while (i < nt) {
        let Target t = *v::at<Target>(ref targets,i);
        if (t.binding == 0 || t.declaration == 0 || !shapes::valid_type(ref nominal,t.result,true) || t.start >= t.end || t.end > ns || t.signature_start > ng || t.signature_count > ng - t.signature_start) { return .err(Problem { .code = 4, .start = t.start, .end = t.end }); }
        i += 1;
    }
    i = 0; while (i < ng) { let u32 type = v::at<Signature>(ref signatures,i).type; if (!shapes::valid_type(ref nominal,type,false)) { return .err(Problem { .code = 4, .start = 0, .end = 0 }); } i += 1; }
    i = 0;
    while (i < nf) {
        let Function f = *v::at<Function>(ref functions,i);
        if (f.target == 0 || f.target > nt || f.entry == 0 || f.entry > nb || f.local_start > nl || f.local_count > nl - f.local_start || f.block_start > nb || f.block_count > nb - f.block_start || f.parameter_start > ne || f.parameter_count > ne - f.parameter_start || f.start >= f.end || f.end > ns) { return .err(Problem { .code = 4, .start = f.start, .end = f.end }); }
        let Target t = *v::at<Target>(ref targets,f.target - 1);
        if (t.external || t.module_index != module_index || f.entry <= f.block_start || f.entry > f.block_start + f.block_count || t.signature_count != f.parameter_count) { return .err(Problem { .code = 4, .start = f.start, .end = f.end }); }
        var usize j = 0;
        while (j < f.parameter_count) {
            let usize id = *v::at<usize>(ref edges,f.parameter_start + j);
            if (id <= f.local_start || id > f.local_start + f.local_count) { return .err(Problem { .code = 4, .start = f.start, .end = f.end }); }
            let Local p = *v::at<Local>(ref locals,id - 1);
            if (!p.parameter || p.function != i + 1 || p.type != v::at<Signature>(ref signatures,t.signature_start + j).type) { return .err(Problem { .code = 4, .start = p.start, .end = p.end }); } j += 1;
        }
        i += 1;
    }
    i = 0;
    while (i < nl) {
        let Local l = *v::at<Local>(ref locals,i);
        if (l.function == 0 || l.function > nf || !shapes::valid_type(ref nominal,l.type,false) || l.start >= l.end || l.end > ns) { return .err(Problem { .code = 4, .start = l.start, .end = l.end }); }
        let Function f = *v::at<Function>(ref functions,l.function - 1);
        if (i < f.local_start || i >= f.local_start + f.local_count) { return .err(Problem { .code = 4, .start = l.start, .end = l.end }); } i += 1;
    }
    i = 0;
    while (i < nb) {
        let Block b = *v::at<Block>(ref blocks,i);
        if (b.function == 0 || b.function > nf || b.kind < 1 || b.kind > 7 || b.first > no || b.last > no || (b.first == 0) != (b.last == 0) || b.start > b.end || b.end > ns || b.value > nl || b.left > nl || b.right > nl || b.target > nb || b.alternative > nb || b.check > no || b.callee > nt || b.destination > nl || b.argument_start > ne || b.argument_count > ne - b.argument_start) { return .err(Problem { .code = 4, .start = b.start, .end = b.end }); }
        let Function f = *v::at<Function>(ref functions,b.function - 1);
        if (i < f.block_start || i >= f.block_start + f.block_count) { return .err(Problem { .code = 4, .start = b.start, .end = b.end }); }
        if ((b.kind == 1 || b.kind == 6 || b.kind == 7) && (b.target != 0 || b.alternative != 0)) { return .err(Problem { .code = 4, .start = b.start, .end = b.end }); }
        if ((b.kind >= 2 && b.kind <= 5) && (b.target <= f.block_start || b.target > f.block_start + f.block_count)) { return .err(Problem { .code = 4, .start = b.start, .end = b.end }); }
        if ((b.kind == 3 || b.kind == 4) && (b.alternative <= f.block_start || b.alternative > f.block_start + f.block_count || b.target == b.alternative)) { return .err(Problem { .code = 4, .start = b.start, .end = b.end }); }
        if (b.kind == 2 && b.alternative != 0) { return .err(Problem { .code = 4, .start = b.start, .end = b.end }); }
        if (b.kind == 3 && (b.value == 0 || b.type != 6 || v::at<Local>(ref locals,b.value - 1).type != 6 || v::at<Local>(ref locals,b.value - 1).function != b.function)) { return .err(Problem { .code = 4, .start = b.start, .end = b.end }); }
        if (b.kind == 4 && (b.check == 0 || b.check != b.last || b.trap_mask == 0 || v::at<Operation>(ref operations,b.check - 1).kind != 5 || v::at<Block>(ref blocks,b.alternative - 1).kind != 6 || v::at<Block>(ref blocks,b.alternative - 1).trap_mask != b.trap_mask)) { return .err(Problem { .code = 4, .start = b.start, .end = b.end }); }
        if (b.kind == 5 && (b.callee == 0 || b.argument_count != v::at<Target>(ref targets,b.callee - 1).signature_count)) { return .err(Problem { .code = 4, .start = b.start, .end = b.end }); }
        var usize op = b.first; var usize last = 0;
        while (op != 0) {
            if (op > no || op <= last || v::at<Operation>(ref operations,op - 1).block != i + 1) { return .err(Problem { .code = 4, .start = b.start, .end = b.end }); }
            last = op; op = v::at<Operation>(ref operations,op - 1).next;
        }
        if (last != b.last) { return .err(Problem { .code = 4, .start = b.start, .end = b.end }); } i += 1;
    }
    i = 0;
    while (i < no) {
        let Operation o = *v::at<Operation>(ref operations,i);
        if (o.function == 0 || o.function > nf || o.block == 0 || o.block > nb || o.kind < 1 || o.kind > 20 || o.result > nl || o.left > nl || o.right > nl || o.next > no || (o.next != 0 && o.next <= i + 1) || o.start >= o.end || o.end > ns || o.payload_start > o.payload_end || o.payload_end > np || o.argument_start > ne || o.argument_count > ne - o.argument_start || v::at<Block>(ref blocks,o.block - 1).function != o.function) { return .err(Problem { .code = 4, .start = o.start, .end = o.end }); }
        if ((o.kind != 5 && o.kind != 20 && (o.result == 0 || !shapes::valid_type(ref nominal,o.type,false))) || ((o.kind == 5 || o.kind == 20) && o.result != 0)) { return .err(Problem { .code = 4, .start = o.start, .end = o.end }); } i += 1;
    }
    return .ok(Program { .nominal, .source, .payload, .targets, .signatures, .functions, .locals, .blocks, .operations, .edges, .module_index, .success = true });
}
