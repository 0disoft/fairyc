module module_state_graph_windows_persistent;
import module_state_graph_windows_build as build;
import module_state_graph_windows_run as runner;
import module_state_graph_transport as transport;
import module_state_graph_native as native;
import module_state_graph_request as request;
import module_state_graph_response as response;
import module_state_local_windows as store;
import module_state_local_codec as codec;
import module_state_layout as layout;
import windows_cancel as cancellation;
import std::bytes as bytes;
import std::vec as v;

// Separate cumulative ledgers. Counters count calls, including refused calls.
pub struct Counts{usize snapshots;usize runs;usize publishes;}
pub struct Work{codec::Work local;runner::Work execution;Counts calls;}
pub fn Work empty_work(){return Work{.local=codec::empty_work(),.execution=runner::empty_work(),.calls=Counts{.snapshots=0,.runs=0,.publishes=0}};}
// Acknowledged is irrevocable, including when the provider's receipt disagrees.
// In that case return Outcome with consistent=false, never a retryable error.
pub struct Publication{bool acknowledged;bool consistent;store::Receipt receipt;}
struct ProgramIdentity{u32[8] hash;}
pub resource Outcome{response::Admitted admitted;Publication publication;ProgramIdentity program_identity;cleanup{}}
pub fn ref response::Admitted admitted(ref Outcome outcome){return ref outcome.admitted;}
pub fn Publication publication(ref Outcome outcome){return outcome.publication;}
fn ProgramIdentity identity(ref transport::Prepared prepared){
    let raw=transport::program_hash(prepared);assert(raw.len==32);var u32[8] hash=[0;8];var usize i=0;
    while(i<8){let at=i*4;hash[i]=(raw[at] as u32)<<24|(raw[at+1] as u32)<<16|(raw[at+2] as u32)<<8|(raw[at+3] as u32);i+=1;}return ProgramIdentity{.hash};
}
pub fn bool matches(ref Outcome outcome,ref transport::Prepared prepared){let current=identity(prepared);return layout::equal(outcome.program_identity.hash,current.hash);}
// 1 admission, 2 snapshot, 3 detached request, 4 runner, 5 mutation copy,
// 6 version exhaustion, 7 CAS, 8 graph runtime status, 9 cancellation.
// Original storage/runner/token causes remain available.
// Reject/trap are completed Outcomes: callers inspect admitted.header.exit.
// Failure owners use pre-reserved vectors because resource field moves are unsupported.
pub struct Cause{u32 stage;u32 code;store::Failure storage;u32 runner_stage;u32 runner_status;u32 runner_os_code;usize runtime_status;u32 cancellation_kind;u32 cancellation_os_code;}
pub resource Failure{Cause cause;v::Vec<response::Admitted> admitted;v::Vec<runner::Failure> execution;cleanup{}}
pub fn Cause cause(ref Failure failure){return failure.cause;}
pub fn Result<ref response::Admitted,u32> failure_admitted(ref Failure failure){return if(v::len<response::Admitted>(ref failure.admitted)==0){.err(0)}else{.ok(v::at<response::Admitted>(ref failure.admitted,0))};}
pub fn Result<ref runner::Failure,u32> failure_runner(ref Failure failure){return if(v::len<runner::Failure>(ref failure.execution)==0){.err(0)}else{.ok(v::at<runner::Failure>(ref failure.execution,0))};}
fn store::Receipt empty_receipt(){return store::Receipt{.layout=[0;8],.version=0,.writes=0};}
fn Cause cause_of(u32 stage,u32 code){return Cause{.stage,.code,.storage=store::Failure{.stage=0,.code=0,.os_code=0,.operation=0,.observed=false,.current=empty_receipt()},.runner_stage=0,.runner_status=0,.runner_os_code=0,.runtime_status=0,.cancellation_kind=0,.cancellation_os_code=0};}
@effects(filesystem) fn Result<bool,Cause> checkpoint(ref cancellation::Token token){
    return switch(cancellation::requested(token)){case .ok(stop):{if(stop){.err(cause_of(9,1009))}else{.ok(false)}}case .err(e):{var cause=cause_of(9,1010);cause.cancellation_kind=e.kind;cause.cancellation_os_code=e.os_code;.err(cause)}};
}
fn Failure canceled(Cause cause,v::Vec<response::Admitted> admitted){return Failure{.cause,.admitted,.execution=v::empty<runner::Failure>()};}
fn Failure fail(u32 stage,u32 code){return Failure{.cause=cause_of(stage,code),.admitted=v::empty<response::Admitted>(),.execution=v::empty<runner::Failure>()};}
fn Failure storage_failure(u32 stage,store::Failure error,v::Vec<response::Admitted> admitted){var cause=cause_of(stage,error.code);cause.storage=error;return Failure{.cause,.admitted,.execution=v::empty<runner::Failure>()};}
fn v::Vec<response::Admitted> retained(v::Vec<response::Admitted> reserved,response::Admitted admitted){var out=reserved;assert(v::push<response::Admitted>(ref mut out,admitted).is_ok);return out;}
fn Failure after_run(u32 stage,u32 code,v::Vec<response::Admitted> reserved,response::Admitted admitted){return Failure{.cause=cause_of(stage,code),.admitted=retained(reserved,admitted),.execution=v::empty<runner::Failure>()};}
fn bool valid_limits(request::Limits limits){return limits.writes>0 && limits.writes<=9223372036854775807 && limits.bytes>0 && limits.value>0 && limits.value<=9223372036854775807 && limits.key>0 && limits.key<=9223372036854775807;}
fn bool valid_work(ref Work work){return work.local.structure<=16777216 && work.local.bytes<=1073741824 && work.execution.request_structure<=16777216 && work.execution.request_bytes<=1073741824 && work.execution.response_structure<=16777216 && work.execution.response_bytes<=1073741824 && work.calls.snapshots<16777216 && work.calls.runs<16777216 && work.calls.publishes<16777216;}
fn Result<u32[8],u32> expected(ref build::Built built){
    let raw=native::layout_hash(transport::entry(build::prepared(built)));
    if(build::object_count(built)!=2 || build::data(built).len==0 || raw.len!=32){return .err(1001);}
    var u32[8] hash=[0;8];var bool nonzero=false;var usize i=0;
    while(i<8){let at=i*4;hash[i]=(raw[at] as u32)<<24|(raw[at+1] as u32)<<16|(raw[at+2] as u32)<<8|(raw[at+3] as u32);if(hash[i]!=0){nonzero=true;}i+=1;}
    return if(nonzero){.ok(hash)}else{.err(1001)};
}
fn Result<codec::Mutations,u32> mutations(ref response::Admitted admitted,ref mut codec::Work work){
    let count=response::write_count(admitted);let size=response::header(admitted).write_bytes;
    if(work.structure>16777216 || count>16777216-work.structure || work.bytes>1073741824 || size>1073741824-work.bytes){return .err(1002);}
    work.structure+=count;work.bytes+=size;
    var data=switch(bytes::with_capacity(size)){case .ok(p):{p}case .err(_):{return .err(1003);}};
    var rows=v::empty<codec::Mutation>();if(!v::reserve<codec::Mutation>(ref mut rows,count).is_ok){return .err(1003);}var usize i=0;
    // Admitted alone owns the canonical sorted final overlay, never event writes.
    while(i<count){let row=response::write_at(admitted,i);let value=response::write_value(admitted,i);let start=bytes::len(ref data);
        if(!bytes::append(ref mut data,value).is_ok || !v::push<codec::Mutation>(ref mut rows,codec::Mutation{.key=row.key,.start,.end=start+value.len,.deleted=row.deleted,.was_null=false}).is_ok){return .err(1003);}i+=1;
    }return .ok(codec::mutations_of(data,rows));
}

// Private Windows binding: one snapshot, one Built-backed run, at most one CAS.
// Existing callers retain an uncancelled default. Explicit cancellation observes
// stage boundaries and the last gate before entering the atomic provider CAS.
@effects(filesystem,console,environment,clock)
pub fn Result<Outcome,Failure> run(ref build::Built built,ref store::Store store,request::Limits limits,runner::Options options,ref mut Work work){
    let token=cancellation::none();return run_cancel(built,store,limits,options,ref token,work);
}
@effects(filesystem,console,environment,clock)
pub fn Result<Outcome,Failure> run_cancel(ref build::Built built,ref store::Store store,request::Limits limits,runner::Options options,ref cancellation::Token token,ref mut Work work){
    if(!runner::valid(options) || !valid_limits(limits)){return .err(fail(1,1001));}
    if(!valid_work(work)){return .err(fail(1,1002));}
    let hash=switch(expected(built)){case .ok(p):{p}case .err(code):{return .err(fail(1,code));}};
    switch(checkpoint(token)){case .ok(_):{}case .err(cause):{return .err(canceled(cause,v::empty<response::Admitted>()));}};
    // Reserve retention slots before I/O: preserving a later admitted owner or
    // runner diagnostic never requires a fallible allocation on the error path.
    var retained_response=v::empty<response::Admitted>();var retained_runner=v::empty<runner::Failure>();
    if(!v::reserve<response::Admitted>(ref mut retained_response,1).is_ok || !v::reserve<runner::Failure>(ref mut retained_runner,1).is_ok){return .err(fail(1,1003));}
    work.calls.snapshots+=1;
    let state=switch(store::snapshot(store,hash,ref mut work.local)){case .ok(p):{p}case .err(error):{return .err(storage_failure(2,error,v::empty<response::Admitted>()));}};
    switch(checkpoint(token)){case .ok(_):{}case .err(cause):{return .err(canceled(cause,retained_response));}};
    let version=codec::version(ref state);
    let snapshot=switch(codec::snapshot(ref state,ref mut work.local)){case .ok(p):{p}case .err(code):{return .err(fail(3,code));}};
    switch(checkpoint(token)){case .ok(_):{}case .err(cause):{return .err(canceled(cause,retained_response));}};
    work.calls.runs+=1;
    let result=switch(runner::run_cancel(built,ref snapshot,limits,options,token,ref mut work.execution)){case .ok(p):{p}case .err(error):{
        var cause=cause_of(4,0);cause.runner_stage=runner::stage(ref error);cause.runner_status=runner::status(ref error);cause.runner_os_code=runner::os_code(ref error);
        assert(v::push<runner::Failure>(ref mut retained_runner,error).is_ok);return .err(Failure{.cause,.admitted=v::empty<response::Admitted>(),.execution=retained_runner});
    }};
    let header=response::header(ref result);
    if(header.status!=0){var cause=cause_of(8,0);cause.runtime_status=header.status;return .err(Failure{.cause,.admitted=retained(retained_response,result),.execution=v::empty<runner::Failure>()});}
    if(header.exit!=0){return .ok(Outcome{.admitted=result,.publication=Publication{.acknowledged=false,.consistent=true,.receipt=empty_receipt()},.program_identity=identity(build::prepared(built))});}
    switch(checkpoint(token)){case .ok(_):{}case .err(cause):{return .err(canceled(cause,retained(retained_response,result)));}};
    let count=response::write_count(ref result);
    if(count!=0 && version==18446744073709551615){return .err(after_run(6,1006,retained_response,result));}
    let writes=switch(mutations(ref result,ref mut work.local)){case .ok(p):{p}case .err(code):{return .err(after_run(5,code,retained_response,result));}};
    switch(checkpoint(token)){case .ok(_):{}case .err(cause):{return .err(canceled(cause,retained(retained_response,result)));}};
    // Entering CAS is the uncancellable publication phase. Never poll afterward:
    // a signaled token cannot revoke an acknowledged rename or cause replay.
    work.calls.publishes+=1;
    let receipt=switch(store::compare_and_publish(store,hash,version,ref writes,ref mut work.local)){case .ok(p):{p}case .err(error):{return .err(storage_failure(7,error,retained(retained_response,result)));}};
    let next=if(count==0){version}else{version+1};
    let consistent=layout::equal(receipt.layout,hash) && receipt.version==next && receipt.writes==count;
    return .ok(Outcome{.admitted=result,.publication=Publication{.acknowledged=true,.consistent,.receipt},.program_identity=identity(build::prepared(built))});
}
