"""Extract Windows CLI shard weights from completed GitHub Actions job logs."""
import argparse
import json
import re
from pathlib import Path

HEADER = re.compile(r"Windows Go CLI shard (\d+)/(\d+) runs (\d+) of (\d+) entries")
RESULT = re.compile(r"--- (PASS|SKIP|FAIL): ((?:Test|Example|Fuzz)\w*) \((\d+)\.(\d{2})s\)")
LOG_LINE = re.compile(r"[^\t]+\t[^\t]+\t\S+ (.*)")


def extract(logs):
    timings = {}
    seen_shards = set()
    expected = None
    for log in logs:
        lines = [
            match[1]
            for line in log.read_text(encoding="utf-8-sig").splitlines()
            if (match := LOG_LINE.fullmatch(line))
        ]
        headers = [match for line in lines if (match := HEADER.match(line))]
        if len(headers) != 1:
            raise ValueError(f"{log}: expected one shard header")
        shard, shard_count, count, total = map(int, headers[0].groups())
        if expected is None:
            expected = shard_count, total
        if (shard_count, total) != expected or not 0 <= shard < shard_count or shard in seen_shards:
            raise ValueError(f"{log}: inconsistent or duplicate shard {shard}")
        seen_shards.add(shard)
        results = [match for line in lines if (match := RESULT.fullmatch(line))]
        if len(results) != count:
            raise ValueError(f"{log}: found {len(results)} results for {count} entries")
        for status, name, seconds, fraction in (match.groups() for match in results):
            if status == "FAIL" or name in timings:
                raise ValueError(f"{log}: failing or duplicate test {name}")
            timings[name] = 100 * int(seconds) + int(fraction)

    if expected != (len(seen_shards), len(timings)):
        raise ValueError("logs do not cover the complete shard matrix")
    return dict(sorted(timings.items()))


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("run_id", type=int)
    parser.add_argument("log_directory", type=Path)
    args = parser.parse_args()
    try:
        timings = extract(sorted(args.log_directory.glob("*.log")))
    except ValueError as error:
        parser.error(str(error))
    print(json.dumps({"runId": args.run_id, "elapsedCentiseconds": timings}, indent=2))


if __name__ == "__main__":
    main()
