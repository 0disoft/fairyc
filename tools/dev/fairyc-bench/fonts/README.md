# Report fonts

The benchmark embeds renamed subsets of Noto Sans, Noto Sans SC and Noto Sans Devanagari
under the SIL Open Font License 1.1. Their license and copyright notices are in the adjacent
`OFL-*.txt` files. SVG text uses glyph outlines; PNG uses the same shaped outlines.
No installed fonts or font downloads are needed to render reports.

Sources from [Google Fonts](https://github.com/google/fonts/tree/main/ofl):

| Source | SHA-256 |
| --- | --- |
| `notosans/NotoSans[wdth,wght].ttf` | `bfb7bb691513f12e734dc346c03a03f784912432d7e3fa8e56efcf906fe86b3d` |
| `notosanssc/NotoSansSC[wght].ttf` | `a3041811a78c361b1de50f953c805e0244951c21c5bd412f7232ef0d899af0da` |
| `notosansdevanagari/NotoSansDevanagari[wdth,wght].ttf` | `14ec4af41f27482216d1c2229f417ff9b1425e1babb014e57d1d40d03229853e` |

To rebuild subsets after adding translated characters, download those files, rename them
`NotoSans.ttf`, `NotoSansSC.ttf` and `NotoSansDevanagari.ttf`, install `fonttools`, and run:

```text
python tools/dev/fairyc-bench/fonts/subset.py SOURCE_DIRECTORY
```

The script keeps regular weight and the characters used in the locale catalogs, plus Latin
and punctuation. It preserves OpenType substitution tables for Hindi shaping and renames the
modified families to `BenchSans`, `BenchChinese` and `BenchDevanagari`.

The benchmark renderer uses `github.com/go-text/typesetting` (BSD-3-Clause or Unlicense) and
`golang.org/x/image` (BSD-3-Clause). Full notices are in `LICENSE-go-text.txt` and
`LICENSE-x-image.txt`. These packages are used by the benchmark tool, not the compiler.
