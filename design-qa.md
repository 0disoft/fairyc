# Build footprint report design QA

final result: passed

## Evidence and state

- Source: `C:/Users/cherr/.codex/generated_images/01a155ab-49b4-7632-abd2-2485f0bdaa06/exec-f5c2e096-f60d-445e-8b6e-7c5221ad5c46.png` (1487 x 1058).
- Implementation: `docs/benchmarks/latest/report.html#summary`, served locally on port 4183.
- Browser: Codex in-app browser, light theme, English Summary, existing measured receipt.
- CSS viewport: 1487 x 1058. Browser zoom was 90%; viewport override 1338 x 952 compensates for it.
- Raw capture: `.cache/bench-design-qa/summary.png` (1652 x 1174). The screenshot API adds right/bottom canvas outside the page; `.cache/bench-design-qa/summary-normalized.png` crops that canvas to the actual 1487 x 1058 page, without scaling content.
- Full comparison: `.cache/bench-design-qa/comparison.png` (2974 x 1058), source left and implementation right.
- Focused comparison: `.cache/bench-design-qa/detail.png`, logo, sidebar, heading and table at readable size.
- Scroll evidence: `.cache/bench-design-qa/fixed-sidebar.png`.
- Responsive evidence: `.cache/bench-design-qa/mobile-zh.png`, 390 CSS px wide.

## Comparison history

1. Initial capture: P2 density mismatch. Table, menu text and chart rows were substantially smaller than the selected design; the entire summary was enclosed in a card.
2. Fix: match desktop sidebar width, menu spacing, typography, table row heights and chart spacing; give only the table a border. Integrate the existing logo and measured summary rows.
3. Final captures and the full/focused comparisons above: no remaining actionable P0/P1/P2 findings.

## Fidelity review

- Typography: system/Segoe UI fallback, 42px heading and 20–21px desktop table/menu text preserve the reference hierarchy. Generated mock typography differs slightly (P3).
- Layout: fixed sidebar, main content scroll container, table above proportional bars. The sidebar was widened from 260px to 296px in the follow-up below. Compact icon navigation and horizontal table scrolling work at 390px.
- Colors: pale sidebar, white content, mint selected menu, teal totals/project output and blue shared-cache bars match the selected direction.
- Assets: reuse the project logo and bundled Feather icons; the existing logo differs slightly in spacing from the generated reference (P3).
- Content: receipt values match the reference. Existing catalog terminology and accurate cache/measurement notes are retained. The chart scales to the largest actual total, rather than naming Zig as a permanent baseline.

## Interaction checks

- Main content scrolled 277 CSS px while sidebar top remained 0, menu top 130, and document scroll 0.
- Six menu panels, hash navigation, keyboard Home, and switching back to the top of a panel work.
- Rust measurement filter shows six matching rows.
- Growth export points to `growth.svg`; Notes export contains only JSON. Escape closes the export menu.
- Language switching preserves the active panel. English, Chinese, Spanish, French and Hindi pages opened successfully.
- Narrow Chinese layout has no document-wide horizontal overflow; the table scrolls inside its container.
- Browser console: no warnings or errors observed.
- `go test ./tools/dev/fairyc-bench` and `git diff --check`: passed.
- Five-language reports regenerated from the existing receipt; no benchmark rerun or compiler/CI changes.

## Remaining limits

- Minor font/icon and logo-spacing differences from the generated mock are P3 polish.
- No compiler-wide test run: this change only affects report presentation. No API, database, command contract, source-of-truth specification or CI runner changes.

## Follow-up: sidebar width and pinned measurement columns (2026-10-21)

- User-requested adjustments: 296px desktop sidebar and 260px medium-width sidebar. English Measurements and all French menu labels fit on one line at the desktop viewport.
- Language, Profile and Workload headers and cells stay pinned to the left of the measurement scroller. ResizeObserver recalculates offsets from the actual translated header widths after filters, panel visibility or viewport size changes.
- Verified English horizontal scroll of 672.89px: pinned header x positions stayed 366.89, 444.89, 519.56 while Run moved from 625.06 to -47.83. Data cells match their headers and have opaque backgrounds.
- Verified filtering to one Fairy C/dev/tokenizer row updates the narrower column offsets correctly; Korean columns also stay visible at the right edge of the table.
- Screenshots: `.cache/bench-design-qa/pinned-columns.png` and `.cache/bench-design-qa/pinned-columns-ko.png` at the same desktop viewport used above.
- Re-rendered all six HTML reports from the unchanged receipt. Report package tests and diff checks passed; browser console had no warnings or errors.
- final result: passed
