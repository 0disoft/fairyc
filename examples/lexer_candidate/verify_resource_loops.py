#!/usr/bin/env python3
"""Observe source loop equality and retain exact loop-header/move diagnostics."""
import json,sys
from pathlib import Path
import verify_resource_types as observer
from verify_resource_loop_cases import POSITIVES,NEGATIVES

def main():
    observer.POSITIVES=POSITIVES;observer.NEGATIVES=NEGATIVES
    receipt=Path(sys.argv[sys.argv.index('--receipt')+1]).resolve();verified=False
    try:
        observer.main();result=json.loads(receipt.read_text(encoding='utf-8'))
        rejected={str(receipt.parent/(name+'.fy')):code for name,_,code in NEGATIVES}
        for row in result['cli']:
            code=rejected.get(row['argv'][-1])
            if code:
                assert '  move: ' in row['stderr'],row
                if code=='E1105':assert '  loop-header: ' in row['stderr'],row
        verified=True
    finally:
        if receipt.exists():
            result=json.loads(receipt.read_text(encoding='utf-8'));result['passed']=result['passed'] and verified
            result.update(schema='fairyc.resource-loop-flow/v1',unit=20,executionBoundary='Each reachable normal/break/continue edge preserves post-condition root state after exited scopes are killed. Returns/propagation error paths are terminal; resource MIR/native stays closed until cleanup.')
            receipt.write_text(json.dumps(result,ensure_ascii=False,indent=2)+'\n',encoding='utf-8',newline='\n')
if __name__=='__main__':main()
