"""Prepare an isolated Linux candidate workspace; never overwrite authored sources."""
import argparse
from pathlib import Path
import shutil

ROOT = Path(__file__).resolve().parents[1]
SOURCE = ROOT / 'examples/lexer_candidate'


def prepare(output):
    output = output.resolve()
    if not output.is_relative_to((ROOT / '.cache').resolve()):
        raise ValueError('output must be inside the repository .cache directory')
    # Refuse reuse rather than mixing different source snapshots.
    output.mkdir(parents=True, exist_ok=False)
    shutil.copyfile(SOURCE / 'fairy.toml', output / 'fairy.toml')
    shutil.copytree(SOURCE / 'src', output / 'src')
    main = output / 'src/main.fy'
    source = main.read_text(encoding='utf-8')
    old = 'import file_cli as file_cli;'
    if source.count(old) != 1:
        raise ValueError('candidate main dispatcher import changed')
    main.write_text(source.replace(old, 'import linux_file_cli as file_cli;'),
                    encoding='utf-8', newline='\n')
    return output


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--output', required=True, type=Path)
    args = parser.parse_args()
    print(prepare(args.output))


if __name__ == '__main__':
    main()
