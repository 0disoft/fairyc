"""Observe candidate-emitted Result pair programs; all type logic stays Fairy."""
import sys
from pathlib import Path
import verify_resource_lifetime as observer

UNIT = 151
observer.CASES[UNIT] = [
    ("enum-record", 'enum K{A;B;}struct E{u32 n;}fn Result<K,E> make(bool yes){return if(yes){.ok(.B)}else{.err(E{.n=9})};}fn i32 main(){assert(switch(make(true)){case .ok(k):{switch(k){case .A:{false}case .B:{true}}}case .err(_):{false}});assert(switch(make(false)){case .ok(_):{false}case .err(e):{e.n==9}});return 7;}', "", 7, 0),
    ("record-enum", 'enum K{A;B;}struct E{u32 n;}fn Result<E,K> make(bool yes){return if(yes){.ok(E{.n=9})}else{.err(.B)};}fn i32 main(){assert(switch(make(true)){case .ok(e):{e.n==9}case .err(_):{false}});assert(switch(make(false)){case .ok(_):{false}case .err(k):{switch(k){case .A:{false}case .B:{true}}}});return 7;}', "", 7, 0),
    ("option-record", 'struct V{u32 n;}struct E{u32 n;}fn Result<Option<V>,E> make(i32 n){return if(n==0){.ok(.none)}else if(n==1){.ok(.some(V{.n=9}))}else{.err(E{.n=4})};}fn i32 main(){assert(switch(make(0)){case .ok(v):{switch(v){case .none:{true}case .some(_):{false}}}case .err(_):{false}});assert(switch(make(1)){case .ok(v):{switch(v){case .none:{false}case .some(n):{n.n==9}}}case .err(_):{false}});assert(switch(make(2)){case .ok(_):{false}case .err(e):{e.n==4}});return 7;}', "", 7, 0),
]
observer.REJECTIONS[UNIT] = [
    ("different-record", 'enum K{A;B;}struct E{u32 n;}struct F{u32 n;}fn Result<K,E> make(Result<K,F> r){return r;}fn i32 main(){return 7;}'),
    ("nested-pair-bound", 'enum K{A;B;}struct E{u32 n;}fn Option<Result<K,E>> make(){return .none;}fn i32 main(){return 7;}'),
]
if __name__ == "__main__":
    sys.argv[1:1] = ["--unit", str(UNIT)]
    # Native build products belong in the cache even when the requested receipt
    # is a checked-in evidence document.
    at = sys.argv.index("--receipt") + 1
    destination = Path(sys.argv[at]).resolve()
    cache = observer.REPO / ".cache/candidate-result-pairs"
    cache.mkdir(parents=True, exist_ok=True)
    receipt = cache / "observation.json"
    sys.argv[at] = str(receipt)
    try:
        observer.main()
    finally:
        if receipt.exists():
            destination.parent.mkdir(parents=True, exist_ok=True)
            destination.write_bytes(receipt.read_bytes())
