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
    if(code==101){return b"name or qualified path cannot be resolved";}
    if(code==102){return b"declaration duplicates a name in the same scope";}
    if(code==103){return b"item is not visible from this module";}
    if(code==1007){return b"condition or logical operand is not bool";}
    if(code==1008){return b"assignment target is not a mutable place";}
    if(code==1010){return b"call argument count does not match the declaration";}
    if(code==1013){return b"integer literal is outside the selected type's range";}
    if(code==1014){return b"call argument label does not match its position or target";}
    if(code==4201){return b"invalid must_use declaration";}
    if(code==4101){return b"Result value must be returned, propagated, matched, or explicitly discarded";}
    if(code==4202){return b"must_use call result is ignored without discard";}
    if(code==4301){return b"invalid compile-time declaration";}
    if(code==4302){return b"expression is outside the supported compile-time subset";}
    if(code==4303){return b"compile-time evaluation traps";}
    if(code==4304){return b"compile-time dependency cycle or evaluator budget exhausted";}
    if(code==4305){return b"static_assert evaluates to false";}
    if(code==4401){return b"source_location requires its reserved exact call form";}
    if(code==4501){return b"Generic source preparation rejects a malformed, colliding, shadowing, or reserved declaration or binding.";}
    if(code==4502){return b"A generic call requires supported explicit scalar type arguments with the declared arity.";}
    if(code==4503){return b"A generic declaration or concrete request exceeds a catalog or expansion limit, or triggers a specialization cycle.";}
    if(code==5101){return b"unchecked operation is outside an unsafe boundary";}
    if(code==5102){return b"raw foreign signature contains a safe reference";}
    if(code==5106){return b"invalid or conflicting link_name declaration";}
    if(code==7101){return b"invalid effects declaration";}
    if(code==7102){return b"call or resource cleanup exceeds the declared effects";}
    if(code==7201){return b"invalid package state declaration or contract UID authority";}
    if(code==7202){return b"package state field name is declared more than once";}
    if(code==7203){return b"unsupported or malformed package state storage codec";}
    return b"";
}
