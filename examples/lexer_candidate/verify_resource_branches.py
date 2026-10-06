#!/usr/bin/env python3
"""Observe explicit source join diagnostics, retaining move provenance."""
import json,sys
from pathlib import Path
import verify_resource_types as observer
from verify_resource_branch_cases import POSITIVES,NEGATIVES

def main():
    observer.POSITIVES=POSITIVES;observer.NEGATIVES=NEGATIVES
    receipt=Path(sys.argv[sys.argv.index('--receipt')+1]).resolve();verified=False
    try:
        observer.main();result=json.loads(receipt.read_text(encoding='utf-8'))
        rejected={str(receipt.parent/(name+'.fy')) for name,_,_ in NEGATIVES}
        for row in result['cli']:
            if row['argv'][-1] in rejected:assert '  move: ' in row['stderr'],row
        verified=True
    finally:
        if receipt.exists():
            result=json.loads(receipt.read_text(encoding='utf-8'));result['passed']=result['passed'] and verified
            result.update(schema='fairyc.resource-branch-flow/v1',unit=19,executionBoundary='Reachable Live/Dead joins must agree even without a later use; terminal branches are excluded. Loop-header/exit equality is unit 20; resource MIR/native remains closed until cleanup.')
            receipt.write_text(json.dumps(result,ensure_ascii=False,indent=2)+'\n',encoding='utf-8',newline='\n')
if __name__=='__main__':main()
