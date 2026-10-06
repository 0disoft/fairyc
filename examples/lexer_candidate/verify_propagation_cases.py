"""Propagation fixtures and expected traces, never compiler/runtime logic."""
from verify_results import SCALARS, VALUES, PROPAGATION_NEGATIVES

POSITIVES = [
    (f'propagate-pair-{success}-{failure}',
     f'fn Result<{success},{failure}> make(bool ok){{return if(ok){{.ok({VALUES[success]})}}else{{.err({VALUES[failure]})}};}}'
     f'fn Result<bool,{failure}> convert(bool ok){{let {success} n=make(ok)?;assert(n=={VALUES[success]});return .ok(true);}}'
     f'fn i32 main(){{assert(switch(convert(true)){{case .ok(n):{{n}}case .err(_):{{false}}}});'
     'assert(switch(convert(false)){case .ok(_):{false}case .err(e):{e=='
     + VALUES[failure] + '}});return 0;}', 0)
    for success in SCALARS for failure in SCALARS
]
POSITIVES += [
    ('propagate-ok', 'fn Result<i32,bool> f(Result<i32,bool> r){return .ok(r?+1);}fn i32 main(){return switch(f(.ok(6))){case .ok(n):{n}case .err(_):{99}};}', 7),
    ('propagate-err', 'fn i32 forbidden(){assert(false);return 0;}fn Result<i32,bool> f(Result<i32,bool> r){let n=r?;return .ok(forbidden()+n);}fn i32 main(){return switch(f(.err(true))){case .ok(_):{99}case .err(e):{if(e){7}else{0}}};}', 7),
    ('propagate-once', 'fn Result<i32,bool> f(){var i32 count=0;let Result<i32,bool> r=.ok(6);let n={count+=1;r}?;assert(count==1);return .ok(n+count);}fn i32 main(){return switch(f()){case .ok(n):{n}case .err(_):{99}};}', 7),
    ('propagate-call-argument', 'fn i32 add(i32 a,i32 b){return a+b;}fn Result<i32,bool> f(){let Result<i32,bool> r=.ok(6);return .ok(add(r?,1));}fn i32 main(){return switch(f()){case .ok(n):{n}case .err(_):{99}};}', 7),
    ('propagate-skip-later-argument', 'fn i32 forbidden(){assert(false);return 0;}fn i32 add(i32 a,i32 b){return a+b;}fn Result<i32,bool> f(Result<i32,bool> r){return .ok(add(r?,forbidden()));}fn i32 main(){return switch(f(.err(true))){case .ok(_):{99}case .err(e):{if(e){7}else{0}}};}', 7),
    ('propagate-freeze-earlier-argument', 'fn i32 add(i32 a,i32 b){return a+b;}fn Result<i32,bool> f(){var i32 n=2;let Result<i32,bool> r=.ok(5);return .ok(add(n,{n=99;r}?));}fn i32 main(){return switch(f()){case .ok(n):{n}case .err(_):{99}};}', 7),
    ('propagate-in-if', 'fn Result<i32,bool> f(Result<bool,bool> r){return .ok(if(r?){7}else{8});}fn i32 main(){return switch(f(.ok(true))){case .ok(n):{n}case .err(_):{99}};}', 7),
    ('propagate-in-while-condition', 'fn Result<i32,bool> f(Result<bool,bool> r){while(r?){return .ok(7);}return .ok(8);}fn i32 main(){return switch(f(.err(true))){case .ok(_):{99}case .err(e):{if(e){7}else{0}}};}', 7),
    ('propagate-loop-success', 'fn Result<i32,bool> f(){var i32 n=0;while(n<3){let Result<i32,bool> r=.ok(1);n+=r?;}return .ok(n);}fn i32 main(){return switch(f()){case .ok(n):{n}case .err(_):{99}};}', 3),
    ('propagate-loop-error', 'fn Result<i32,bool> f(){var i32 n=0;while(n<3){let Result<i32,bool> r=if(n==1){.err(true)}else{.ok(1)};n+=r?;}return .ok(99);}fn i32 main(){return switch(f()){case .ok(_):{99}case .err(e):{if(e){7}else{0}}};}', 7),
    ('propagate-early-operand-return', 'fn Result<i32,bool> f(Result<i32,bool> r){let n={return .err(true);r}?;return .ok(n);}fn i32 main(){return switch(f(.ok(3))){case .ok(_):{99}case .err(e):{if(e){7}else{0}}};}', 7),
    ('propagate-short-circuit', 'fn Result<bool,bool> fail(){return .err(true);}fn Result<bool,bool> f(){return .ok(false && fail()?);}fn i32 main(){return switch(f()){case .ok(n):{if(n){99}else{7}}case .err(_):{99}};}', 7),
]
MARK = 'fn void mark(i32 tag){assert(tag>0);}'
for name, body, expected_trace in [
    ('propagate-defer-error', 'defer mark(1);{defer mark(2);let Result<i32,bool> r=.err(true);discard r?;}return .ok(99);', [2,1]),
    ('propagate-defer-success', 'defer mark(1);{defer mark(2);let Result<i32,bool> r=.ok(7);discard r?;}return .ok(7);', [2,1]),
    ('propagate-defer-registration-error', 'defer mark(1);let Result<i32,bool> r=.err(true);defer mark(r?);return .ok(99);', [1]),
]:
    POSITIVES.append((name, MARK+'fn Result<i32,bool> f(){'+body+'}fn i32 main(){return switch(f()){case .ok(n):{n}case .err(e):{if(e){7}else{0}}};}', 7))
TRACES = {
    'propagate-defer-error': [2,1],
    'propagate-defer-success': [2,1],
    'propagate-defer-registration-error': [1],
}
POSITIVES.append(('propagate-defer-siblings', MARK+'fn Result<i32,bool> f(bool yes){defer mark(1);let Result<i32,bool> r=.err(true);if(yes){defer mark(2);discard r?;}else{defer mark(3);discard r?;}return .ok(99);}fn i32 read(Result<i32,bool> r){return switch(r){case .ok(n):{n}case .err(e):{if(e){7}else{0}}};}fn i32 main(){return read(f(true))+read(f(false));}', 14))
TRACES['propagate-defer-siblings'] = [2,1,3,1]
TRAPS = [
    ('propagate-overflow', 'fn Result<i32,bool> f(){let Result<i32,bool> r=.ok(2147483647+1);return .ok(r?);}fn i32 main(){discard f();return 0;}', 1),
    ('propagate-operand-assert', 'fn Result<i32,bool> f(){let Result<i32,bool> r=.ok(1);return .ok({assert(false);r}?);}fn i32 main(){discard f();return 0;}', 5),
    ('propagate-cleanup-trap', 'fn void fail(i32 tag){assert(false);}fn Result<i32,bool> f(){defer fail(1);let Result<i32,bool> r=.err(true);return .ok(r?);}fn i32 main(){discard f();return 0;}', 5),
]
for name, type_name, expression, category in [
    ('i64-sub', 'i64', '-9223372036854775808-1', 1),
    ('usize-add', 'usize', '18446744073709551615+1', 1),
    ('u8-sub', 'u8', '0-1', 1),
    ('i64-mul', 'i64', '-9223372036854775808*-1', 1),
    ('i32-div', 'i32', '-2147483648/-1', 3),
    ('i64-shift', 'i64', '1<<64', 4),
    ('i32-shift-overflow', 'i32', '2147483647<<1', 1),
]:
    TRAPS.append(('propagate-'+name,
        f'fn Result<{type_name},bool> f(){{let Result<{type_name},bool> r=.ok({expression});return .ok(r?);}}'
        'fn i32 main(){discard f();return 0;}', category))
for name, type_name, expression, expected in [
    ('i64-cancel', 'i64', '-9223372036854775808+9223372036854775807', '-1'),
    ('usize-cancel', 'usize', '18446744073709551615-18446744073709551615', '0'),
    ('i64-zero-mul', 'i64', '-9223372036854775808*0', '0'),
]:
    POSITIVES.append(('propagate-'+name,
        f'fn Result<{type_name},bool> f(){{let Result<{type_name},bool> r=.ok({expression});return .ok(r?);}}'
        f'fn i32 main(){{return switch(f()){{case .ok(n):{{if(n=={expected}){{7}}else{{99}}}}case .err(_):{{99}}}};}}', 7))
NEGATIVES = PROPAGATION_NEGATIVES
