import fs from 'node:fs';
import {spawnSync} from 'node:child_process';

const mode = process.argv[2];
const output = 'docs/ops/evidence/c-scanner-storage-ab-20260925.json';
if (mode === 'verify') {
  const report = JSON.parse(fs.readFileSync(output, 'utf8'));
  const markdown = fs.readFileSync('docs/ops/c-scanner-storage-ab-20260925.md', 'utf8');
  if (report.schema !== 'fairyc.c-scanner-storage-ab/v1' || report.mode !== 'measure' ||
      report.warmup_pairs !== 1 || report.measured_pairs !== 12 ||
      report.images?.length !== 2 || report.samples?.length !== 13 ||
      report.images[0].lane !== 'preallocated' || report.images[1].lane !== 'geometric') {
    throw new Error('Incomplete measured storage A/B evidence');
  }
  const preallocated = [], geometric = [], paired = [];
  let preallocatedFirst = 0, geometricFirst = 0;
  for (const [index, sample] of report.samples.entries()) {
    if (sample.pair !== index || sample.warmup !== (index === 0) ||
        sample.lanes?.length !== 2 || sample.order?.length !== 2 ||
        sample.order[0] === sample.order[1]) {
      throw new Error(`Invalid storage pair ${index}`);
    }
    const lanes = Object.fromEntries(sample.lanes.map(lane => [lane.lane, lane]));
    if (!lanes.preallocated || !lanes.geometric) throw new Error(`Missing lane ${index}`);
    for (const lane of sample.lanes) {
      if (lane.completed !== report.rounds || lane.total !== report.per_round_result * report.rounds ||
          lane.qpc_ticks <= 0 || lane.qpc_frequency <= 0) {
        throw new Error(`Invalid lane accounting ${index}`);
      }
    }
    if (index === 0) continue;
    if (sample.order[0] === 'preallocated') preallocatedFirst++;
    else geometricFirst++;
    const p = 1000 * lanes.preallocated.qpc_ticks / lanes.preallocated.qpc_frequency;
    const g = 1000 * lanes.geometric.qpc_ticks / lanes.geometric.qpc_frequency;
    preallocated.push(p);
    geometric.push(g);
    paired.push(g / p);
  }
  const median = values => {
    const ordered = [...values].sort((a, b) => a - b);
    return (ordered[5] + ordered[6]) / 2;
  };
  const row = (values, unit) => {
    const middle = median(values);
    const low = Math.min(...values), high = Math.max(...values);
    const digits = unit === 'ms' ? 3 : 5;
    const format = value => `${(value + 1e-9).toFixed(digits)}${unit ? ` ${unit}` : ''}`;
    return `${format(middle)} | ${format(low)} | ${format(high)} | ` +
      `${(100 * (high - low) / middle).toFixed(2)}%`;
  };
  for (const expected of [
    `| Preallocated | ${row(preallocated, 'ms')} |`,
    `| Geometric | ${row(geometric, 'ms')} |`,
    `| Paired geometric/preallocated | ${row(paired, '')} |`,
  ]) {
    if (!markdown.includes(expected)) throw new Error(`Report table differs: ${expected}`);
  }
  if (preallocatedFirst !== 6 || geometricFirst !== 6 ||
      report.images[0].exe_bytes !== 181760 || report.images[1].exe_bytes !== 181248 ||
      !markdown.includes('timing inconclusive') ||
      !markdown.includes('512 bytes smaller')) {
    throw new Error('Storage A/B interpretation differs from evidence');
  }
  process.stdout.write('Storage A/B raw pairs, capacity comparison and report table verified.\n');
  process.exit(0);
}
if (mode !== 'smoke' && mode !== 'measure') {
  throw new Error('Expected smoke, measure or verify mode');
}
if (mode === 'measure' && fs.existsSync(output)) {
  throw new Error('Storage A/B evidence already exists');
}
const env = {...process.env, GOMAXPROCS: '1', FAIRYC_C_SCANNER_STORAGE_AB: mode};
const run = spawnSync('bun', [
  'tools/run-go.ts', 'test', '-mod=readonly', '-count=1', '-v',
  '-run', '^TestPrivateVectorSDKCandidateIndependentCScannerStorageAB$',
  './internal/compiler',
], {encoding: 'utf8', shell: false, env, timeout: 240000, maxBuffer: 8 << 20});
if (run.status !== 0) {
  throw new Error(`${run.stderr || ''}\n${(run.stdout || '').slice(-9000)}`);
}
const match = run.stdout.match(/C_SCANNER_STORAGE_AB (\{[^\r\n]+\})/u);
if (!match) throw new Error('Missing storage A/B report');
const report = JSON.parse(match[1]);
const expectedPairs = mode === 'measure' ? 13 : 1;
if (report.schema !== 'fairyc.c-scanner-storage-ab/v1' || report.mode !== mode ||
    report.images?.length !== 2 || report.images[0]?.lane !== 'preallocated' ||
    report.images[1]?.lane !== 'geometric' || report.samples?.length !== expectedPairs ||
    report.rounds !== (mode === 'measure' ? 250000 : 1000) ||
    report.gomaxprocs !== 1) {
  throw new Error('Incomplete storage A/B report');
}
for (const sample of report.samples) {
  if (sample.order?.length !== 2 || sample.lanes?.length !== 2 ||
      sample.lanes.some(lane => lane.qpc_ticks <= 0 || lane.qpc_frequency <= 0 ||
        lane.completed !== report.rounds)) {
    throw new Error('Invalid storage A/B sample');
  }
}
if (mode === 'measure') {
  fs.writeFileSync(output, `${JSON.stringify(report, null, 2)}\n`, {flag: 'wx'});
  process.stdout.write(`Saved ${report.samples.length} checked C storage pairs to ${output}.\n`);
} else {
  process.stdout.write('Both optimized C storage images matched the Go reporter and 1000-round checksum.\n');
}
