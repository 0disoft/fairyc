# Lexer consumer cost observations: 2026-09-10

## Interpretation

These are bounded local test observations, not an optimization comparison, a
performance gate, peak-memory evidence, or a pure lexer-latency measurement.
No optimization is justified from this snapshot alone.

The optimized empty-driver median is 29.064 ms, versus 31.275 ms for ten small
source cycles. Fixed driver overhead dominates that small-input comparison.
Token-heavy input also requires inspecting 10,206 token records per cycle; this
experiment does not separate scanning, allocation, inspection and destruction.

## Measurement contract

- Schema: `fairyc.lexer-cost-observation/v1`.
- Command: `fairyc_go_measure_lexer_stream`; opt-in test flag
  `-fairyc-lexer-cost`. Ordinary correctness tests skip the measurement.
- Target: Windows AMD64 MSVC; toolset 14.51.36231; go1.26.4;
  12 logical CPUs. Shared developer machine, no fixed affinity,
  no dedicated benchmark isolation, no hardware counters.
- Each non-empty process performs ten source Buffer copies, complete collection,
  logical storage inspection/assertions and scoped destruction. Empty-driver runs
  zero cycles and creates no Stream.
- One untimed successful warmup process, then five fresh successful processes for
  each scenario/profile, concurrency one. Profiles run in fixed low/optimized
  order; no cross-profile speedup inference or statistical regression threshold.
- The monotonic host timer surrounds the entire native driver call: temporary PE
  materialization, process launch, workload, output/trap capture and cleanup.
  Compilation is outside the interval. OS caches are not flushed. Within-process
  cycles reuse the same static input; native lexer results are not cached.
- Every timed result must exit successfully with no output or trap and match the
  Go reference's token and logical payload counts. Failures stop the experiment;
  there is no retry-to-improve policy.
- Baseline compiler revision: `ea7a7c3272d5802dcded99f38cc88818ae0fbe12`,
  with the scanner storage-inspection API added for this measurement.
  Candidate source-closure SHA-256:
  `def3e6cae13831e77aac243d76c491dc0b2841b10966aa57c83e2959b0e81604`.
  The test hashes framed logical filenames and exact bytes for the manifest and
  six copied source inputs; native program and PE hashes follow below.

## Driver-roundtrip medians

Milliseconds per complete process, rounded to three decimals; not per token or
per isolated collection. Empty-driver is an overhead reference, not subtracted
from other scenarios.

| Scenario | Profile | Cycles | Median ms |
| --- | --- | ---: | ---: |
| empty-driver | low | 0 | 26.608 |
| small-source | low | 10 | 30.771 |
| token-heavy-16k | low | 10 | 258.095 |
| literal-10k | low | 10 | 99.984 |
| empty-driver | optimized | 0 | 29.064 |
| small-source | optimized | 10 | 31.275 |
| token-heavy-16k | optimized | 10 | 61.717 |
| literal-10k | optimized | 10 | 35.801 |

## Logical retained data per live Stream

| Scenario | Source bytes | Owned cooked bytes | Token records | Payload-byte subtotal |
| --- | ---: | ---: | ---: | ---: |
| small-source | 41 | 2 | 22 | 43 |
| token-heavy-16k | 16384 | 1200 | 10206 | 17584 |
| literal-10k | 10002 | 20000 | 2 | 30002 |

All three have zero diagnostic-token bytes. The long string's cooked
representation is hexadecimal text, explaining its 20,000 retained cooked bytes.
Keyword cooked views alias source bytes and are not counted as separately owned.

These subtotals exclude token/diagnostic metadata, vector spare capacity, Buffer
capacity, allocator headers, compiler data, process memory and transient peaks.
They are payload lower bounds, not heap/RSS values or leak checks. In particular,
the token-heavy case has much more metadata than the literal case despite its
smaller payload subtotal. Peak heap/RSS and allocation counts remain unmeasured.

## Replay identities and raw samples

Only logical scenario names, version information, hashes and numeric samples are
retained. No absolute paths, environments, arguments, raw heap or source dumps.

```json
[
  {
    "scenario": "empty-driver",
    "profile": "low",
    "program_sha256": "96bae0e23f4c113355c25becf5310380c425eddfe678b6b8d22102be75d075fe",
    "pe_sha256": "91b0bec2a3ae9717d1d680cfd3a9df40bc5882f21c7d0a6fb2e3c81eff72aef9",
    "driver_roundtrip_ms": [
      25.668400000000002,
      26.8266,
      25.9164,
      27.988599999999998,
      26.6077
    ]
  },
  {
    "scenario": "small-source",
    "profile": "low",
    "program_sha256": "31b172b4b3948039bc74543c7b030f522321d6faed5cb7aba8a81b04a78b5068",
    "pe_sha256": "6dabb08c99a6658740ca09a6fc4cae1a75aa83d3eb5b52fe3b359b3e00641dfb",
    "driver_roundtrip_ms": [
      28.1442,
      28.3644,
      33.5992,
      33.108,
      30.7707
    ]
  },
  {
    "scenario": "token-heavy-16k",
    "profile": "low",
    "program_sha256": "7972d5e5b1ac51ddfad6ed6c13f8b34c1fb7d54549485063584ec640db406fc1",
    "pe_sha256": "047ced36af9dbf25f97d4dcaa6c4e738de6f8fc94d72eb7f01b82b294de48d00",
    "driver_roundtrip_ms": [
      258.0951,
      258.9742,
      262.5268,
      253.3955,
      251.43110000000001
    ]
  },
  {
    "scenario": "literal-10k",
    "profile": "low",
    "program_sha256": "8c0c42cc17335272dfa43b42a97fdb0c7216948be241c974e0f4c18fa6e1f815",
    "pe_sha256": "8fb64e4beafa61ca97f90afe531e8db5e99fb390f68cea9b397cfa2b4d710ccc",
    "driver_roundtrip_ms": [
      99.98400000000001,
      107.54990000000001,
      97.99130000000001,
      101.4977,
      92.9633
    ]
  },
  {
    "scenario": "empty-driver",
    "profile": "optimized",
    "program_sha256": "96bae0e23f4c113355c25becf5310380c425eddfe678b6b8d22102be75d075fe",
    "pe_sha256": "c139218f1e4b2fc8a5cbbca4ab41bae9d415c25ce949430ffb0a8633b2f41bd0",
    "driver_roundtrip_ms": [
      31.3058,
      28.291,
      29.0641,
      28.9856,
      31.327
    ]
  },
  {
    "scenario": "small-source",
    "profile": "optimized",
    "program_sha256": "31b172b4b3948039bc74543c7b030f522321d6faed5cb7aba8a81b04a78b5068",
    "pe_sha256": "3345eecae76d65d9999772cbdffcabd1d12eccdc98b18d9a3ba8db6294c8bb3d",
    "driver_roundtrip_ms": [
      31.2754,
      28.9862,
      32.6323,
      31.362800000000004,
      28.5802
    ]
  },
  {
    "scenario": "token-heavy-16k",
    "profile": "optimized",
    "program_sha256": "7972d5e5b1ac51ddfad6ed6c13f8b34c1fb7d54549485063584ec640db406fc1",
    "pe_sha256": "73076f8edcff08af6eb001d3af12d22bd59373b268aac95cb84e992648a266d8",
    "driver_roundtrip_ms": [
      69.5624,
      61.49,
      62.394999999999996,
      60.4461,
      61.7165
    ]
  },
  {
    "scenario": "literal-10k",
    "profile": "optimized",
    "program_sha256": "8c0c42cc17335272dfa43b42a97fdb0c7216948be241c974e0f4c18fa6e1f815",
    "pe_sha256": "79f9fe98af2aed0c43044ae7593446243115cb4b6f3db6639704f31b447ec997",
    "driver_roundtrip_ms": [
      35.8013,
      36.487899999999996,
      37.125699999999995,
      34.96,
      34.9678
    ]
  }
]
```
