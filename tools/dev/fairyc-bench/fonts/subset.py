"""Regenerate report font subsets: python subset.py SOURCE_DIRECTORY (fonttools required)."""
import json
from pathlib import Path
import sys

from fontTools import subset
from fontTools.ttLib import TTFont
from fontTools.varLib.instancer import instantiateVariableFont

destination = Path(__file__).resolve().parent
characters = set(range(0x20, 0x250)) | set(range(0x2000, 0x2070))
for catalog in (destination.parent / "locales").glob("*.json"):
    for text in json.loads(catalog.read_text(encoding="utf-8")).values():
        characters.update(map(ord, text))

for source, name in [("NotoSans.ttf", "BenchSans"),
                     ("NotoSansDevanagari.ttf", "BenchDevanagari"),
                     ("NotoSansSC.ttf", "BenchChinese"),
                     ("NotoSansKR.ttf", "BenchKorean")]:
    font = TTFont(Path(sys.argv[1]) / source)
    axes = {axis.axisTag: axis.defaultValue for axis in font["fvar"].axes}
    axes["wght"] = 400
    font = instantiateVariableFont(font, axes, inplace=True)
    options = subset.Options()
    options.layout_features = ["*"]  # Keep Hindi shaping and substitution closure.
    options.recalc_timestamp = False
    sub = subset.Subsetter(options=options)
    sub.populate(unicodes=characters)
    sub.subset(font)
    for record in font["name"].names:
        if record.nameID in (1, 3, 4, 6, 16, 25):
            record.string = name.encode(record.getEncoding())
        elif record.nameID in (2, 17):
            record.string = "Regular".encode(record.getEncoding())
    output = destination / (name + ".ttf")
    font.save(output)
    print(output.name, output.stat().st_size)
