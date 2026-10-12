from pathlib import Path
import argparse,json,subprocess,re,hashlib,time
root=Path(__file__).resolve().parents[1]
parser=argparse.ArgumentParser(description='Probe workspace status and first diagnostic over semantic/resource fixtures; differences are recorded, not treated as successful parity.')
parser.add_argument('--candidate',type=Path,required=True)
parser.add_argument('--reference',type=Path,required=True)
parser.add_argument('--output',type=Path,required=True)
args=parser.parse_args()
if not __debug__:parser.error('verification requires Python without -O')
candidate=args.candidate.resolve();oracle=args.reference.resolve()
work=args.output.resolve();work.mkdir(parents=True,exist_ok=True)
rows=[]
save=lambda p,v:p.write_text(json.dumps(v,indent=2)+'\n',encoding='utf-8',newline='\n')
sha=lambda p:hashlib.sha256(p.read_bytes()).hexdigest()
tool_hashes=(sha(candidate),sha(oracle))
for family in ['semantic','resource']:
    manifest=root/f'spec/fixtures/{family}/manifest-v0.json';data=json.loads(manifest.read_text())
    for index,case in enumerate(data['cases']):
        workspace=work/family/case['id'];(workspace/'src').mkdir(parents=True,exist_ok=True)
        (workspace/'fairy.toml').write_text(f'schema = "fairyc.manifest/v1"\n[package]\nuid = "fixture.{family}.case{index}"\nname = "frontend-fixture"\nversion = "0.1.0"\nlanguage = "fairyc-language/20"\nsource = "src"\n',encoding='utf-8',newline='\n')
        inputs=[]
        for filename in case['sources']:
            original=manifest.parent/case['sourceRoot']/filename;raw=original.read_bytes()
            declared=re.search(rb'^\s*module\s+([A-Za-z0-9_:]+)\s*;',raw,re.M)
            relative=(declared[1].decode().replace('::','/')+'.fy') if declared else filename
            target=workspace/'src'/relative;target.parent.mkdir(parents=True,exist_ok=True);target.write_bytes(raw)
            inputs.append(dict(original=original.relative_to(root).as_posix(),sourceSHA256=sha(original),projectSource=relative))
        observations=[]
        for label,exe in [('go',oracle),('candidate',candidate)]:
            started=time.monotonic();p=subprocess.run([exe,'check'],cwd=workspace,capture_output=True,timeout=60)
            err=p.stderr.decode(errors='replace');matches=re.findall(r'(E\d{4}).*?\[bytes (\d+)\.\.(\d+)',err)
            observations.append(dict(engine=label,exit=p.returncode,seconds=time.monotonic()-started,stdout=p.stdout.decode(errors='replace'),stderr=err,diagnostics=[dict(code=c,start=int(a),end=int(b)) for c,a,b in matches]))
        expected=case['expect']['status'];go,c=observations
        expected_codes=[d['code'] for d in case['expect'].get('diagnostics',[])]
        status_match=(go['exit']==0)==(c['exit']==0)
        first_match=(go['diagnostics'][:1]==c['diagnostics'][:1])
        row=dict(family=family,id=case['id'],manifestSHA256=sha(manifest),expected=expected,sources=inputs,observations=observations,statusMatch=status_match,firstDiagnosticMatch=first_match,goMatchesFixture=((go['exit']==0)==(expected=='pass') and (not expected_codes or [d['code'] for d in go['diagnostics'][:1]]==expected_codes[:1])))
        rows.append(row);save(work/'observations.json',rows)
        if not status_match:print('STATUS DIFFERENCE',family,case['id'],go['exit'],c['exit'],flush=True)
assert (sha(candidate),sha(oracle))==tool_hashes,'tools changed during the probe'
assert all(r['goMatchesFixture'] for r in rows),'reference does not match owned fixtures'
save(work/'receipt.json',dict(schema='fairyc.candidate-frontend-corpus-probe/v1',scriptSHA256=sha(Path(__file__)),allStatusMatch=all(r['statusMatch'] for r in rows),allFirstDiagnosticMatch=all(r['firstDiagnosticMatch'] for r in rows),cases=len(rows),matched=sum(r['statusMatch'] and r['firstDiagnosticMatch'] and r['goMatchesFixture'] for r in rows),candidateSHA256=sha(candidate),oracleSHA256=sha(oracle),rows=rows))
print('frontend corpus completed',len(rows),'cases;',sum(not r['statusMatch'] for r in rows),'status differences;',sum(not r['firstDiagnosticMatch'] for r in rows),'first diagnostic differences',flush=True)
