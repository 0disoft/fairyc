#!/usr/bin/env python3
"""Observe emitted nominal MIR; the Go interpreter owns expected semantics."""
import argparse, hashlib, json, subprocess
from pathlib import Path
from verify_mir import read_mir, validate, execute, TYPES
from verify_hir import source_tokens

ROOT=Path(__file__).resolve().parent
def sha(path): return hashlib.sha256(Path(path).read_bytes()).hexdigest()

def compare(fixture,row):
    assert (row['exit'],row['stderr'])==(0,''),row
    source=bytes.fromhex(fixture['inputHex']);t=read_mir(row['stdout'],source);names=validate(t,source)
    structs=fixture['program']['StructTypes'];assert len(structs)==len(t['N'])
    types={**TYPES,**{id:s['Type'] for id,s in zip(t['N'],structs,strict=True)}}
    for n,s in zip(t['N'].values(),structs,strict=True):
        assert n['field_count']==len(s['Fields'] or [])
        for i,f in enumerate(s['Fields'] or []):
            actual=t['D'][n['field_start']+i+1]
            span=f['Span'];start,end=span['StartByte'],span['EndByte'];name,a,b=source_tokens(source[start:end])[-2]
            assert (name.decode('ascii'),start+a,start+b)==(f['Name'],actual['start'],actual['end'])
            assert types[actual['type']]==f['Type']
    expected={f['Name']:f for f in fixture['program']['Functions']};assert set(names.values())==set(expected)
    for f in t['F'].values():
        target=t['T'][f['target']];go=expected[names[f['target']]]
        assert (f['start'],f['end'])==(go['Span']['StartByte'],go['Span']['EndByte'])
        assert types[target['result']]==go['ResultType']
        assert [types[t['S'][j]['type']] for j in range(target['signature_start'],target['signature_start']+target['signature_count'])]==(go['ParameterTypes'] or [])
    actual=execute(t,names,fixture.get('traceCalls',False))
    assert actual=={k:fixture['outcome'][k] for k in actual},dict(case=fixture['name'],actual=actual,expected=fixture['outcome'])
    assert fixture['outcomeOrigin']=='go-mir-interpreter'
    row['mirObservation']=actual;row['outcomeOrigin']='go-mir-interpreter'

def main():
    parser=argparse.ArgumentParser();parser.add_argument('--images',nargs=2,required=True);parser.add_argument('--reference',required=True);parser.add_argument('--receipt',required=True)
    args=parser.parse_args();images=[Path(p).resolve() for p in args.images];reference=Path(args.reference).resolve();receipt=Path(args.receipt).resolve();receipt.parent.mkdir(parents=True,exist_ok=True)
    data=json.loads(reference.read_text(encoding='utf-8'));assert data['schema']=='fairyc.nominal-mir-oracle/v1'
    rows=[]
    for fixture in data['fixtures']:
        path=receipt.parent/(fixture['name']+'.fy');path.write_bytes(bytes.fromhex(fixture['inputHex']));digest=sha(path)
        for image in images:
            argv=[str(image),'--mir-file',str(path)];p=subprocess.run(argv,cwd=ROOT,capture_output=True,timeout=30)
            row=dict(case=fixture['name'],argv=argv,cwd=str(ROOT),exit=p.returncode,stdout=p.stdout.decode('utf-8'),stderr=p.stderr.decode('utf-8'),inputSHA256=digest,unchanged=sha(path)==digest)
            assert row['unchanged'];compare(fixture,row);rows.append(row)
    result=dict(schema='fairyc.nominal-mir-cli-verification/v1',passed=True,semanticComparisons=len(rows),images=[dict(path=str(p),sha256=sha(p)) for p in images],reference=dict(path=str(reference),sha256=sha(reference)),sources=[dict(path=str(ROOT/p),sha256=sha(ROOT/p)) for p in ('verify_nominal_mir.py','verify_mir.py','verify_hir.py')],runs=rows)
    receipt.write_text(json.dumps(result,ensure_ascii=False,indent=2)+'\n',encoding='utf-8',newline='\n');print(f'PASS: {len(rows)} nominal MIR interpreter comparisons')
if __name__=='__main__':main()
