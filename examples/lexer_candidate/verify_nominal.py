#!/usr/bin/env python3
"""Independent Go type outcomes versus Fairy checking; no product semantics."""
import argparse, hashlib, json, re, subprocess
from pathlib import Path

ROOT=Path(__file__).resolve().parent
def sha(path):return hashlib.sha256(Path(path).read_bytes()).hexdigest()
def main():
    parser=argparse.ArgumentParser();parser.add_argument('--images',nargs=2,required=True);parser.add_argument('--reference',required=True);parser.add_argument('--receipt',required=True)
    args=parser.parse_args();images=[Path(p).resolve() for p in args.images];reference=Path(args.reference).resolve();receipt=Path(args.receipt).resolve();receipt.parent.mkdir(parents=True,exist_ok=True)
    data=json.loads(reference.read_text(encoding='utf-8'));assert data['schema']=='fairyc.nominal-types-oracle/v1'
    rows=[];counts=dict(semanticComparisons=0,candidateLayoutRejections=0,candidateValueRejections=0,backendRejections=0,hirAccepted=0)
    for f in data['fixtures']:
        path=receipt.parent/(f['name']+'.fy');source=bytes.fromhex(f['inputHex']);path.write_bytes(source);before=sha(path)
        for image in images:
            argv=[str(image),'--check-types-file',str(path)];p=subprocess.run(argv,cwd=ROOT,capture_output=True,timeout=30)
            row=dict(case=f['name'],argv=argv,cwd=str(ROOT),exit=p.returncode,stdout=p.stdout.decode('utf-8'),stderr=p.stderr.decode('utf-8'),inputSHA256=before,unchanged=sha(path)==before);rows.append(row);assert row['unchanged']
            if f['outcome']=='pass':assert (p.returncode,row['stdout'],row['stderr'])==(0,'types ok\n',''),row
            else:
                assert p.returncode==2 and row['stdout']=='',row
                match=re.fullmatch(re.escape(str(path))+r':(\d+):(\d+): error: .+ \(bytes (\d+)\.\.(\d+)\)\n',row['stderr']);assert match,row
                line,column,start,end=map(int,match.groups());assert 0<=start<end<=len(source)
                prefix=source[:start];assert line==prefix.count(b'\n')+1 and column==start-(prefix.rfind(b'\n')+1)+1,row
            if f['outcome']=='candidate-layout-rejection':
                assert not f['diagnostics'] and not f['unsupported'];assert 'recursive value type' in row['stderr'];counts['candidateLayoutRejections']+=1
            elif f['outcome']=='candidate-value-rejection':
                assert not f['diagnostics'] and not f['unsupported'];assert 'invalid nominal type' in row['stderr'];counts['candidateValueRejections']+=1
            else:
                assert f['outcome'] in ('pass','error');assert f['outcome']=='pass' or f['diagnostics'] or f['unsupported'];counts['semanticComparisons']+=1
    path=receipt.parent/'backend-boundary.fy';source=b'module main;struct Pair{i32 x;}fn i32 main(){return Pair{.x=1}.x;}';path.write_bytes(source)
    for image in images:
        for mode in ('--hir-file','--mir-file','--c-file'):
            argv=[str(image),mode,str(path)];p=subprocess.run(argv,cwd=ROOT,capture_output=True,timeout=30)
            row=dict(case='backend-boundary',argv=argv,cwd=str(ROOT),exit=p.returncode,stdout=p.stdout.decode('utf-8'),stderr=p.stderr.decode('utf-8'),inputSHA256=sha(path),unchanged=path.read_bytes()==source);rows.append(row)
            if mode=='--hir-file':
                assert p.returncode==0 and row['stdout'].startswith('fairyc.nominal-hir/v1\n') and row['stderr']=='' and row['unchanged'],row;counts['hirAccepted']+=1
            else:
                message='invalid or unsupported MIR input'
                assert p.returncode==2 and row['stdout']=='' and message in row['stderr'] and row['unchanged'],row;counts['backendRejections']+=1
    result=dict(schema='fairyc.nominal-type-cli-verification/v1',passed=True,counts=counts,images=[dict(path=str(p),sha256=sha(p)) for p in images],reference=dict(path=str(reference),sha256=sha(reference)),scriptSHA256=sha(__file__),runs=rows)
    receipt.write_text(json.dumps(result,ensure_ascii=False,indent=2)+'\n',encoding='utf-8',newline='\n');print(f'PASS: {len(rows)} nominal CLI executions; {counts}')
if __name__=='__main__':main()
