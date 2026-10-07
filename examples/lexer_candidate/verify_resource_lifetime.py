#!/usr/bin/env python3
"""Bounded fixtures and native observations; all lifetime logic is Fairy code."""
import argparse, hashlib, json, re, subprocess
from pathlib import Path

ROOT = Path(__file__).resolve().parent
REPO = ROOT.parents[1]
PREFIX = 'module main;fn void mark(i32 n){}'
CASES = {
    22: [
        ('lifo', 'resource R{i32 n;cleanup{mark(this.n);}}fn i32 main(){{let R a=R{.n=65};let R b=R{.n=66};};return 0;}', 'BA', 0, 0),
        ('nested', 'resource R{i32 n;cleanup{mark(this.n);}}fn i32 main(){{let R a=R{.n=65};{let R b=R{.n=66};let R c=R{.n=67};};};return 0;}', 'CBA', 0, 0),
        ('move-binding', 'resource R{i32 n;cleanup{mark(this.n);}}fn i32 main(){{let R a=R{.n=65};let R b=a;};return 0;}', 'A', 0, 0),
        ('call-transfer', 'resource R{i32 n;cleanup{mark(this.n);}}fn void take(R r){mark(88);}fn i32 main(){{let R a=R{.n=65};take(a);};return 0;}', 'XA', 0, 0),
        ('receiver-store', 'resource R{i32 n;cleanup{this.n+=1;mark(this.n);}}fn i32 main(){{let R a=R{.n=65};};return 0;}', 'B', 0, 0),
        ('empty', 'resource R{cleanup{}}fn void take(R r){}fn i32 main(){{let R a=R{};let R b=a;take(b);};return 0;}', '', 0, 0),
        ('shadow', 'resource R{i32 n;cleanup{mark(this.n);}}fn i32 main(){{let R r=R{.n=65};{let R r=R{.n=66};};};return 0;}', 'BA', 0, 0),
        ('wide-field', 'resource R{i64 n;cleanup{assert(this.n==4000000000);mark(65);}}fn i32 main(){{let R r=R{.n=4000000000};assert(r.n==4000000000);};return 7;}', 'A', 7, 0),
    ],
    23: [
        ('return-root', 'resource R{i32 n;cleanup{mark(this.n);}}fn i32 main(){let R r=R{.n=65};return 7;}', 'A', 7, 0),
        ('return-nested', 'resource R{i32 n;cleanup{mark(this.n);}}fn i32 main(){let R a=R{.n=65};{let R b=R{.n=66};return 9;};return 0;}', 'BA', 9, 0),
        ('return-frozen', 'resource R{i32 n;cleanup{this.n+=1;mark(this.n);}}fn i32 main(){let R r=R{.n=65};return r.n;}', 'B', 65, 0),
        ('return-donor', 'resource R{i32 n;cleanup{mark(this.n);}}fn R make(){return R{.n=65};}fn R id(R r){return r;}fn i32 main(){let R r=id(make());return 7;}', 'A', 7, 0),
        ('return-if', 'resource R{i32 n;cleanup{mark(this.n);}}fn R select(bool yes){return if(yes){R{.n=65}}else{R{.n=66}};}fn i32 main(){let R a=select(true);let R b=select(false);return 0;}', 'BA', 0, 0),
        ('return-switch', 'resource R{i32 n;cleanup{mark(this.n);}}enum E{A;B;}fn R select(E e){return switch(e){case .A:{R{.n=65}}case .B:{R{.n=66}}};}fn i32 main(){let R r=select(.B);return 0;}', 'B', 0, 0),
        ('question-ok', 'resource R{i32 n;cleanup{mark(this.n);}}fn Result<i32,bool> f(bool ok){let R r=R{.n=65};let Result<i32,bool> input=if(ok){.ok(7)}else{.err(true)};let i32 n=input?;mark(88);return .ok(n);}fn i32 main(){return switch(f(true)){case .ok(n):{n}case .err(_):{99}};}', 'XA', 7, 0),
        ('question-err', 'resource R{i32 n;cleanup{mark(this.n);}}fn Result<i32,bool> f(bool ok){let R r=R{.n=65};let Result<i32,bool> input=if(ok){.ok(7)}else{.err(true)};let i32 n=input?;mark(88);return .ok(n);}fn i32 main(){return switch(f(false)){case .ok(n):{n}case .err(e):{if(e){1}else{2}}};}', 'A', 1, 0),
        ('short-return', 'resource R{i32 n;cleanup{mark(this.n);}}fn i32 f(bool yes){let R r=R{.n=65};discard yes && {return 7;true};return 8;}fn i32 main(){return f(false)+f(true);}', 'AA', 15, 0),
        ('cleanup-local', 'resource S{cleanup{mark(83);}}resource R{cleanup{let S s=S{};mark(82);return;}}fn i32 main(){let R r=R{};return 0;}', 'RS', 0, 0),
    ],
    27: [
        ('wildcard-scope-order', 'resource R{i32 n;cleanup{mark(this.n);}}fn i32 main(){let Option<R> x=.some(R{.n=65});discard switch(x){case .some(_):{let R b=R{.n=66};mark(88);}case .none:{mark(89);}};return 0;}', 'XBA', 0, 0),
        ('wildcard-continue', 'resource R{i32 n;cleanup{mark(this.n);}}fn i32 main(){var i32 n=0;while(n<2){let Option<R> x=.some(R{.n=65});n+=1;discard switch(x){case .some(_):{let R b=R{.n=66};continue;}case .none:{break;}};}return 0;}', 'BABA', 0, 0),
        ('option-some-move', 'resource R{i32 n;cleanup{mark(this.n);}}fn i32 main(){let Option<R> x=.some(R{.n=65});let Option<R> y=x;return switch(y){case .some(r):{r.n}case .none:{0}};}', 'A', 65, 0),
        ('option-none', 'resource R{i32 n;cleanup{mark(this.n);}}fn i32 main(){let Option<R> x=.none;return 0;}', '', 0, 0),
        ('option-wildcard', 'resource R{i32 n;cleanup{mark(this.n);}}fn i32 main(){let Option<R> x=.some(R{.n=65});discard switch(x){case .some(_):{mark(88);}case .none:{mark(89);}};return 0;}', 'XA', 0, 0),
        ('option-nested', 'resource R{i32 n;cleanup{mark(this.n);}}fn i32 main(){let Option<Option<R>> x=.some(.some(R{.n=65}));return 0;}', 'A', 0, 0),
        ('resource-tag-field', 'resource S{i32 n;cleanup{mark(this.n);}}resource R{Option<S> s;cleanup{mark(82);}}fn i32 main(){let R r=R{.s=.some(S{.n=65})};return 0;}', 'RA', 0, 0),
        ('result-ok', 'resource R{i32 n;cleanup{mark(this.n);}}fn Result<R,bool> make(){return .ok(R{.n=65});}fn i32 main(){return switch(make()){case .ok(r):{r.n}case .err(_):{0}};}', 'A', 65, 0),
        ('result-err-resource', 'resource R{i32 n;cleanup{mark(this.n);}}fn Result<i32,R> make(){return .err(R{.n=69});}fn Result<i32,R> f(){let R a=R{.n=65};let i32 n=make()?;return .ok(n);}fn i32 main(){return switch(f()){case .ok(n):{n}case .err(e):{e.n}};}', 'AE', 69, 0),
        ('replace-question-ok', 'resource R{i32 n;cleanup{mark(this.n);}}fn Result<R,bool> make(){mark(88);return .ok(R{.n=66});}fn Result<i32,bool> f(){var R old=R{.n=65};old=make()?;return .ok(old.n);}fn i32 main(){return switch(f()){case .ok(n):{n}case .err(_):{1}};}', 'XAB', 66, 0),
        ('replace-question-err', 'resource R{i32 n;cleanup{mark(this.n);}}fn Result<R,bool> make(){mark(88);return .err(true);}fn Result<i32,bool> f(){var R old=R{.n=65};old=make()?;return .ok(old.n);}fn i32 main(){return switch(f()){case .ok(n):{n}case .err(_):{1}};}', 'XA', 1, 0),
        ('tagged-if-return', 'resource R{i32 n;cleanup{mark(this.n);}}fn Option<R> f(bool yes){return if(yes){.some(R{.n=65})}else{.none};}fn i32 main(){let Option<R> a=f(true);let Option<R> b=f(false);return 0;}', 'A', 0, 0),
    ],
    26: [
        ('partial-factory', 'resource S{i32 n;cleanup{mark(this.n);}}resource R{S s;i32 n;cleanup{mark(82);}}fn S make(){return S{.n=65};}fn Result<i32,bool> bad(){return .err(true);}fn Result<i32,bool> f(){let R r=R{.s=make(),.n=bad()?};return .ok(0);}fn i32 main(){return switch(f()){case .ok(n):{n}case .err(_):{1}};}', 'A', 1, 0),
        ('nested-success', 'resource S{i32 n;cleanup{mark(this.n);}}resource R{S a;S b;cleanup{mark(82);}}fn i32 main(){let R r=R{.a=S{.n=65},.b=S{.n=66}};return 0;}', 'RBA', 0, 0),
        ('partial-middle', 'resource S{i32 n;cleanup{mark(this.n);}}resource R{S a;S b;i32 n;cleanup{mark(82);}}fn Result<i32,bool> bad(){mark(88);return .err(true);}fn Result<i32,bool> f(){let S outer=S{.n=79};let R r=R{.b=S{.n=66},.a=S{.n=65},.n=bad()?};return .ok(0);}fn i32 main(){return switch(f()){case .ok(n):{n}case .err(_):{1}};}', 'XABO', 1, 0),
        ('partial-first', 'resource S{i32 n;cleanup{mark(this.n);}}resource R{i32 n;S a;cleanup{mark(82);}}fn Result<i32,bool> bad(){return .err(true);}fn Result<i32,bool> f(){let R r=R{.n=bad()?,.a=S{.n=65}};return .ok(0);}fn i32 main(){return switch(f()){case .ok(n):{n}case .err(_):{1}};}', '', 1, 0),
        ('partial-nested', 'resource S{i32 n;cleanup{mark(this.n);}}resource Inner{S s;cleanup{mark(73);}}resource R{Inner a;i32 n;cleanup{mark(82);}}fn Result<i32,bool> bad(){return .err(true);}fn Result<i32,bool> f(){let R r=R{.a=Inner{.s=S{.n=65}},.n=bad()?};return .ok(0);}fn i32 main(){return switch(f()){case .ok(n):{n}case .err(_):{1}};}', 'IA', 1, 0),
        ('later-argument-fails', 'resource S{i32 n;cleanup{mark(this.n);}}fn void take(S a,S b,i32 n){mark(84);}fn Result<i32,bool> bad(){mark(88);return .err(true);}fn Result<i32,bool> f(){take(S{.n=65},S{.n=66},bad()?);return .ok(0);}fn i32 main(){return switch(f()){case .ok(n):{n}case .err(_):{1}};}', 'XBA', 1, 0),
        ('nested-move', 'resource S{i32 n;cleanup{mark(this.n);}}resource R{S a;cleanup{mark(82);}}fn R make(){return R{.a=S{.n=65}};}fn i32 main(){let R a=make();let R b=a;return 0;}', 'RA', 0, 0),
        ('temporary-expression', 'resource S{i32 n;cleanup{mark(this.n);}}fn i32 main(){discard S{.n=65};mark(88);return 0;}', 'AX', 0, 0),
    ],
    25: [
        ('overwrite-live', 'resource R{i32 n;cleanup{mark(this.n);}}fn R make(){mark(88);return R{.n=66};}fn i32 main(){var R r=R{.n=65};r=make();mark(89);return r.n;}', 'XAYB', 66, 0),
        ('overwrite-donor', 'resource R{i32 n;cleanup{mark(this.n);}}fn i32 main(){var R r=R{.n=65};let R donor=R{.n=66};r=donor;return 0;}', 'AB', 0, 0),
        ('overwrite-self', 'resource R{i32 n;cleanup{mark(this.n);}}fn i32 main(){var R r=R{.n=65};r=r;return 0;}', 'A', 0, 0),
        ('overwrite-dead', 'resource R{i32 n;cleanup{mark(this.n);}}fn void take(R r){mark(88);}fn i32 main(){var R r=R{.n=65};take(r);r=R{.n=66};return 0;}', 'XAB', 0, 0),
        ('overwrite-error', 'resource R{i32 n;cleanup{mark(this.n);}}fn Result<i32,bool> next(){mark(88);return .err(true);}fn Result<i32,bool> f(){var R r=R{.n=65};r=R{.n=next()?};return .ok(r.n);}fn i32 main(){return switch(f()){case .ok(n):{n}case .err(_):{1}};}', 'XA', 1, 0),
        ('overwrite-success', 'resource R{i32 n;cleanup{mark(this.n);}}fn Result<i32,bool> next(){mark(88);return .ok(66);}fn Result<i32,bool> f(){var R r=R{.n=65};r=R{.n=next()?};return .ok(r.n);}fn i32 main(){return switch(f()){case .ok(n):{n}case .err(_):{1}};}', 'XAB', 66, 0),
    ],
    24: [
        ('loop-fallthrough', 'resource R{i32 n;cleanup{mark(this.n);}}fn i32 main(){let R o=R{.n=79};var i32 n=0;while(n<3){let R r=R{.n=65};n+=1;}return 0;}', 'AAAO', 0, 0),
        ('loop-continue-break', 'resource R{i32 n;cleanup{mark(this.n);}}fn i32 main(){let R o=R{.n=79};var i32 n=0;while(n<3){let R r=R{.n=65};n+=1;{let R b=R{.n=66};if(n==1){continue;}else{break;};};}mark(88);return 0;}', 'BABAXO', 0, 0),
        ('nearest-loop', 'resource R{i32 n;cleanup{mark(this.n);}}fn i32 main(){let R o=R{.n=79};var i32 n=0;while(n<2){let R a=R{.n=65};while(true){let R b=R{.n=66};break;}mark(88);n+=1;}return 0;}', 'BXABXAO', 0, 0),
        ('for-post-order', 'resource R{i32 n;cleanup{mark(this.n);}}fn i32 main(){for(var i32 n=0;n<2;n+=1){let R a=R{.n=65+n};continue;}return 0;}', 'AB', 0, 0),
        ('loop-return', 'resource R{i32 n;cleanup{mark(this.n);}}fn i32 main(){let R o=R{.n=79};while(true){let R a=R{.n=65};return 7;}return 0;}', 'AO', 7, 0),
        ('loop-zero', 'resource R{i32 n;cleanup{mark(this.n);}}fn i32 main(){let R o=R{.n=79};while(false){let R a=R{.n=65};break;}return 0;}', 'O', 0, 0),
    ]
}
REJECTIONS = {
    22: [
        ('break-live', 'resource R{cleanup{}}fn i32 main(){while(true){let R r=R{};break;}return 0;}'),
        ('overwrite-live', 'resource R{cleanup{}}fn i32 main(){{var R r=R{};r=R{};};return 0;}'),
        ('nested-field', 'resource S{cleanup{}}resource R{S s;cleanup{}}fn i32 main(){{let R r=R{.s=S{}};};return 0;}'),
    ],
    23: [
        ('break-live', 'resource R{cleanup{}}fn i32 main(){while(true){let R r=R{};break;}return 0;}'),
        ('overwrite-live', 'resource R{cleanup{}}fn i32 main(){var R r=R{};r=R{};return 0;}'),
        ('nested-field', 'resource S{cleanup{}}resource R{S s;cleanup{}}fn i32 main(){let R r=R{.s=S{}};return 0;}'),
    ],
    27: [
        ('partial-field-move', 'resource S{cleanup{}}resource R{S s;cleanup{}}fn void take(S s){}fn i32 main(){let R r=R{.s=S{}};take(r.s);return 0;}'),
        ('tag-use-after-move', 'resource R{cleanup{}}fn void take(Option<R> r){}fn i32 main(){let Option<R> r=.some(R{});take(r);take(r);return 0;}'),
    ],
    26: [
        ('owned-tag', 'resource S{cleanup{}}fn i32 main(){let Option<S> s=.some(S{});return 0;}'),
        ('field-transfer', 'resource S{cleanup{}}resource R{S s;cleanup{}}fn void take(S s){}fn i32 main(){let R r=R{.s=S{}};take(r.s);return 0;}'),
    ],
    25: [
        ('nested-field', 'resource S{cleanup{}}resource R{S s;cleanup{}}fn i32 main(){let R r=R{.s=S{}};return 0;}'),
        ('maybe-live', 'resource R{cleanup{}}fn void take(R r){}fn void f(bool yes){var R r=R{};if(yes){take(r);};r=R{};}fn i32 main(){f(true);return 0;}'),
    ],
    24: [
        ('overwrite-live', 'resource R{cleanup{}}fn i32 main(){var R r=R{};r=R{};return 0;}'),
        ('nested-field', 'resource S{cleanup{}}resource R{S s;cleanup{}}fn i32 main(){let R r=R{.s=S{}};return 0;}'),
    ]
}

CASES[28] = [
 ('burn-no-glue', '@abilities(store,key)resource R{i32 n;cleanup{mark(this.n);}}fn i32 main(){let R r=R{.n=65};burn r;mark(88);return 7;}', 'X', 7, 0),
 ('burn-parameter', '@abilities(store)resource R{i32 n;cleanup{mark(this.n);}}fn void destroy(R r){burn r;mark(88);}fn i32 main(){destroy(R{.n=65});return 0;}', 'X', 0, 0),
 ('burn-moved-local', '@abilities(store)resource R{cleanup{mark(65);}}fn i32 main(){let R a=R{};let R b=a;burn b;return 0;}', '', 0, 0),
 ('explicit-drop', '@abilities(drop,store,key)resource R{cleanup{mark(65);}}fn i32 main(){let R r=R{};return 0;}', 'A', 0, 0),
 ('burn-branches', '@abilities(store)resource R{cleanup{mark(65);}}fn void destroy(R r,bool yes){if(yes){burn r;}else{burn r;};}fn i32 main(){destroy(R{},true);destroy(R{},false);return 0;}', '', 0, 0),
 ('burn-loop', '@abilities(store)resource R{cleanup{mark(65);}}fn i32 main(){var i32 n=0;while(n<2){let R r=R{};burn r;n+=1;}return n;}', '', 2, 0),
]
REJECTIONS[28] = [
 ('implicit-no-drop','@abilities(store)resource R{cleanup{}}fn i32 main(){let R r=R{};return 0;}', 'E1108'),
 ('overwrite-no-drop','@abilities(store)resource R{cleanup{}}fn i32 main(){var R r=R{};r=R{};burn r;return 0;}', 'E1108'),
 ('nested-no-drop','@abilities(store)resource S{cleanup{}}resource R{S s;cleanup{}}fn i32 main(){let R r=R{.s=S{}};return 0;}', 'E1108'),
 ('option-no-drop','@abilities(store)resource S{cleanup{}}fn i32 main(){let Option<S> s=.some(S{});return 0;}', 'E1108'),
 ('burn-drop','resource R{cleanup{}}fn i32 main(){let R r=R{};burn r;return 0;}', 'E1109'),
 ('burn-nested','@abilities(store)resource S{cleanup{}}@abilities(store)resource R{S s;cleanup{}}fn i32 main(){let R r=R{.s=S{}};burn r;return 0;}', 'E1109'),
 ('burn-twice','@abilities(store)resource R{cleanup{}}fn i32 main(){let R r=R{};burn r;burn r;return 0;}', 'E1101'),
 ('burn-temporary','@abilities(store)resource R{cleanup{}}fn i32 main(){burn R{};return 0;}', 'E1109'),
 ('burn-wrapper','@abilities(store)resource R{cleanup{}}fn i32 main(){let Option<R> r=.some(R{});burn r;return 0;}', 'E1109'),
 ('bad-ability','@abilities(copy)resource R{cleanup{}}fn i32 main(){return 0;}', 'E1107'),
 ('duplicate-ability','@abilities(drop,drop)resource R{cleanup{}}fn i32 main(){return 0;}', 'E1107'),
 ('key-without-store','@abilities(key)resource R{cleanup{}}fn i32 main(){return 0;}', 'E1107'),
]

CASES[29] = [
 ('defer-scope','resource R{i32 n;cleanup{mark(this.n);}}fn i32 main(){{let R r=R{.n=65};defer mark(66);defer mark(67);mark(88);};return 0;}', 'XCBA', 0, 0),
 ('defer-return-capture','resource R{cleanup{mark(65);}}fn i32 main(){let R r=R{};var i32 n=66;defer mark(n);n=67;defer mark(n);return n;}', 'CBA', 67, 0),
 ('defer-crossed','resource R{i32 n;cleanup{mark(this.n);}}fn i32 main(){let R outer=R{.n=79};defer mark(68);{let R inner=R{.n=65};defer mark(66);return 7;};return 0;}', 'BDAO', 7, 0),
 ('defer-loop','resource R{cleanup{mark(65);}}fn i32 main(){defer mark(79);var i32 n=0;while(n<2){let R r=R{};defer mark(66);n+=1;if(n==1){continue;}break;}return n;}', 'BABAO', 2, 0),
 ('defer-question-error','resource R{cleanup{mark(65);}}fn Result<i32,bool> bad(){mark(88);return .err(true);}fn Result<i32,bool> f(){let R r=R{};defer mark(66);let i32 n=bad()?;return .ok(n);}fn i32 main(){return switch(f()){case .ok(n):{n}case .err(_):{1}};}', 'XBA', 1, 0),
 ('defer-question-success','resource R{cleanup{mark(65);}}fn Result<i32,bool> good(){mark(88);return .ok(7);}fn Result<i32,bool> f(){let R r=R{};defer mark(66);let i32 n=good()?;mark(89);return .ok(n);}fn i32 main(){return switch(f()){case .ok(n):{n}case .err(_):{1}};}', 'XYBA', 7, 0),
 ('defer-argument-order','resource R{cleanup{mark(65);}}fn i32 arg(i32 n){mark(n);return n+2;}fn void final(i32 a,i32 b){mark(a);mark(b);}fn i32 main(){let R r=R{};defer final(arg(66),arg(67));mark(88);return 0;}', 'BCXDEA', 0, 0),
 ('defer-in-cleanup','resource R{cleanup{defer mark(65);mark(66);}}fn i32 main(){let R r=R{};defer mark(67);return 0;}', 'CBA', 0, 0),
]
REJECTIONS[29] = [
 ('defer-resource','resource R{cleanup{}}fn void take(R r){}fn i32 main(){let R r=R{};defer take(r);return 0;}'),
 ('defer-aggregate','struct S{i32 n;}fn void take(S s){}fn i32 main(){defer take(S{.n=1});return 0;}'),
 ('defer-result','fn i32 give(){return 1;}fn i32 main(){defer give();return 0;}'),
]

CASES[30] = [
 ('trap-assertion','resource R{cleanup{mark(65);}}fn i32 main(){let R r=R{};defer mark(66);mark(88);assert(false);}', 'X', 0, 5),
 ('trap-overflow','resource R{cleanup{mark(65);}}fn i32 main(){let R r=R{};defer mark(66);mark(88);var i32 n=2147483647;return n+1;}', 'X', 0, 1),
 ('trap-division','resource R{cleanup{mark(65);}}fn i32 main(){let R r=R{};defer mark(66);mark(88);var i32 zero=0;return 7/zero;}', 'X', 0, 2),
 ('trap-deferred-call','resource R{cleanup{mark(65);}}fn void fail(){mark(88);assert(false);}fn i32 main(){let R r=R{};defer mark(66);defer fail();return 0;}', 'X', 0, 5),
 ('trap-cleanup-stops','resource R{i32 n;cleanup{mark(this.n);assert(false);}}fn i32 main(){let R a=R{.n=65};let R b=R{.n=66};defer mark(88);return 0;}', 'XB', 0, 5),
 ('trap-partial-constructor','resource S{cleanup{mark(65);}}resource R{S s;i32 n;cleanup{mark(66);}}fn i32 main(){var i32 n=2147483647;let R r=R{.s=S{},.n=n+1};return 0;}', '', 0, 1),
 ('trap-linear-obligation','@abilities(store)resource R{cleanup{mark(65);}}fn i32 main(){let R r=R{};mark(88);assert(false);}', 'X', 0, 5),
]

CASES[31] = [
 ('ref-i32','fn i32 read(ref i32 p){return *p;}fn i32 main(){let i32 n=7;let ref i32 p=ref n;return read(p);}', '', 7, 0),
 ('ref-wide','fn i64 read(ref i64 p){return *p;}fn i32 main(){let i64 n=4000000000;let ref i64 p=ref n;assert(read(p)==4000000000);return 0;}', '', 0, 0),
 ('ref-bool','fn bool read(ref bool p){return *p;}fn i32 main(){let bool n=true;let ref bool p=ref n;return if(read(p)){1}else{0};}', '', 1, 0),
 ('ref-struct','struct S{i32 n;}fn i32 read(ref S p){return p.n;}fn i32 main(){let S s=S{.n=7};let ref S p=ref s;return read(p);}', '', 7, 0),
 ('ref-struct-copy','struct S{i32 n;}fn S read(ref S p){return *p;}fn i32 main(){let S s=S{.n=7};let ref S p=ref s;let S copy=read(p);return copy.n;}', '', 7, 0),
 ('ref-resource','resource R{i32 n;cleanup{mark(this.n);}}fn i32 read(ref R p){return p.n;}fn i32 main(){let R r=R{.n=65};let ref R p=ref r;let i32 n=read(p);return n;}', 'A', 65, 0),
 ('ref-mut-read','fn i32 read(ref mut i32 p){return *p;}fn i32 main(){var i32 n=7;let ref mut i32 p=ref mut n;return read(p);}', '', 7, 0),
]
REJECTIONS[31] = [
 ('ref-local-return','fn ref i32 bad(){let i32 n=7;return ref n;}fn i32 main(){return 0;}'),
 ('ref-value-param-return','fn ref i32 bad(i32 n){return ref n;}fn i32 main(){return 0;}'),
 ('ref-deref-integer','fn i32 main(){let i32 n=7;return *n;}'),
 ('ref-immutable-mut','fn i32 main(){let i32 n=7;let ref mut i32 p=ref mut n;return *p;}'),
 ('ref-owner-write','fn i32 main(){var i32 n=7;let ref i32 p=ref n;n=9;return *p;}'),
 ('ref-owner-move','resource R{i32 n;cleanup{}}fn void take(R r){}fn i32 main(){let R r=R{.n=7};let ref R p=ref r;take(r);return p.n;}'),
 ('ref-field-storage','struct S{ref i32 p;}fn i32 main(){return 0;}'),
]

CASES[32]=[
 ('shared-field','struct S{i32 n;}fn i32 main(){let S s=S{.n=7};let ref i32 p=ref s.n;return *p;}', '',7,0),
 ('shared-nested','struct S{i32 n;}struct T{S s;}fn ref i32 field(ref T t){return ref t.s.n;}fn i32 main(){let T t=T{.s=S{.n=7}};return *field(ref t);}', '',7,0),
 ('shared-index','fn i32 main(){let i64[2] a=[4000000000,7];let ref i64 p=ref a[0];assert(*p==4000000000);return 0;}', '',0,0),
 ('shared-index-field','struct S{i32 n;}fn i32 main(){let S[2] a=[S{.n=7};2];let ref i32 p=ref a[1].n;return *p;}', '',7,0),
 ('shared-resource-field','resource R{i32 n;cleanup{mark(this.n);}}fn i32 main(){let R r=R{.n=65};let ref i32 p=ref r.n;return *p;}', 'A',65,0),
 ('shared-index-bounds','fn i32 main(){let i32[2] a=[1,2];let usize i=2;let ref i32 p=ref a[i];return *p;}', '',0,6),
]
REJECTIONS[32]=[
 ('shared-field-owner-write','struct S{i32 n;}fn i32 main(){var S s=S{.n=7};let ref i32 p=ref s.n;s.n=9;return *p;}', 'E2101'),
 ('shared-index-owner-write','fn i32 main(){var i32[2] a=[1,2];let ref i32 p=ref a[0];a[1]=9;return *p;}', 'E2101'),
 ('shared-field-escape','struct S{i32 n;}fn ref i32 bad(){let S s=S{.n=7};return ref s.n;}fn i32 main(){return 0;}', 'E2102'),
 ('shared-field-temporary','struct S{i32 n;}fn S make(){return S{.n=7};}fn i32 main(){let ref i32 p=ref make().n;return *p;}'),
]

CASES[33]=[
 ('mut-scalar-store','fn void edit(ref mut i64 p){*p=*p+1;}fn i32 main(){var i64 n=4000000000;edit(ref mut n);assert(n==4000000001);return 0;}', '',0,0),
 ('mut-field','struct S{i32 n;i32 other;}fn void edit(ref mut i32 p){*p=9;}fn i32 main(){var S s=S{.n=7,.other=3};edit(ref mut s.n);return s.n+s.other;}', '',12,0),
 ('mut-nested','struct S{i32 n;}struct T{S s;}fn void edit(ref mut i32 p){*p=9;}fn i32 main(){var T t=T{.s=S{.n=7}};edit(ref mut t.s.n);return t.s.n;}', '',9,0),
 ('mut-index','fn void edit(ref mut i32 p){*p=9;}fn i32 main(){var i32[2] a=[1,2];edit(ref mut a[1]);return a[0]+a[1];}', '',10,0),
 ('mut-reference-field','struct S{i32 n;}fn void edit(ref mut S s){s.n+=2;}fn i32 main(){var S s=S{.n=7};edit(ref mut s);return s.n;}', '',9,0),
 ('mut-resource-reference-field','resource R{i32 n;cleanup{mark(this.n);}}fn void edit(ref mut R p){p.n+=1;}fn i32 main(){var R r=R{.n=65};edit(ref mut r);return r.n;}', 'B',66,0),
]
REJECTIONS[33]=[
 ('mut-shared-write','fn i32 main(){let i32 n=7;let ref i32 p=ref n;*p=9;return *p;}'),
 ('mut-field-immutable','struct S{i32 n;}fn i32 main(){let S s=S{.n=7};let ref mut i32 p=ref mut s.n;return *p;}'),
 ('mut-field-siblings','struct S{i32 a;i32 b;}fn i32 main(){var S s=S{.a=1,.b=2};let ref mut i32 a=ref mut s.a;let ref mut i32 b=ref mut s.b;return *a+*b;}', 'E2101'),
 ('mut-resource-field-address','resource R{i32 n;cleanup{}}fn i32 main(){var R r=R{.n=7};let ref mut i32 p=ref mut r.n;return *p;}'),
 ('mut-deref-compound','fn i32 main(){var i32 n=7;let ref mut i32 p=ref mut n;*p+=1;return *p;}'),
]

CASES[34]=[
 ('shared-call-from-mut','fn i32 read(ref i32 p){return *p;}fn i32 main(){var i32 n=7;let ref mut i32 p=ref mut n;let i32 value=read(p);*p=9;return value+n;}', '',16,0),
 ('shared-child-resumes','fn ref i32 share(ref i32 p){return p;}fn i32 main(){var i32 n=7;let ref mut i32 p=ref mut n;let ref i32 child=share(p);let ref i32 alias=child;let i32 value=*alias;*p=9;return value+n;}', '',16,0),
 ('shared-field-through-mut','struct S{i32 n;}fn i32 main(){var S s=S{.n=7};let ref mut S p=ref mut s;let ref i32 child=ref p.n;let i32 value=p.n+*child;p.n=9;return value+s.n;}', '',23,0),
 ('owner-after-last-use','fn i32 main(){var i32 n=7;let ref i32 p=ref n;let i32 value=*p;n=9;return value+n;}', '',16,0),
]
REJECTIONS[34]=[
 ('shared-return-parent-write','fn ref i32 share(ref i32 p){return p;}fn i32 main(){var i32 n=7;let ref mut i32 p=ref mut n;let ref i32 child=share(p);*p=9;return *child;}', 'E2101'),
 ('shared-return-parent-call','fn ref i32 share(ref i32 p){return p;}fn void edit(ref mut i32 p){*p=9;}fn i32 main(){var i32 n=7;let ref mut i32 p=ref mut n;let ref i32 child=share(p);edit(p);return *child;}', 'E2101'),
 ('shared-implicit-binding','fn i32 main(){var i32 n=7;let ref mut i32 p=ref mut n;let ref i32 child=p;return *child;}'),
]

CASES[35]=[
 ('mut-child-resume','fn i32 main(){var i32 n=7;let ref mut i32 p=ref mut n;let ref mut i32 child=ref mut *p;*child=8;let i32 value=*child;*p=9;return value+n;}', '',17,0),
 ('mut-descendant-resume','fn i32 main(){var i32 n=7;let ref mut i32 p=ref mut n;let ref mut i32 child=ref mut *p;let ref mut i32 grand=ref mut *child;*grand=8;let i32 value=*grand;*child=9;*p=10;return value+n;}', '',18,0),
 ('mut-call-return-resume','fn ref mut i32 id(ref mut i32 p){return p;}fn i32 main(){var i32 n=7;let ref mut i32 p=ref mut n;let ref mut i32 child=id(p);*child=8;let i32 value=*child;*p=9;return value+n;}', '',17,0),
 ('shared-explicit-deref','fn i32 main(){var i32 n=7;let ref mut i32 p=ref mut n;let ref i32 child=ref *p;let i32 value=*child;*p=9;return value+n;}', '',16,0),
]
REJECTIONS[35]=[
 ('mut-child-parent-write','fn i32 main(){var i32 n=7;let ref mut i32 p=ref mut n;let ref mut i32 child=ref mut *p;*p=9;return *child;}', 'E2101'),
 ('mut-child-parent-read','fn i32 main(){var i32 n=7;let ref mut i32 p=ref mut n;let ref mut i32 child=ref mut *p;let i32 value=*p;return value+*child;}', 'E2101'),
 ('mut-child-parent-alias','fn i32 main(){var i32 n=7;let ref mut i32 p=ref mut n;let ref mut i32 alias=p;let ref mut i32 child=ref mut *p;*alias=9;return *child;}', 'E2101'),
 ('mut-from-shared-deref','fn i32 main(){var i32 n=7;var ref i32 p=ref n;let ref mut i32 child=ref mut *p;return *child;}'),
]

CASES[36]=[
 ('args-source-order','fn i32 read(ref i32 p){return *p;}fn i32 value(i32 n){mark(n);return n-65;}fn i32 pair(ref i32 p,i32 n){return *p+n;}fn i32 main(){let i32 n=7;return pair(ref n,value(65));}', 'A',7,0),
 ('arg-read-before-edit','fn i32 pair(i32 a,i32 b){return a+b;}fn i32 edit(ref mut i32 p){*p=9;return 1;}fn i32 main(){var i32 n=7;let ref mut i32 p=ref mut n;return pair(*p,edit(p));}', '',8,0),
 ('nested-returned-arg','fn ref i32 id(ref i32 p){return p;}fn i32 pair(ref i32 p,i32 n){return *p+n;}fn i32 main(){let i32 n=7;return pair(id(ref n),1);}', '',8,0),
 ('distinct-arg-owners','fn i32 pair(ref i32 p,ref mut i32 q){*q=9;return *p+*q;}fn i32 main(){var i32 a=7;var i32 b=1;return pair(ref a,ref mut b);}', '',16,0),
]
REJECTIONS[36]=[
 ('arg-shared-then-mutable','fn i32 pair(ref i32 a,ref mut i32 b){return *a+*b;}fn i32 main(){var i32 n=7;let ref mut i32 p=ref mut n;return pair(p,p);}', 'E2101'),
 ('arg-mutable-then-shared','fn i32 pair(ref mut i32 a,ref i32 b){return *a+*b;}fn i32 main(){var i32 n=7;let ref mut i32 p=ref mut n;return pair(p,p);}', 'E2101'),
 ('arg-first-then-nested-edit','fn i32 pair(ref i32 p,i32 n){return *p+n;}fn i32 edit(ref mut i32 p){*p=9;return 1;}fn i32 main(){var i32 n=7;let ref mut i32 p=ref mut n;return pair(p,edit(p));}', 'E2101'),
 ('arg-returned-then-edit','fn ref i32 id(ref i32 p){return p;}fn i32 pair(ref i32 p,i32 n){return *p+n;}fn i32 edit(ref mut i32 p){*p=9;return 1;}fn i32 main(){var i32 n=7;let ref mut i32 p=ref mut n;return pair(id(p),edit(p));}', 'E2101'),
 ('arg-exclusive-parent-read','fn i32 pair(ref mut i32 p,i32 n){return *p+n;}fn i32 main(){var i32 n=7;let ref mut i32 p=ref mut n;return pair(p,*p);}', 'E2101'),
]

CASES[37]=[
 ('origin-forward-reorder','fn ref i32 wrap(ref i32 a,ref i32 b){return choose(b,a);}fn ref i32 choose(ref i32 a,ref i32 b){return b;}fn i32 main(){var i32 a=7;var i32 b=1;let ref i32 p=wrap(ref a,ref b);b=9;return *p+b;}', '',16,0),
 ('origin-wrapper-chain','fn ref i32 one(ref i32 a){return two(a);}fn ref i32 two(ref i32 a){return three(a);}fn ref i32 three(ref i32 a){return a;}fn i32 main(){let i32 n=7;return *one(ref n);}', '',7,0),
 ('origin-switch','enum E{A;B;}fn ref i32 choose(ref i32 p,E e){return switch(e){case .A:{p}case .B:{p}};}fn i32 main(){let i32 n=7;return *choose(ref n,.B);}', '',7,0),
 ('origin-view-return','fn ref u8[] tail(ref u8[] p){return p[1..];}fn i32 main(){let u8[2] a=[1,7];let ref u8[] p=ref a;let ref u8[] q=tail(p);assert(q.len==1 && q[0]==7);return 0;}', '',0,0),
]
REJECTIONS[37]=[
 ('origin-selected-owner-write','fn ref i32 second(ref i32 a,ref i32 b){return b;}fn i32 main(){var i32 a=7;var i32 b=1;let ref i32 p=second(ref a,ref b);b=9;return *p;}', 'E2101'),
 ('origin-wrapper-local-escape','fn ref i32 id(ref i32 p){return p;}fn ref i32 bad(){let i32 n=7;return id(ref n);}fn i32 main(){return 0;}', 'E2102'),
 ('origin-two-inputs','fn ref i32 choose(ref i32 a,ref i32 b,bool yes){return if(yes){a}else{b};}fn i32 main(){return 0;}', 'E2103'),
]

CASES[38]=[
 ('recursive-single-origin','fn ref i32 walk(ref i32 p,i32 n){if(n==0){return p;};return walk(p,n-1);}fn i32 main(){let i32 n=7;return *walk(ref n,3);}', '',7,0),
 ('mutual-single-origin','fn ref i32 a(ref i32 p,i32 n){if(n==0){return p;};return b(p,n-1);}fn ref i32 b(ref i32 p,i32 n){return a(p,n);}fn i32 main(){let i32 n=7;return *a(ref n,3);}', '',7,0),
 ('static-shortened-origin','fn ref u8[] choose(ref u8[] p,bool yes){return if(yes){p}else{b"xy"};}fn i32 main(){let u8[2] a=[1,7];let ref u8[] p=ref a;let ref u8[] q=choose(p,false);assert(q[0]==120);return 0;}', '',0,0),
]
REJECTIONS[38]=[
 ('recursive-unproven','fn ref i32 a(ref i32 p){return b(p);}fn ref i32 b(ref i32 p){return a(p);}fn i32 main(){return 0;}', 'E2103'),
 ('recursive-swaps-origins','fn ref i32 a(ref i32 p,ref i32 q,i32 n){if(n==0){return p;};return a(q,p,n-1);}fn i32 main(){return 0;}', 'E2103'),
 ('static-local-escape','fn ref u8[] choose(bool yes){let u8[2] a=[1,2];let ref u8[] p=ref a;return if(yes){p}else{b"xy"};}fn i32 main(){return 0;}', 'E2102'),
]

CASES[39]=[
 ('option-mut-none','fn Option<ref mut i32> no(){return .none;}fn i32 main(){return switch(no()){case .some(p):{*p}case .none:{7}};}', '',7,0),
 ('option-ref-return','fn Option<ref i32> id(ref i32 p){return .some(p);}fn i32 main(){let i32 n=7;return switch(id(ref n)){case .some(p):{*p}case .none:{0}};}', '',7,0),
 ('option-ref-none','fn Option<ref i32> id(){return .none;}fn i32 main(){return switch(id()){case .some(p):{*p}case .none:{7}};}', '',7,0),
 ('option-nested-ref','fn Option<Option<ref i32>> id(ref i32 p){return .some(.some(p));}fn i32 main(){let i32 n=7;return switch(id(ref n)){case .some(p):{switch(p){case .some(q):{*q}case .none:{0}}}case .none:{0}};}', '',7,0),
 ('result-ref-success','fn Result<ref i32,bool> id(ref i32 p){return .ok(p);}fn Result<i32,bool> f(){let i32 n=7;let ref i32 p=id(ref n)?;return .ok(*p);}fn i32 main(){return switch(f()){case .ok(n):{n}case .err(_):{0}};}', '',7,0),
 ('result-ref-error','fn Result<i32,ref i32> bad(ref i32 p){return .err(p);}fn Result<i32,ref i32> f(ref i32 p){let i32 n=bad(p)?;return .ok(n);}fn i32 main(){let i32 n=7;return switch(f(ref n)){case .ok(n):{n}case .err(p):{*p}};}', '',7,0),
 ('option-mut-reference','fn Option<ref mut i32> id(ref mut i32 p){return .some(p);}fn i32 main(){var i32 n=7;let ref mut i32 p=ref mut n;let Option<ref mut i32> x=id(p);discard switch(x){case .some(q):{*q=8;}case .none:{}};*p=9;return n;}', '',9,0),
]
REJECTIONS[39]=[
 ('option-local-escape','fn Option<ref i32> bad(){let i32 n=7;return .some(ref n);}fn i32 main(){return 0;}', 'E2102'),
 ('option-holds-owner','fn Option<ref i32> id(ref i32 p){return .some(p);}fn i32 main(){var i32 n=7;let Option<ref i32> x=id(ref n);n=9;return switch(x){case .some(p):{*p}case .none:{0}};}', 'E2101'),
 ('result-error-local-escape','fn Result<i32,ref i32> bad(ref i32 p){return .err(p);}fn Result<i32,ref i32> f(){let i32 n=7;let i32 value=bad(ref n)?;return .ok(value);}fn i32 main(){return 0;}', 'E2102'),
 ('struct-tagged-reference','struct S{Option<ref i32> p;}fn i32 main(){return 0;}'),
 ('two-borrow-leaves','fn Result<ref i32,ref u8[]> bad(ref i32 p,ref u8[] q){return .ok(p);}fn i32 main(){return 0;}'),
]

def sha(path): return hashlib.sha256(path.read_bytes()).hexdigest()
def save(path, value): path.write_text(json.dumps(value, ensure_ascii=False, indent=2)+'\n', encoding='utf-8', newline='\n')

def main():
    ap = argparse.ArgumentParser()
    ap.add_argument('--unit', type=int, required=True)
    ap.add_argument('--images', nargs='+', required=True)
    ap.add_argument('--native-runner', required=True)
    ap.add_argument('--receipt', required=True)
    args = ap.parse_args()
    images = [Path(p).resolve() for p in args.images]
    runner = Path(args.native_runner).resolve()
    receipt = Path(args.receipt).resolve(); receipt.parent.mkdir(parents=True, exist_ok=True)
    rows, fixtures, jobs, native, units = [], [], [], [], []
    passed = False
    def run(argv, expected=0, timeout=120):
        argv = list(map(str, argv))
        p = subprocess.run(argv, cwd=ROOT, capture_output=True, encoding='utf-8', timeout=timeout)
        row = dict(argv=argv, cwd=str(ROOT), exit=p.returncode, stdout=p.stdout, stderr=p.stderr)
        rows.append(row); save(receipt.parent/'raw.json', rows)
        assert p.returncode == expected, row
        return row
    try:
        for name, source, trace, value, trap in CASES[args.unit]:
            path = receipt.parent/(name+'.fy'); path.write_text(PREFIX+source, encoding='utf-8', newline='\n')
            digest = sha(path); outputs = []
            fixtures.append(dict(name=name, inputHex=path.read_bytes().hex(), sha256=digest, expectedTrace=trace, expectedValue=value, expectedTrap=trap))
            for image in images:
                for mode in ('--check-types-file', '--hir-file', '--mir-file', '--c-file'):
                    row = run([image, mode, path]); assert row['stdout'] and not row['stderr'], row
                    if mode == '--c-file': outputs.append(row['stdout'])
                assert sha(path) == digest
            assert all(value==outputs[0] for value in outputs), name
            plain = receipt.parent/(name+'.c'); plain.write_text(outputs[0], encoding='utf-8', newline='\n')
            signature = 'void fy_fn_1(int32_t fy_l_1){\n'
            assert outputs[0].count(signature) == 1, name
            # Observation only: record entry to the existing emitted mark body.
            # Neither cleanup registration nor emitted control flow is replaced.
            code = outputs[0].replace(signature, signature+'    (void)fputc((unsigned char)fy_l_1,stdout);(void)fflush(stdout);\n', 1)
            wrapper = '#define FY_CANDIDATE_NO_MAIN\n#include <stdio.h>\n#include <inttypes.h>\n'
            observer = 'int main(void){int32_t (*volatile entry)(void)=fy_candidate_entry;int32_t result=entry();fprintf(stderr,"fairyc-candidate-result/v1:%" PRId32 "\\n",result);return 0;}\n'
            observed = receipt.parent/(name+'.observed.c'); observed.write_text(wrapper+code+observer, encoding='utf-8', newline='\n')
            jobs.append(dict(name=name, source=str(observed), observe=not trap))
            units.append(dict(name=name, emittedSHA256=sha(plain), observedSHA256=sha(observed)))
        for rejection in REJECTIONS.get(args.unit, []):
            name, source = rejection[:2]
            diagnostic = rejection[2] if len(rejection)>2 else None
            path = receipt.parent/(name+'.fy'); path.write_text(PREFIX+source, encoding='utf-8', newline='\n')
            fixtures.append(dict(name=name, inputHex=path.read_bytes().hex(), sha256=sha(path), rejectedAt='MIR/C'))
            for image in images:
                for mode in ('--mir-file', '--c-file'):
                    row = run([image, mode, path], 2); assert not row['stdout'] and row['stderr'], row
                    if diagnostic: assert diagnostic in row['stderr'], row
        request = receipt.parent/'native-request.json'; report = receipt.parent/'native.json'
        save(request, dict(schema='fairyc.candidate-c-native-request/v1', output=str(receipt.parent/'native'), jobs=jobs))
        run([runner, request, report], timeout=300)
        native = json.loads(report.read_text(encoding='utf-8'))
        expected = {name: (trace, value, trap) for name, _, trace, value, trap in CASES[args.unit]}
        assert len(native) == 2*len(jobs)
        for row in native:
            trace, value, trap = expected[row['name']]
            assert not row.get('error') and row['trap'] == trap and row['stdout'] == trace, row
            if not trap: assert row['exit'] == 0 and row['value'] == value, row
        passed = True
    finally:
        save(receipt, dict(schema='fairyc.resource-lifetime-native/v1', unit=args.unit, passed=passed,
            images=[dict(path=str(p), sha256=sha(p)) for p in images], nativeRunner=dict(path=str(runner), sha256=sha(runner)),
            fixtures=fixtures, units=units, cli=rows, native=native,
            sources=[dict(path=str(p.relative_to(REPO)), sha256=sha(p)) for p in sorted((ROOT/'src').glob('*.fy'))],
            observer=dict(path='examples/lexer_candidate/verify_resource_lifetime.py', sha256=sha(Path(__file__))),
            lock=dict(sha256=sha(ROOT/'fairy.lock'), packages=json.loads((ROOT/'fairy.lock').read_text())['packages'])))
    print(f'PASS unit {args.unit}: {len(fixtures)} fixtures, {len(native)} native runs')

if __name__ == '__main__': main()
