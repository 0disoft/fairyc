"""Observe dependency-closed source plans in the real Go source planner."""
import hashlib
import json
import os
from pathlib import Path
import subprocess

ROOT = Path(__file__).resolve().parents[1]
OUTPUT = ROOT / '.cache/generic-source-plan-20261013'
OUTPUT.mkdir(parents=True, exist_ok=True)
source = ROOT / 'tools/verify-candidate-generic-source-plan.go'
data = source.read_bytes()
prefix = b'//go:build ignore\n\n'
assert data.startswith(prefix)
oracle = OUTPUT / 'oracle_test.go'
oracle.write_bytes(data[len(prefix):])
overlay = OUTPUT / 'overlay.json'
overlay.write_text(json.dumps({'Replace': {str(ROOT / 'internal/genericsource/candidate_source_plan_oracle_test.go'): str(oracle)}}), encoding='utf-8')
env = dict(os.environ, FAIRYC_SOURCE_PLAN_REPO=str(ROOT), FAIRYC_SOURCE_PLAN_OUTPUT=str(OUTPUT))
command = ['bun', 'tools/run-go.ts', 'test', '-overlay', str(overlay), './internal/genericsource', '-run', '^TestCandidateSourcePlanOracle$', '-count=1', '-v']
result = subprocess.run(command, cwd=ROOT, env=env, capture_output=True, timeout=180)
(OUTPUT / 'go.stdout').write_bytes(result.stdout)
(OUTPUT / 'go.stderr').write_bytes(result.stderr)
if result.returncode:
    print(result.stdout.decode('utf-8', errors='replace'))
    print(result.stderr.decode('utf-8', errors='replace'))
    raise SystemExit(result.returncode)
report = json.loads((OUTPUT / 'oracle.json').read_text(encoding='utf-8'))
report.update(command=command, exit=result.returncode, toolSHA256=hashlib.sha256(data).hexdigest(), fixtureSHA256=hashlib.sha256((OUTPUT / 'oracle.bin').read_bytes()).hexdigest())
(OUTPUT / 'oracle.json').write_text(json.dumps(report, indent=2)+'\n', encoding='utf-8', newline='\n')
print(json.dumps(report, indent=2))
