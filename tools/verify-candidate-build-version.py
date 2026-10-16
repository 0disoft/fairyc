"""Compare a narrow real candidate version image with a fresh source Go build."""
import argparse
import json
from pathlib import Path
import re
import subprocess

from candidate_build_version import ROOT, generate, sha


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--candidate', type=Path, required=True)
    parser.add_argument('--builder', type=Path, required=True)
    parser.add_argument('--output', type=Path, required=True)
    args = parser.parse_args()
    output = args.output.resolve()
    output.mkdir(parents=True, exist_ok=False)
    version_source, metadata = generate(output)
    entry = output / 'main.fy'
    entry.write_text('module main;\nimport public_cli_data as data;\nimport std::io as io;\n'
                     '@effects(console) fn i32 main(){return if(io::write(data::version()).is_ok){0}else{4};}\n',
                     encoding='utf-8', newline='\n')
    modules = {'main': entry, 'public_cli_data': version_source}
    for path in (ROOT / 'internal/sdkstdlib/sources').rglob('*.fy'):
        name = re.search(r'^module\s+([\w:]+);', path.read_text(encoding='utf-8'))[1]
        modules[name] = path
    seen, pending = set(), ['main']
    while pending:
        name = pending.pop()
        if name not in seen:
            seen.add(name)
            pending.extend(re.findall(r'^import\s+([\w:]+)', modules[name].read_text(encoding='utf-8'), re.M))
    sources = [entry] + [modules[name] for name in sorted(seen - {'main'})]
    manifest = output / 'sources.list'
    manifest.write_text(''.join(p.relative_to(ROOT).as_posix() + '\n' for p in sources), encoding='utf-8', newline='\n')
    packet = output / 'version.bin'
    emitted = subprocess.run([str(args.candidate.resolve()), '--c-project-file', str(manifest)],
                             cwd=ROOT, capture_output=True, check=True, timeout=120)
    if emitted.stderr or not emitted.stdout.startswith(b'FYCPARTS1\n'):
        raise ValueError('candidate did not emit a clean C packet')
    packet.write_bytes(emitted.stdout)
    outside = output / 'unrelated-workspace'
    outside.mkdir()
    (outside / 'fairy.toml').write_text('deliberately invalid workspace')
    images = []
    for profile, flags in [('development', []), ('release', ['--release'])]:
        image = output / (profile + '.exe')
        subprocess.run([str(args.builder.resolve()), '--packet', str(packet), '--packet-sha256', sha(packet),
                        '--output', str(image), *flags], cwd=ROOT, check=True, capture_output=True, timeout=180)
        for cwd in (ROOT, outside):
            actual = subprocess.run([str(image)], cwd=cwd, capture_output=True, check=True, timeout=30)
            if actual.stderr or actual.stdout != metadata['text'].encode('utf-8'):
                raise ValueError('candidate source identity differs from Go')
        images.append(dict(profile=profile, path=str(image), sha256=sha(image)))
    receipt = dict(passed=True, scope='generated version module through real candidate C emission and native linking',
                   buildVersion=metadata, images=images, sources=[dict(path=p.relative_to(ROOT).as_posix(), sha256=sha(p)) for p in sources],
                   tools={name: dict(path=str(p.resolve()), sha256=sha(p)) for name, p in [('candidate', args.candidate), ('builder', args.builder)]})
    (output / 'receipt.json').write_text(json.dumps(receipt, indent=2) + '\n', encoding='utf-8')
    print(metadata['text'], end='')
    print('verified: both profiles, repository and unrelated workspace')


if __name__ == '__main__':
    main()
