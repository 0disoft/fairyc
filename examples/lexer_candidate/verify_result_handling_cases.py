"""Result handling fixtures; Python supplies input and fixed expected outcomes only."""
R = 'Result<i32,bool>'
MAKE = f'fn {R} make(){{return .ok(7);}}'
POSITIVES = [
    ('discard-call', MAKE+'fn i32 main(){discard make();return 7;}', 7),
    ('discard-local', 'fn i32 main(){let '+R+' r=.err(true);discard r;return 7;}', 7),
    ('is-ok-both', 'fn i32 main(){let '+R+' a=.ok(7);let '+R+' b=.err(true);assert(a.is_ok);assert(!b.is_ok);return 7;}', 7),
    ('is-ok-call', MAKE+'fn i32 main(){assert(make().is_ok);return 7;}', 7),
    ('parameter-is-ok', 'fn bool handled('+R+' r){return r.is_ok;}fn i32 main(){assert(handled(.ok(1)));assert(!handled(.err(false)));return 7;}', 7),
    ('return-transfer', MAKE+'fn '+R+' pass('+R+' r){return r;}fn i32 main(){discard pass(make());return 7;}', 7),
    ('copy-double', 'fn i32 main(){let '+R+' r=.ok(7);let a=r;discard a;discard r;return 7;}', 7),
    ('self-assign', 'fn i32 main(){var '+R+' r=.ok(7);r=r;discard r;return 7;}', 7),
    ('handled-overwrite', 'fn i32 main(){var '+R+' r=.ok(1);discard r;r=.err(false);discard r;return 7;}', 7),
    ('branch-both', 'fn void use(bool b,'+R+' r){if(b){discard r;}else{assert(r.is_ok);}}fn i32 main(){use(true,.ok(1));use(false,.ok(1));return 7;}', 7),
    ('branch-return', 'fn void use(bool b,'+R+' r){if(b){discard r;return;}discard r;}fn i32 main(){use(true,.ok(1));use(false,.err(true));return 7;}', 7),
    ('branch-local', 'fn i32 main(){if(true){let '+R+' r=.ok(1);discard r;}else{let '+R+' r=.err(false);discard r;}return 7;}', 7),
    ('loop-handled', 'fn i32 main(){var i32 i=0;while(i<2){let '+R+' r=.ok(i);discard r;i+=1;}return 7;}', 7),
    ('loop-outer', 'fn i32 main(){var '+R+' r=.ok(1);var i32 i=0;while(i<2){discard r;r=.ok(i);i+=1;}discard r;return 7;}', 7),
    ('loop-break', 'fn i32 main(){while(true){let '+R+' r=.ok(1);discard r;break;}return 7;}', 7),
    ('loop-continue', 'fn i32 main(){var i32 i=0;while(i<2){let '+R+' r=.ok(i);discard r;i+=1;continue;}return 7;}', 7),
    ('for-handled', 'fn i32 main(){for(var i32 i=0;i<2;i+=1){let '+R+' r=.ok(i);discard r;}return 7;}', 7),
    ('shadow', 'fn i32 main(){let '+R+' r=.ok(1);{let '+R+' r=.err(true);discard r;};discard r;return 7;}', 7),
    ('group-block', MAKE+'fn i32 main(){discard ({make()});return 7;}', 7),
    ('switch-result-transfer', 'fn '+R+' choose('+R+' r){return switch(r){case .ok(n):{.ok(n)}case .err(e):{.err(e)}};}fn i32 main(){discard choose(.ok(1));return 7;}', 7),
    ('propagate', 'fn '+R+' pass('+R+' r){let n=r?;return .ok(n);}fn i32 main(){assert(pass(.ok(7)).is_ok);assert(!pass(.err(true)).is_ok);return 7;}', 7),
    ('propagate-other-handled', 'fn '+R+' pass('+R+' r){let '+R+' other=.ok(1);discard other;let n=r?;return .ok(n);}fn i32 main(){discard pass(.err(true));return 7;}', 7),
    ('short-circuit-handled', 'fn bool use(bool b,'+R+' r){let x=b&&r.is_ok;discard r;return x;}fn i32 main(){assert(!use(false,.ok(1)));return 7;}', 7),
    ('early-operand', MAKE+'fn i32 main(){discard {return 7;make()};return 1;}', 7),
    ('unreachable-call', MAKE+'fn i32 main(){return 7;make();}', 7),
]
BAD = [
    ('bare-call', MAKE+'fn i32 main(){make();return 0;}'),
    ('bare-call-after-return', MAKE+'fn i32 main(){make();return 0;make();}'),
    ('pending-local', 'fn i32 main(){let '+R+' r=.ok(1);return 0;}'),
    ('pending-parameter', 'fn void use('+R+' r){}fn i32 main(){return 0;}'),
    ('bare-local', 'fn i32 main(){let '+R+' r=.ok(1);r;return 0;}'),
    ('overwrite', 'fn i32 main(){var '+R+' r=.ok(1);r=.err(false);discard r;return 0;}'),
    ('new-generation', 'fn i32 main(){var '+R+' r=.ok(1);discard r;r=.err(false);return 0;}'),
    ('copy-pending', 'fn i32 main(){let '+R+' r=.ok(1);let copy=r;return 0;}'),
    ('branch-one', 'fn void use(bool b,'+R+' r){if(b){discard r;}}fn i32 main(){return 0;}'),
    ('branch-early', 'fn void use(bool b,'+R+' r){if(b){return;}discard r;}fn i32 main(){return 0;}'),
    ('branch-inner', 'fn i32 main(){if(true){let '+R+' r=.ok(1);}return 0;}'),
    ('literal-false-local', 'fn i32 main(){while(false){let '+R+' r=.ok(1);}return 0;}'),
    ('loop-zero', 'fn void use(bool b,'+R+' r){while(b){discard r;break;}}fn i32 main(){return 0;}'),
    ('loop-backedge', 'fn i32 main(){var '+R+' r=.ok(1);discard r;var i32 i=0;while(i<2){r=.ok(i);i+=1;}discard r;return 0;}'),
    ('loop-break-pending', 'fn i32 main(){while(true){let '+R+' r=.ok(1);break;}return 0;}'),
    ('loop-continue-pending', 'fn i32 main(){var i32 i=0;while(i<2){let '+R+' r=.ok(1);i+=1;continue;}return 0;}'),
    ('for-inner-pending', 'fn i32 main(){for(var i32 i=0;i<1;i+=1){let '+R+' r=.ok(1);}return 0;}'),
    ('scope-exit', 'fn i32 main(){{let '+R+' r=.ok(1);};return 0;}'),
    ('propagate-pending', 'fn '+R+' pass('+R+' r){let '+R+' other=.ok(1);let n=r?;discard other;return .ok(n);}fn i32 main(){return 0;}'),
    ('assert-false-pending', 'fn void use('+R+' r){assert(false);}fn i32 main(){return 0;}'),
    ('early-return-pending', MAKE+'fn i32 main(){let '+R+' r=make();discard {return 0;r};return 0;}'),
    ('group-unhandled', MAKE+'fn i32 main(){(make());return 0;}'),
    ('block-unhandled', MAKE+'fn i32 main(){{make()};return 0;}'),
    ('if-unhandled', MAKE+'fn i32 main(){if(true){make()}else{make()};return 0;}'),
    ('switch-unhandled', MAKE+'fn i32 main(){let '+R+' r=.ok(1);switch(r){case .ok(_):{make()}case .err(_):{make()}};return 0;}'),
    ('short-circuit-and', 'fn void use(bool b,'+R+' r){discard b&&r.is_ok;}fn i32 main(){return 0;}'),
    ('short-circuit-or', 'fn void use(bool b,'+R+' r){discard b||r.is_ok;}fn i32 main(){return 0;}'),
]
NEGATIVES = [(name, source, 'Result value must be handled') for name, source in BAD]
TRAPS = []

# Keep prior evaluation-order regressions executable under the newly enforced
# handling contract. These fixtures retain their original native outcomes.
from verify_result_execution import POSITIVES as PREVIOUS
from verify_propagation_cases import POSITIVES as PROPAGATION
POSITIVES += [case for case in PREVIOUS if case[0] in {'copy', 'early-argument-frozen', 'subject-once'}]
POSITIVES += [case for case in PROPAGATION if case[0] == 'propagate-early-operand-return']
