#!/usr/bin/env python3
"""Reuse the byte-preserving CLI observer with unit-16 construction fixtures."""
import json,sys
from pathlib import Path
import verify_resource_types as observer
from verify_resource_factory_cases import POSITIVES,NEGATIVES

def main():
    observer.POSITIVES=POSITIVES;observer.NEGATIVES=NEGATIVES
    receipt=Path(sys.argv[sys.argv.index('--receipt')+1]).resolve()
    try:
        observer.main()
    finally:
        if receipt.exists():
            result=json.loads(receipt.read_text(encoding='utf-8'))
            result.update(schema='fairyc.resource-factory-hir/v1',unit=16,
                executionBoundary='Same-module explicit record construction and ordinary factory HIR only; move flow and cleanup follow. All resource MIR/C stays closed. Cross-module ownership visibility follows in module integration.')
            receipt.write_text(json.dumps(result,ensure_ascii=False,indent=2)+'\n',encoding='utf-8',newline='\n')
if __name__=='__main__':main()
