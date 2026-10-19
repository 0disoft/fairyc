"""Additional Go-vs-copied-interpreter CLI cases; no native companion at runtime."""

def extend_tagged(fixture, compare):
    tags=fixture('tagged',{'tests/tagged.fy':r'''module tests::tagged;
struct Pair{i32 first;i32 second;}
enum State{Empty;Code{i32 value;};PairValue{Pair pair;i32 bias;};}
enum Envelope{Missing;Present{State state;};}
fn i32 score(State state){return switch(state){case .Empty:{0}case .Code{.value=value}:{value}case .PairValue{.bias=bias,.pair=pair}:{pair.first+pair.second+bias}};}
fn Result<i32,i32> leaf(bool ok){return if(ok){.ok(40)}else{.err(7)};}
fn Result<Pair,i32> middle(bool ok){let i32 n=leaf(ok)?;return .ok(Pair{.first=n,.second=2});}
fn Result<Option<Pair>,i32> outer(bool ok){let Pair p=middle(ok)?;return .ok(.some(p));}
fn Result<i32,i32> discard_success(){discard leaf(true)?;return .ok(42);}
@test fn void enum_variants(){assert(score(.Empty)==0 && score(.Code{.value=42})==42);let State s=.PairValue{.bias=3,.pair=Pair{.first=20,.second=19}};assert(score(s)==42 && score(s)==42);}
@test fn void nested_enum(){let Envelope e=.Present{.state=.Code{.value=42}};assert(switch(e){case .Missing:{0}case .Present{.state=state}:{score(state)}}==42);}
@test fn void option_variants(){let Option<Pair> a=.some(Pair{.first=40,.second=2});let Option<Pair> b=.none;assert(switch(a){case .none:{0}case .some(p):{p.first+p.second}}==42);assert(switch(b){case .some(_):{false}case .none:{true}});}
@test fn void result_variants(){let Result<Pair,i32> a=.ok(Pair{.first=40,.second=2});let Result<Pair,i32> b=.err(7);assert(a.is_ok && !b.is_ok && a.is_ok);assert(switch(a){case .err(_):{0}case .ok(p):{p.first+p.second}}==42);assert(switch(b){case .ok(_):{0}case .err(n):{n}}==7);}
@test fn void propagation(){assert(switch(outer(true)){case .ok(o):{switch(o){case .some(p):{p.first+p.second}case .none:{0}}}case .err(_):{0}}==42);assert(switch(outer(false)){case .ok(_):{0}case .err(n):{n}}==7);assert(discard_success().is_ok);}
@test fn void switch_loops(){var State s=.Empty;var i32 sum=0;var i32 n=0;while(n<4){s=.Code{.value=n};sum+=score(s);n+=1;}assert(sum==6);}
@test fn void payload_copy(){let State s=.PairValue{.pair=Pair{.first=1,.second=2},.bias=3};let i32 n=switch(s){case .Empty:{0}case .Code{.value=_}:{0}case .PairValue{.pair=p,.bias=b}:{var Pair q=p;q.first=37;q.first+q.second+b}};assert(n==42 && score(s)==6);}
'''})
    compare('tagged-json-locations',tags,['--json','--locations'])
    compare('tagged-text',tags)
    compare('tagged-exact',tags,['--exact','tests::tagged::propagation','--json'])

    resources=fixture('resources',{'tests/resources.fy':r'''module tests::resources;
resource Empty{i32 value;cleanup{}}
resource Checked{i32 value;cleanup{assert(this.value==42);}}
resource Mutated{i32 value;cleanup{this.value+=1;assert(this.value==42);}}
resource Parent{Checked child;cleanup{assert(this.child.value==42);}}
resource Nested{Parent child;Checked last;cleanup{assert(this.last.value==42);}}
@abilities(store,key) resource Burned{i32 value;cleanup{assert(false);}}
fn void frozen(i32 n){assert(n==7);}
fn Empty identity(Empty value){return value;}
fn Checked returned(){let Checked c=Checked{.value=42};return c;}
fn Result<i32,Checked> fail(){return .err(Checked{.value=42});}
fn Result<Empty,Checked> forward(){let Checked local=Checked{.value=42};defer frozen(7);let i32 n=fail()?;assert(false);return .ok(Empty{.value=n});}
fn Result<i32,i32> success(){return .ok(42);}
fn Result<i32,i32> forward_ok(){let Checked local=Checked{.value=42};let i32 n=success()?;return .ok(n);}
@test fn void owning_resource(){let Empty r=identity(Empty{.value=42});assert(r.value==42);}
@test fn void cleanup_assert(){let Checked r=Checked{.value=42};assert(r.value==42);}
@test fn void cleanup_receiver_writes(){let Mutated r=Mutated{.value=41};}
@test fn void cleanup_receiver_reads(){let Parent p=Parent{.child=Checked{.value=42}};}
@test fn void nested_cleanup(){let Nested n=Nested{.last=Checked{.value=42},.child=Parent{.child=Checked{.value=42}}};}
@test fn void cleanup_returns(){let Checked r=returned();assert(r.value==42);}
@test fn void cleanup_temporaries(){discard Checked{.value=42};discard returned();}
@test fn void cleanup_replacement(){var Checked r=Checked{.value=42};r=Checked{.value=42};}
@test fn void option_some(){let Option<Checked> o=.some(Checked{.value=42});}
@test fn void option_none(){let Option<Checked> o=.none;}
@test fn void result_ok(){let Result<Checked,Checked> r=.ok(Checked{.value=42});discard r;}
@test fn void result_err(){let Result<Checked,Checked> r=.err(Checked{.value=42});discard r;}
@test fn void active_switch(){let Option<Checked> o=.some(Checked{.value=42});switch(o){case .none:{}case .some(c):{assert(c.value==42);}};}
@test fn void early_propagation(){let Result<Empty,Checked> r=forward();assert(!r.is_ok);}
@test fn void success_propagation(){assert(forward_ok().is_ok);}
@test fn void explicit_burn(){let Burned r=Burned{.value=42};burn r;}
'''})
    compare('resource-cleanup-json',resources,['--json','--locations'])
    compare('resource-cleanup-text',resources)
    compare('resource-cleanup-early-question',resources,['--exact','tests::resources::early_propagation','--json'])

    traps=fixture('cleanup-traps',{'tests/cleanup_traps.fy':r'''module tests::cleanup_traps;
resource Bad{i32 value;cleanup{assert(false);}}
resource Zero{i32 value;cleanup{let i32 zero=0;discard 1/zero;}}
resource Reverse{Bad first;Zero last;cleanup{}}
resource BeforeFields{Zero child;cleanup{assert(false);}}
fn void bad_defer(){let i32 zero=0;discard 1/zero;}
fn Result<i32,i32> fail(){return .err(7);}
fn Result<i32,i32> early(){let Bad local=Bad{.value=1};let i32 n=fail()?;return .ok(n);}
@test fn void a_cleanup_assert(){let Bad r=Bad{.value=1};}
@test fn void b_reverse_fields(){let Reverse r=Reverse{.first=Bad{.value=1},.last=Zero{.value=2}};}
@test fn void c_body_before_fields(){let BeforeFields r=BeforeFields{.child=Zero{.value=1}};}
@test fn void d_return_stops_cleanup(){let Bad r=Bad{.value=1};defer bad_defer();return;}
@test fn void e_question_cleanup(){discard early();}
@test fn void f_early_return_cleanup(){let Bad r=Bad{.value=1};return;}
@test fn void g_after(){assert(true);}
'''})
    compare('cleanup-traps-json',traps,['--json','--locations'],5)
    compare('cleanup-traps-text',traps,code=5)
    for name in ('b_reverse_fields','c_body_before_fields','d_return_stops_cleanup','e_question_cleanup','f_early_return_cleanup'):
        compare('cleanup-trap-'+name,traps,['--exact','tests::cleanup_traps::'+name,'--json'],5)

    ignored=fixture('ignored-cleanup-difference',{'tests/ignored.fy':r'''module tests::ignored;
resource Bad{i32 value;cleanup{assert(false);}}
fn Bad make(){return Bad{.value=1};}
@test fn void direct(){discard Bad{.value=1};}
@test fn void returned(){discard make();}
'''})
    compare('discarded-resource-cleanup',ignored,['--json'],5)

    unsupported=fixture('refs-boundary',{'tests/refs.fy':r'''module tests::refs;
struct Pair{i32 value;}
fn i32 shared(ref Pair p){return p.value;}
@test fn void a_before(){assert(true);}
@test fn void b_shared(){let Pair p=Pair{.value=42};assert(shared(ref p)==42);}
'''})
    compare('shared-record-reference',unsupported,['--json'])

    for name,source in {
        'scalar-switch':'@test fn void invalid(){discard switch(1){case .none:{42}};}',
        'bool-switch':'@test fn void invalid(){discard switch(true){case .none:{42}};}',
        'whole-wildcard':'@test fn void invalid(){let Option<i32> o=.none;discard switch(o){case _:{0}};}',
        'cleanup-write':'resource R{i32 value;cleanup{}} resource Parent{R child;cleanup{this.child.value=42;}} @test fn void invalid(){let Parent r=Parent{.child=R{.value=1}};}',
    }.items():
        home=fixture('rejected-'+name,{'tests/invalid.fy':'module tests::invalid;\n'+source+'\n'})
        compare('admission-'+name,home,['--json'],3)

    limits=fixture('tagged-limits',{'tests/limits.fy':r'''module tests::limits;
@test fn void a_before(){assert(true);}
@test fn void b_heap(){var usize i=0;while(true){let Option<i32> o=.some(42);assert(switch(o){case .some(n):{n}case .none:{0}}==42);i+=1;}}
'''})
    compare('tagged-step-no-partial-report',limits,['--json'],2,limit='step limit exceeded')

"""Standalone bounded semantic CLI probes. Every parity case must succeed in Go."""
CASES = {
'scalar-refs': r'''module tests::refs;
fn void add(ref mut i32 x,i32 delta){*x+=delta;}
fn i32 read(ref i32 x){return *x;}
fn void chain(ref mut i32 x){add(x,3);assert(read(x)==10);*x*=2;}
@test fn void scalar_mutation(){var i32 x=7;chain(ref mut x);assert(x==20);let ref i32 r=ref x;assert(*r==20);}
@test fn void scalar_types(){var i64 a=-4;var u8 b=3;var u32 c=9;var usize d=6;var bool e=false;{let ref mut i64 r=ref mut a;*r-=3;}{let ref mut u8 r=ref mut b;*r*=4;}{let ref mut u32 r=ref mut c;*r=17;}{let ref mut usize r=ref mut d;*r+=4;}{let ref mut bool r=ref mut e;*r=true;}assert(a== -7 && b==12 && c==17 && d==10 && e);}
@test fn void direct(){var i32 x=2;let ref mut i32 r=ref mut x;*r+=5;assert(*r==7);}
''',
'struct-refs': r'''module tests::refs;
struct Pair{i32 first;i32 second;}
struct Outer{Pair pair;i32[3] items;}
fn i32 read(ref Pair p){return p.first+p.second;}
fn void inc(ref mut i32 x){*x+=4;}
fn void update(ref mut Pair p){inc(ref mut p.first);p.second*=3;assert(read(p)==11);}
fn void replace(ref mut Pair p,Pair value){*p=value;}
@test fn void struct_copy_and_calls(){let Pair original=Pair{.first=1,.second=2};var Pair q=original;update(ref mut q);assert(original.first==1 && original.second==2 && q.first==5 && q.second==6);replace(ref mut q,Pair{.first=12,.second=13});assert(q.first==12 && q.second==13 && original.first==1);}
@test fn void projected_paths(){var Outer x=Outer{.pair=Pair{.first=1,.second=2},.items=[3,4,5]};let Outer copy=x;update(ref mut x.pair);inc(ref mut x.items[1]);assert(x.pair.first==5 && x.items[1]==8 && copy.pair.first==1 && copy.items[1]==4);}
@test fn void repeat_array_value_copy(){var Pair[2] a=[Pair{.first=1,.second=2};2];update(ref mut a[0]);assert(a[0].first==5 && a[1].first==1);}
fn i32 change_index(ref mut usize i){*i=0;return 9;}
@test fn void frozen_selector(){var i32[2] a=[1,2];var usize i=1;{let ref mut i32 r=ref mut a[i];i=0;*r=7;}assert(a[0]==1 && a[1]==7);i=1;a[i]+=change_index(ref mut i);assert(i==0 && a[0]==1 && a[1]==16);}
''',
'array-views': r'''module tests::views;
fn i32 read(ref i32[] a){assert(!a.is_empty);return a[0];}
fn void modify(ref mut i32[] a){a[0]+=7;a[a.len-1]*=2;assert(read(a)==10);}
fn void inc(ref mut i32 x){*x+=5;}
@test fn void calls_and_copies(){var i32[3] a=[3,4,5];let i32[3] copy=a;modify(ref mut a);assert(a[0]==10 && a[2]==10 && copy[0]==3 && copy[2]==5);}
@test fn void reborrow_and_elements(){var i32[4] a=[1,2,3,4];{let ref mut i32[] view=ref mut a;view[1]=8;inc(ref mut view[2]);assert(view[2]==8 && view.len==4);}assert(a[1]==8 && a[2]==8);}
@test fn void scalar_view_types(){var u8[2] b=[2,3];{let ref mut u8[] v=ref mut b;v[0]*=4;}var i64[2] c=[-7,9];{let ref mut i64[] v=ref mut c;v[0]+=3;}var bool[2] d=[false,true];{let ref mut bool[] v=ref mut d;v[0]=v[1];}assert(b[0]==8 && c[0]== -4 && d[0]);}
''',
'slice-views': r'''module tests::slices;
fn void inc(ref mut i32[] a){a[0]+=10;}
struct A{i32[4] items;}
@test fn void slices(){var i32[5] a=[1,2,3,4,5];let i32[5] copy=a;{let ref mut i32[] v=ref mut a;let ref mut i32[] s=v[1..4];inc(s);s[1]*=2;assert(s.len==3 && !s.is_empty && s[0]==12 && s[1]==6);}assert(a[1]==12 && a[2]==6 && copy[1]==2);}
@test fn void omitted_and_empty(){let i32[3] a=[3,4,5];let ref i32[] v=ref a;assert(v[..2].len==2 && v[1..].len==2 && v[..].len==3 && v[3..3].is_empty);assert(v[1..][1]==5);}
@test fn void field_view(){var A a=A{.items=[1,2,3,4]};let A before=a;{let ref mut i32[] v=ref mut a.items;v[1..3][0]=9;}assert(a.items[1]==9 && before.items[1]==2);}
''',
'literals': r'''module tests::literals;
fn usize length(ref str s){return s.len;}
fn u8 pick(ref u8[] s,usize i){return s[i];}
@test fn void static_bytes(){let ref u8[] a=b"A\0\xffZ";assert(a.len==4 && !a.is_empty && a[0]==65 && a[1]==0 && a[2]==255);assert(pick(a[1..3],1)==255 && a[..0].is_empty && a[4..].is_empty);}
@test fn void static_strings(){let ref str s="한글";assert(s.len==6 && !s.is_empty && length(s)==6);let ref str empty="";assert(empty.len==0 && empty.is_empty);}
''',
'combined-tagged-resource': r'''module tests::combined;
struct Pair{i32 first;i32 second;}
enum Choice{No;Yes{Pair payload;};}
resource Guard{i32 value;cleanup{assert(this.value==9);}}
fn void inc(ref mut i32 x){*x+=4;}
fn Result<i32,i32> checked(Result<i32,i32> r){let i32 n=r?;return .ok(n+1);}
@test fn void enum_payload(){let Choice c=.Yes{.payload=Pair{.first=1,.second=2}};let i32 n=switch(c){case .No:{0}case .Yes{.payload=p}:{var Pair q=p;inc(ref mut q.first);assert(p.first==1);q.first}};assert(n==5);}
@test fn void result_and_option(){assert(checked(.ok(8)).is_ok);let Option<i32> o=.some(5);let i32 n=switch(o){case .some(v):{var i32 q=v;inc(ref mut q);q}case .none:{0}};assert(n==9);}
@test fn void resource_drop(){var Guard g=Guard{.value=5};g.value=9;}
''',
'regressions': r'''module tests::regressions;
struct Pair{i32 first;i32 second;}
fn Pair change(Pair p){var Pair q=p;q.first=9;return q;}
fn void frozen(i32 n){assert(n==7);}
fn void late(ref i32 n){assert(*n==11);}
fn i32 factorial(i32 n){return if(n==0){1}else{n*factorial(n-1)};}
@test fn void old_values(){let Pair p=Pair{.first=1,.second=2};let Pair q=change(p);assert(p.first==1 && q.first==9 && factorial(5)==120);var i32[3] a=[1,2,3];a[1]*=4;assert(a[1]==8);}
@test fn void captured_values(){var i32 n=7;defer frozen(n);n=11;}
@test fn void shared_call(){var i32 n=11;late(ref n);assert(n==11);}
@test fn void loops(){var i32 total=0;for(var i32 i=0;i<8;i+=1){if(i==2){continue;}if(i==6){break;}total+=i;}assert(total==13);}
''',
}

CASES['struct-views'] = r'''module tests::structs;
struct Pair{i32 value;}
fn i32 sum(ref Pair[] values){var i32 n=0;for(let ref Pair value in values){n+=value.value;}return n;}
fn void inc(ref mut Pair[] values){for(let ref mut Pair value in values){value.value+=1;}values[0].value+=4;}
@test fn void borrowed_struct_array(){var Pair[3] a=[Pair{.value=7};3];let Pair[3] before=a;inc(ref mut a);assert(a[0].value==12 && a[1].value==8 && before[0].value==7 && sum(ref a)==28);}
@test fn void slices(){var Pair[3] a=[Pair{.value=7};3];{let ref mut Pair[] values=ref mut a;inc(values[1..]);assert(values[..1].len==1 && values[3..].is_empty);}assert(a[0].value==7 && a[1].value==12 && a[2].value==8);}
'''

CASES['long-literals'] = ('module tests::probe;@test fn void long_bytes(){let ref u8[] b=b"'
    + 'a'*5000 + 'Z";assert(b.len==5001 && b[4096]==97 && b[5000]==90 && b[4096..].len==905);}\n'
    + '@test fn void long_string(){let ref str s="' + 'x'*5000 + '";assert(s.len==5000 && !s.is_empty);}\n')

TRAPS = {
'view-load-bounds': 'let i32[2] a=[1,2];let ref i32[] v=ref a;let usize i=2;discard v[i];',
'view-store-bounds': 'var i32[2] a=[1,2];let ref mut i32[] v=ref mut a;let usize i=2;v[i]=3;',
'view-compound-bounds': 'var i32[2] a=[1,2];let ref mut i32[] v=ref mut a;let usize i=2;v[i]+=3;',
'view-element-borrow-bounds': 'var i32[2] a=[1,2];let ref mut i32[] v=ref mut a;let usize i=2;let ref mut i32 r=ref mut v[i];*r=3;',
'slice-inverted': 'let i32[2] a=[1,2];let ref i32[] v=ref a;let usize lo=2;let usize hi=1;discard v[lo..hi];',
'slice-over-end': 'let ref u8[] b=b"ab";let usize hi=3;discard b[..hi];',
'empty-view-index': 'let ref u8[] b=b"";let usize i=0;discard b[i];',
'ref-overflow': 'var i32 n=2147483647;let ref mut i32 r=ref mut n;*r+=1;',
'ref-div-zero': 'var i32 n=9;let ref mut i32 r=ref mut n;*r/=0;',
}


def extend_references(fixture, compare):
    for name, text in CASES.items():
        module = text.split(';', 1)[0].split()[-1].split('::')[-1]
        home = fixture('references-' + name, {'tests/' + module + '.fy': text})
        compare('references-' + name + '-json', home, ['--json', '--locations'])
        compare('references-' + name + '-text', home)
    for name, body in TRAPS.items():
        text = 'module tests::traps;@test fn void a_trap(){' + body + '}@test fn void z_after(){assert(true);}\n'
        home = fixture('references-' + name, {'tests/traps.fy': text})
        compare('references-' + name + '-json', home, ['--json'], 5)
        compare('references-' + name + '-text', home, code=5)


def extend_iteration_and_locations(fixture, compare):
    home = fixture('iteration', {'tests/iteration.fy': r'''module tests::iteration;
struct Pair{i32 value;}
fn void bump(ref mut i32[] values){for(let ref mut i32 value in values){*value+=1;}}
@test fn void scalar_and_mutable(){
 var i32[4] values=[1,2,3,4];let i32[4] before=values;var i32 sum=0;
 for(let i32 value in values){if(value==2){continue;}sum+=value;if(value==3){break;}}assert(sum==4);
 bump(ref mut values);assert(values[0]==2 && values[3]==5 && before[0]==1);
 var i32 total=0;for(let ref i32 value in values){total+=*value;}assert(total==14);
}
@test fn void structs_and_nested(){
 var Pair[2] values=[Pair{.value=7};2];let Pair[2] before=values;
 for(let ref mut Pair value in values){value.value+=1;}
 var i32 sum=0;for(let Pair value in values){sum+=value.value;}assert(sum==16 && before[0].value==7);
 let i32[2] inner=[1,2];var i32 count=0;for(let ref Pair value in values){for(let i32 n in inner){count+=value.value+n;}}assert(count==38);
}
@test fn void views_and_empty(){
 let ref u8[] empty=b"";for(let u8 n in empty){assert(false);}
 let ref u8[] bytes=b"abc";var i32 sum=0;for(let u8 n in bytes[1..]){sum+=n as i32;}assert(sum==197);
}
'''})
    compare('iteration-json', home, ['--json', '--locations'])
    compare('iteration-text', home)
    prod = 'module loc;\npub fn ref str where(){return source_location();}\npub fn T touch<T>(T n){let ref str location=source_location();assert(location.len==G);return n;}\n'

    def location(text, file, occurrence=0):
        raw = text.encode()
        at = 0
        for _ in range(occurrence + 1):
            start = raw.index(b'source_location', at)
            at = start + 1
        before = raw[:start]
        return f'{file}:{before.count(bytes([10])) + 1}:{start - (before.rfind(bytes([10])) + 1) + 1}@{start}'

    prod = prod.replace('G', str(len(location(prod, 'loc.fy', 1))))
    text = 'module tests::locations;\r\nimport loc as loc;\r\n@test fn void lexical(){/*한글*/let ref str here=source_location();assert(here.len==N);assert(loc::where().len==P);assert(!here.is_empty && loc::touch<i32>(42)==42);}\r\n'
    text = text.replace('N', str(len(location(text, 'tests/locations.fy')))).replace('P', str(len(location(prod, 'loc.fy'))))
    home = fixture('source-locations', {'src/loc.fy': prod, 'tests/locations.fy': text})
    compare('source-location-json', home, ['--json', '--locations'])
    compare('source-location-text', home)


def extend_arenas(fixture, compare):
    home = fixture('arenas', {'tests/arena.fy': r'''module tests::arena;
import std::alloc as alloc;
struct Pair{i32 value;}
resource Holder{alloc::Arena arena;cleanup{}}
resource Guard{i32 value;cleanup{assert(this.value==42);}}
resource Trap{i32 value;cleanup{assert(false);}}
fn alloc::Arena make(){return switch(alloc::create()){case .ok(a):{a}case .err(_):{assert(false);make()}};}
fn Result<i32,i32> fail(){return .err(7);}
fn Result<i32,i32> route(bool propagate){
 var alloc::Arena arena=make();region r in ref mut arena{
  let Guard guard=Guard{.value=42};assert(switch(r.place(42)){case .ok(n):{*n==42}case .err(_):{false}});
  if(propagate){let i32 n=fail()?;}return .ok(42);
 }
}
@test fn void a_place_and_reenter(){
 var alloc::Arena arena=make();var i32 i=0;
 while(i<3){region r in ref mut arena{assert(switch(r.place(i)){case .ok(n):{*n==i}case .err(_):{false}});i+=1;continue;}}
 region r in ref mut arena{var Pair original=Pair{.value=42};switch(r.place(original)){case .ok(stored):{original.value=0;assert(stored.value==42);}case .err(_):{assert(false);}};}
 assert(i==3);
}
@test fn void b_transfer_and_propagation(){let Holder owner=Holder{.arena=make()};assert(route(false).is_ok);assert(switch(route(true)){case .ok(_):{false}case .err(n):{n==7}});}
@test fn void c_independent_arenas(){var alloc::Arena first=make();var alloc::Arena second=make();region r in ref mut first{region s in ref mut second{discard r.place(20);discard s.place(22);return;}}}
@test fn void d_body_trap(){var alloc::Arena arena=make();region r in ref mut arena{discard r.place(42);assert(false);}}
@test fn void e_cleanup_trap(){var alloc::Arena arena=make();region r in ref mut arena{let Trap guard=Trap{.value=42};discard r.place(42);return;}}
@test fn void z_fresh_after_traps(){var alloc::Arena arena=make();region r in ref mut arena{assert(r.place(42).is_ok);}}
'''})
    compare('arenas-json', home, ['--json'], 5)
    compare('arenas-locations', home, ['--json', '--locations'], 5)
    compare('arenas-text', home, code=5)
    compare('arenas-placement', home, ['--exact', 'tests::arena::a_place_and_reenter', '--json'])
    compare('arenas-transfer', home, ['--exact', 'tests::arena::b_transfer_and_propagation'])
    compare('arenas-fresh-test', home, ['--exact', 'tests::arena::z_fresh_after_traps', '--json'])


def extend(fixture, compare):
    extend_arenas(fixture, compare)
    extend_tagged(fixture, compare)
    extend_references(fixture, compare)
    extend_iteration_and_locations(fixture, compare)

