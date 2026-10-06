#!/usr/bin/env python3
"""Observe resource consumption metadata and rejection without product emulation."""
import json,sys
from pathlib import Path
import verify_resource_types as observer
from verify_resource_move_cases import POSITIVES,NEGATIVES

def main():
    observer.POSITIVES=POSITIVES;observer.NEGATIVES=NEGATIVES
    receipt=Path(sys.argv[sys.argv.index('--receipt')+1]).resolve()
    try:observer.main()
    finally:
        if receipt.exists():
            result=json.loads(receipt.read_text(encoding='utf-8'))
            result.update(schema='fairyc.resource-move-hir/v1',unit=17,executionBoundary='Typed whole-root transfer metadata and partial-field rejection only; liveness checks follow in units 18..20, resource MIR/native stays closed until cleanup.')
            receipt.write_text(json.dumps(result,ensure_ascii=False,indent=2)+'\n',encoding='utf-8',newline='\n')
if __name__=='__main__':main()
