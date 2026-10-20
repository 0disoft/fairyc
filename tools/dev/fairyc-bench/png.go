package main

import (
	"fmt"
	"github.com/go-text/typesetting/font/opentype"
	"golang.org/x/image/vector"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"math"
	"os"
)

func paint(hex string) color.RGBA {
	var r, g, b uint8
	_, _ = fmt.Sscanf(hex, "#%02x%02x%02x", &r, &g, &b)
	return color.RGBA{r, g, b, 255}
}
func (s scene) png(path string) error {
	const scale = 2
	canvas := image.NewRGBA(image.Rect(0, 0, s.Width*scale, s.Height*scale))
	draw.Draw(canvas, canvas.Bounds(), image.NewUniform(paint("#f8fafc")), image.Point{}, draw.Src)
	rect := func(x, y, w, h int, c color.RGBA) {
		draw.Draw(canvas, image.Rect(x, y, x+w, y+h), image.NewUniform(c), image.Point{}, draw.Src)
	}
	for _, p := range s.Shapes {
		c := paint(p.Color)
		switch p.Kind {
		case "rect":
			rect(int(p.X*scale), int(p.Y*scale), max(1, int(p.W*scale)), max(1, int(p.H*scale)), c)
		case "text":
			paths := textOutline(p, s.Locale, float64(s.Width)-p.X-32)
			if len(paths) == 0 {
				continue
			}
			minX, minY, maxX, maxY := math.Inf(1), math.Inf(1), math.Inf(-1), math.Inf(-1)
			for _, segment := range paths {
				for _, pt := range segment.ArgsSlice() {
					minX = min(minX, float64(pt.X)*scale)
					minY = min(minY, float64(pt.Y)*scale)
					maxX = max(maxX, float64(pt.X)*scale)
					maxY = max(maxY, float64(pt.Y)*scale)
				}
			}
			bounds := image.Rect(int(math.Floor(minX))-1, int(math.Floor(minY))-1, int(math.Ceil(maxX))+1, int(math.Ceil(maxY))+1)
			z := vector.NewRasterizer(bounds.Dx(), bounds.Dy())
			started := false
			for _, segment := range paths {
				pts := segment.Args
				for i := range pts {
					pts[i].X = pts[i].X*scale - float32(bounds.Min.X)
					pts[i].Y = pts[i].Y*scale - float32(bounds.Min.Y)
				}
				switch segment.Op {
				case opentype.SegmentOpMoveTo:
					if started {
						z.ClosePath()
					}
					z.MoveTo(pts[0].X, pts[0].Y)
					started = true
				case opentype.SegmentOpLineTo:
					z.LineTo(pts[0].X, pts[0].Y)
				case opentype.SegmentOpQuadTo:
					z.QuadTo(pts[0].X, pts[0].Y, pts[1].X, pts[1].Y)
				case opentype.SegmentOpCubeTo:
					z.CubeTo(pts[0].X, pts[0].Y, pts[1].X, pts[1].Y, pts[2].X, pts[2].Y)
				}
			}
			z.ClosePath()
			z.Draw(canvas, bounds, image.NewUniform(c), image.Point{})
		case "line":
			for i, a := range p.Points {
				b := a
				if i > 0 {
					b = p.Points[i-1]
				}
				dx, dy := (b.X-a.X)*scale, (b.Y-a.Y)*scale
				steps := max(1, int(math.Max(math.Abs(dx), math.Abs(dy))))
				for j := 0; j <= steps; j++ {
					t := float64(j) / float64(steps)
					rect(int(a.X*scale+dx*t)-1, int(a.Y*scale+dy*t)-1, 4, 4, c)
				}
			}
		}
	}
	file, err := os.Create(path)
	if err != nil {
		return err
	}
	err = png.Encode(file, canvas)
	closeErr := file.Close()
	if err != nil {
		return err
	}
	return closeErr
}
