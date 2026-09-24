import fs from 'node:fs';
import {spawnSync} from 'node:child_process';

const mode = process.argv[2];
if (mode !== 'smoke' && mode !== 'measure') {
  throw new Error('Expected smoke or measure mode');
}
const output = 'docs/ops/evidence/c-scanner-storage-ab-20260925.json';
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
