#!/usr/bin/env python3
"""Use the shared CLI observer and check ownership diagnostic provenance."""
import json,sys
from pathlib import Path
import verify_resource_types as observer
from verify_resource_dead_cases import POSITIVES,NEGATIVES

def main():
    observer.POSITIVES=POSITIVES;observer.NEGATIVES=NEGATIVES
    receipt=Path(sys.argv[sys.argv.index('--receipt')+1]).resolve()
    verified=False
    try:
        observer.main()
        result=json.loads(receipt.read_text(encoding='utf-8'))
        rejected={str(receipt.parent/(name+'.fy')) for name,_,_ in NEGATIVES}
        for row in result['cli']:
            if row['argv'][-1] in rejected:assert '  move: ' in row['stderr'],row
        verified=True
    finally:
        if receipt.exists():
            result=json.loads(receipt.read_text(encoding='utf-8'))
            result['passed']=result['passed'] and verified
            result.update(schema='fairyc.resource-dead-flow/v1',unit=18,executionBoundary='Root use-after-move rejection with exact transfer provenance; branch join and loop-edge equality gates are units 19/20. Resource MIR/native remains closed until cleanup.')
            receipt.write_text(json.dumps(result,ensure_ascii=False,indent=2)+'\n',encoding='utf-8',newline='\n')
if __name__=='__main__':main()
