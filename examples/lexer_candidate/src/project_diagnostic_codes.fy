module project_diagnostic_codes;

// Catalog/name lookup codes overlap internal capacity and type statuses.
// Translate them only at the phase that owns their resolution meaning.
pub fn u32 code(u32 phase,u32 stage,u32 raw){
    if(phase==5 && stage==5 && (raw==2005 || raw==2006)){return 4503;}
    if(phase==5 && stage==2 && raw==4501){return 4501;}
    if(phase==5 && (stage==3 || stage==4)){if(raw==13){return 103;}if(raw==101 || raw==4501 || raw==4502){return raw;}}
    if(phase==1 && raw==2){return 102;}
    if(phase==3 && stage==2 && (raw==4 || raw==5)){return 101;}
    if(phase==3 && stage==3 && raw==1){return 101;}
    if(phase>=1 && phase<=3 && summary(raw).len!=0){return raw;}
    return 0;
}
// Only named language diagnostics have a public E-code. Unknown subset,
// allocation, capacity and invariant failures retain their existing fallback.
pub fn ref u8[] summary(u32 code){
    if(code==101){return b"unresolved name or path";}
    if(code==102){return b"duplicate declaration in one scope";}
    if(code==103){return b"item is not visible from this module";}
    if(code==1007){return b"condition or logical operand must be bool";}
    if(code==1008){return b"assignment target is not mutable";}
    if(code==1010){return b"call argument count does not match the declaration";}
    if(code==1013){return b"Integer literal is outside the range of its selected `i32`, `u8`, `u32`, or `usize` type.";}
    if(code==1014){return b"Call argument label does not match its position or the call target does not support labels.";}
    if(code==4201){return b"@must_use must appear once without arguments on a callable with an eligible inert result";}
    if(code==4101){return b"Result value must be returned, propagated, matched, or explicitly discarded";}
    if(code==4102){return b"postfix ? requires a Result value";}
    if(code==4103){return b"propagated Result error type does not match the enclosing return type";}
    if(code==3101){return b"Region-derived value escapes its lexical region";}
    if(code==3102){return b"plain Region placement cannot contain a resource that needs drop";}
    if(code==3103){return b"compiler-owned Region capability cannot be stored";}
    if(code==3104){return b"compiler-owned Region capability cannot be constructed directly";}
    if(code==4202){return b"@must_use call result must be consumed or explicitly discarded";}
    if(code==4301){return b"A module constant or `@const_eval` declaration is malformed, misplaced, public, or otherwise ineligible.";}
    if(code==4302){return b"A well-typed expression is outside the admitted compile-time subset.";}
    if(code==4303){return b"Compile-time evaluation traps on arithmetic, division, shift, or cast semantics.";}
    if(code==4304){return b"A compile-time dependency cycle or deterministic evaluator budget prevents a result.";}
    if(code==4305){return b"A module-level `static_assert` evaluates to false.";}
    if(code==4401){return b"The reserved `source_location` intrinsic name is declared, bound, or used without its exact call form.";}
    if(code==4501){return b"Generic source preparation rejects a malformed, colliding, shadowing, or reserved declaration or binding.";}
    if(code==4502){return b"A generic call requires supported explicit scalar type arguments with the declared arity.";}
    if(code==4503){return b"A generic declaration or concrete request exceeds a catalog or expansion limit, or triggers a specialization cycle.";}
    if(code==5101){return b"Raw-pointer or unchecked operation appears outside an unsafe boundary.";}
    if(code==5102){return b"Raw foreign signature contains a safe reference.";}
    if(code==5106){return b"@link_name requires exactly one nonempty string and may appear only once";}
    if(code==7101){return b"`@effects` is empty, duplicated, malformed, or names an effect outside the Stage 0 vocabulary.";}
    if(code==7102){return b"A call or compiler-scheduled resource cleanup requires one or more effects not declared by the enclosing function.";}
    if(code==7201){return b"A package state declaration has malformed UID authority, invalid visibility, or more than one package owner.";}
    if(code==7202){return b"A package state field name is declared more than once.";}
    if(code==7203){return b"A package state field uses an unsupported or malformed storage codec declaration.";}
    return b"";
}
