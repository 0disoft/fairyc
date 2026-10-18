"""Compare source-independent candidate commands and every Go diagnostic explanation."""
import argparse
import importlib.util
import json
from pathlib import Path
import subprocess

ROOT = Path(__file__).resolve().parents[1]
spec = importlib.util.spec_from_file_location('behavior', ROOT / 'tools/compare-candidate-go-behavior.py')
behavior = importlib.util.module_from_spec(spec)
spec.loader.exec_module(behavior)


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--build-receipt', type=Path, required=True)
    parser.add_argument('--reference', type=Path, required=True)
    parser.add_argument('--output', type=Path, required=True)
    args = parser.parse_args()
    assert __debug__, 'verification requires Python without -O'
    output = args.output.resolve()
    assert output.is_relative_to(ROOT / '.cache') and not output.exists()
    build_path, reference = args.build_receipt.resolve(), args.reference.resolve()
    build_sha, reference_sha = behavior.sha(build_path), behavior.sha(reference)
    build = json.loads(build_path.read_bytes())
    assert build['passed'] and not build['goFrontendInvoked']
    assert not build.get('compilerOnly') and not build.get('commandsOnly')
    assert build['buildVersion']['oracle']['sha256'] == reference_sha
    sources = {row['path']: row['sha256'] for row in build['sources']}
    sources.update({p: behavior.sha(ROOT / p) for p in [
        'tools/verify-candidate-cli-metadata.py', 'tools/compare-candidate-go-behavior.py',
        'internal/diagnostic/catalog.go', 'internal/cli/run.go', 'go.mod', 'go.sum',
    ]})
    assert all(behavior.sha(ROOT / path) == sha for path, sha in sources.items())
    output.mkdir(parents=True)
    # Inventory comes from the actual Go API, never the generated candidate table:
    # an omitted diagnostic in that table must fail the comparison.
    inventory = output / 'catalog.go'
    inventory.write_text('''package main
import("encoding/json";"os";"runtime";
"github.com/0disoft/fairyc/internal/diagnostic";
"github.com/0disoft/fairyc/internal/cdriver")
func main(){
  codes:=[]string{};for _,entry:=range diagnostic.CatalogEntries(){codes=append(codes,entry.Code)}
  report:=map[string]any{"codes":codes,"go":runtime.Version(),"os":runtime.GOOS,"arch":runtime.GOARCH}
  if runtime.GOOS=="windows"{tool,err:=cdriver.ResolveWindowsHostMSVC();if err!=nil{panic(err)};report["msvc"]=tool.Version}
  if err:=json.NewEncoder(os.Stdout).Encode(report);err!=nil{panic(err)}
}
''', encoding='utf-8', newline='\n')
    result = subprocess.run(['go', 'run', '-mod=readonly', str(inventory)], cwd=ROOT,
                            capture_output=True, check=True, timeout=120)
    assert not result.stderr, result.stderr
    catalog = json.loads(result.stdout)
    codes = catalog['codes']
    assert codes and len(codes) == len(set(codes))
    behavior.save(output / 'catalog.json', catalog)
    # A deliberately invalid manifest proves these commands do not read a workspace.
    outside = output / 'unrelated-workspace'
    outside.mkdir()
    (outside / 'fairy.toml').write_text('deliberately invalid manifest\n', encoding='utf-8')
    invocations = [[], ['--help'], ['--version'], ['targets'], ['targets', '--json']]
    invocations += [['explain', code] for code in codes]
    invocations += [['explain'], ['explain', 'V9999'], ['explain', 'V1003', 'extra']]
    rows = []
    for image in build['images']:
        candidate = Path(image['path'])
        assert behavior.sha(candidate) == image['sha256']
        for argv in invocations:
            go = behavior.execute(reference, argv, outside, 30)
            other = behavior.execute(candidate, argv, outside, 30)
            comparison = behavior.comparison(go, other)
            if argv == ['--version']:
                assert go['stdout'] == build['buildVersion']['text']
            row = dict(profile=image['profile'], argv=argv, go=go, candidate=other,
                       comparison=comparison)
            rows.append(row)
            behavior.save(output / 'observations.json', rows)
            if not comparison['exactMatch']:
                print('difference:', image['profile'], argv, flush=True)
        print('compared:', image['profile'], len(invocations), 'commands', flush=True)
    assert rows
    assert behavior.sha(reference) == reference_sha and behavior.sha(build_path) == build_sha
    assert all(behavior.sha(ROOT / path) == sha for path, sha in sources.items())
    assert all(behavior.sha(Path(row['path'])) == row['sha256'] for row in build['images'])
    report = dict(schema='fairyc.candidate-cli-metadata-comparison/v1',
                  passed=all(row['comparison']['exactMatch'] for row in rows),
                  buildReceipt=str(build_path), buildReceiptSHA256=build_sha,
                  reference=dict(path=str(reference), sha256=reference_sha),
                  images=build['images'], diagnosticCodes=codes, toolchain=catalog,
                  comparisons=len(rows), exactComparisons=sum(row['comparison']['exactMatch'] for row in rows),
                  sourceInputs=[dict(path=path, sha256=sha) for path, sha in sources.items()], rows=rows,
                  limitations=['Source-independent CLI commands only; workspace compilation and test execution are separate gates',
                               'Windows candidate binaries; no compiler structure equivalence or self-hosting fixed-point claim'])
    behavior.save(output / 'receipt.json', report)
    print('passed:', report['passed'], report['exactComparisons'], '/', len(rows), flush=True)
    if not report['passed']:
        raise SystemExit(1)


if __name__ == '__main__':
    main()
