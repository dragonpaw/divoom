// Generative-art backgrounds. A small registry of deterministic
// algorithms — Voronoi, Perlin contours, Recamán, Mandelbrot — keyed by
// the current date so the same day always renders the same art, but
// the wall display rotates through the four pieces across the week.
//
// Pure-Go, no external deps. Each generator returns a full-canvas
// *image.RGBA painted with a gruvbox palette so the result blends with
// the rest of the dashboard.

package render

import (
	"crypto/sha256"
	"encoding/binary"
	"image"
	"image/color"
	"image/draw"
	"math"
	"math/rand/v2"
	"time"
)

// gruvPalette is the rotating accent set every generator picks from.
// Includes bg-darker so large fields don't all blare at full saturation.
var gruvPalette = []color.RGBA{
	GruvBgDarker,
	GruvRed,
	GruvGreen,
	GruvYellow,
	GruvBlue,
	GruvPurple,
	GruvAqua,
	GruvOrange,
	GruvFgDark,
}

// genartGenerator is one entry in the rotation: a name + a renderer.
type genartGenerator struct {
	Name   string
	Render func(seed int64) image.Image
}

// genartRegistry lists every generator in selection order. The
// date-hash modulo this length picks today's algorithm.
var genartRegistry = []genartGenerator{
	{Name: "voronoi", Render: voronoiBackground},
	{Name: "perlin", Render: perlinContoursBackground},
	{Name: "recamán", Render: recamanBackground},
	{Name: "mandelbrot", Render: mandelbrotBackground},
}

// GenartForDate returns the generator's rendered image and its display
// name for the given date. Hashing YYYY-MM-DD makes selection + seeding
// fully deterministic.
func GenartForDate(d time.Time) (image.Image, string) {
	seed := hashDate(d)
	g := genartRegistry[int(uint64(seed)%uint64(len(genartRegistry)))]
	return g.Render(seed), g.Name
}

// GenartBackground encodes the day's generative art to JPEG/PNG —
// mirror of CalendarBackground / HeroBackground signatures.
func GenartBackground(d time.Time, format Format) ([]byte, error) {
	img, _ := GenartForDate(d)
	// All generators return *image.RGBA via blankCanvas; the public
	// GenartForDate type-erases to image.Image so callers reading the
	// preview don't need to know. Assert back for the encoder.
	return encodeImage(img.(*image.RGBA), format)
}

// hashDate is a stable int64 seed derived from the date's YYYY-MM-DD
// representation. SHA-256 truncated to 8 bytes — overkill but cheap,
// and removes any need to think about rand seed weirdness.
func hashDate(d time.Time) int64 {
	day := d.Format("2006-01-02")
	sum := sha256.Sum256([]byte(day))
	return int64(binary.BigEndian.Uint64(sum[:8]))
}

// newRNG builds a deterministic PCG seeded from `seed`.
func newRNG(seed int64) *rand.Rand {
	return rand.New(rand.NewPCG(uint64(seed), uint64(seed)^0xA5A5A5A5))
}

// blankCanvas returns a fresh CanvasW×CanvasH image filled with bg-hard.
func blankCanvas() *image.RGBA {
	img := image.NewRGBA(image.Rect(0, 0, CanvasW, CanvasH))
	draw.Draw(img, img.Bounds(), &image.Uniform{GruvBgHard}, image.Point{}, draw.Src)
	return img
}

// --- Voronoi ---

// voronoiBackground rasterises a closest-site tessellation. ~80 sites,
// each painted with a gruvbox palette colour. Plain L2 distance —
// Lloyd's relaxation would be prettier but the simple form reads fine
// at canvas scale.
func voronoiBackground(seed int64) image.Image {
	r := newRNG(seed)
	const numSites = 80
	type site struct {
		x, y int
		c    color.RGBA
	}
	sites := make([]site, numSites)
	for i := range sites {
		sites[i] = site{
			x: r.IntN(CanvasW),
			y: r.IntN(CanvasH),
			c: gruvPalette[r.IntN(len(gruvPalette))],
		}
	}
	img := blankCanvas()
	// 2×2 down-sampling for speed — every other pixel computed, then
	// the neighbour gets the same site. Visually indistinguishable at
	// the canvas resolution and ~4× faster.
	for y := 0; y < CanvasH; y += 2 {
		for x := 0; x < CanvasW; x += 2 {
			best := 0
			bestD := math.MaxFloat64
			for i, s := range sites {
				dx := float64(s.x - x)
				dy := float64(s.y - y)
				d := dx*dx + dy*dy
				if d < bestD {
					bestD = d
					best = i
				}
			}
			c := sites[best].c
			img.SetRGBA(x, y, c)
			if x+1 < CanvasW {
				img.SetRGBA(x+1, y, c)
			}
			if y+1 < CanvasH {
				img.SetRGBA(x, y+1, c)
				if x+1 < CanvasW {
					img.SetRGBA(x+1, y+1, c)
				}
			}
		}
	}
	return img
}

// --- Perlin contours ---

// perlinContoursBackground draws colour-banded contour lines from a
// simple value-noise field. Bands map to gruvbox palette entries by
// integer floor(value*N).
func perlinContoursBackground(seed int64) image.Image {
	r := newRNG(seed)
	const grid = 8 // value-noise lattice size
	lattice := make([][]float64, grid+1)
	for i := range lattice {
		lattice[i] = make([]float64, grid+1)
		for j := range lattice[i] {
			lattice[i][j] = r.Float64()
		}
	}
	// 2D smooth interpolation across the lattice.
	sample := func(x, y float64) float64 {
		fx := x / float64(CanvasW) * float64(grid)
		fy := y / float64(CanvasH) * float64(grid)
		ix := int(math.Floor(fx))
		iy := int(math.Floor(fy))
		tx := fx - float64(ix)
		ty := fy - float64(iy)
		// Smoothstep for tidier contours than linear.
		sx := tx * tx * (3 - 2*tx)
		sy := ty * ty * (3 - 2*ty)
		a := lattice[ix][iy]*(1-sx) + lattice[ix+1][iy]*sx
		b := lattice[ix][iy+1]*(1-sx) + lattice[ix+1][iy+1]*sx
		return a*(1-sy) + b*sy
	}
	img := blankCanvas()
	bands := len(gruvPalette)
	for y := 0; y < CanvasH; y++ {
		for x := 0; x < CanvasW; x++ {
			v := sample(float64(x), float64(y))
			idx := int(v * float64(bands))
			if idx >= bands {
				idx = bands - 1
			}
			img.SetRGBA(x, y, gruvPalette[idx])
		}
	}
	return img
}

// --- Recamán ---

// recamanBackground draws the Recamán sequence as alternating
// semicircles across rows; gruvbox-coloured by index. Classic
// 2blue1brown visualisation, rendered down the canvas.
func recamanBackground(seed int64) image.Image {
	img := blankCanvas()
	const (
		step  = 18 // pixel scale per sequence step
		baseY = CanvasH / 2
	)
	// Generate the first ~120 terms; that's plenty to fill the canvas
	// at step=18 (max reach ~ 60*18 = 1080 px from origin).
	const n = 120
	seq := make([]int, n)
	seen := map[int]bool{0: true}
	for i := 1; i < n; i++ {
		back := seq[i-1] - i
		if back > 0 && !seen[back] {
			seq[i] = back
		} else {
			seq[i] = seq[i-1] + i
		}
		seen[seq[i]] = true
	}
	// Render each step as a half-circle between adjacent terms.
	originX := CanvasW / 2
	for i := 1; i < n; i++ {
		a := seq[i-1]
		b := seq[i]
		mid := (a + b) / 2
		radius := (b - a) / 2
		if radius < 0 {
			radius = -radius
		}
		cx := originX + mid*step/4
		// Wrap centres back into the canvas so the figure doesn't
		// march off into the void.
		for cx > CanvasW-40 {
			cx -= CanvasW - 80
		}
		for cx < 40 {
			cx += CanvasW - 80
		}
		c := gruvPalette[(i+int(uint64(seed)%9))%len(gruvPalette)]
		above := i%2 == 0
		drawArc(img, cx, baseY, radius*step/8, above, c)
	}
	return img
}

// drawArc paints a half-circle outline (1px) centred at (cx, cy) with
// the given radius. `above` flips between the top and bottom halves.
func drawArc(img *image.RGBA, cx, cy, radius int, above bool, c color.RGBA) {
	if radius < 2 {
		return
	}
	for t := 0; t <= 180; t++ {
		theta := float64(t) * math.Pi / 180.0
		x := cx + int(math.Cos(theta)*float64(radius))
		y := cy
		if above {
			y -= int(math.Sin(theta) * float64(radius))
		} else {
			y += int(math.Sin(theta) * float64(radius))
		}
		// Paint a 2px-thick dot so the arc reads against the field.
		for dy := -1; dy <= 1; dy++ {
			for dx := -1; dx <= 1; dx++ {
				px, py := x+dx, y+dy
				if px >= 0 && px < CanvasW && py >= 0 && py < CanvasH {
					img.SetRGBA(px, py, c)
				}
			}
		}
	}
}

// --- Mandelbrot ---

// mandelbrotBackground renders a Mandelbrot region around a
// deterministic-per-seed point with a gruvbox iteration-count palette.
// Zoom + offset are seeded so each day picks a different patch but
// the same date always returns the same view.
func mandelbrotBackground(seed int64) image.Image {
	r := newRNG(seed)
	// Centre point — wander inside the classic [-2..1, -1..1] box.
	cx0 := -0.7 + (r.Float64()-0.5)*0.6
	cy0 := 0.0 + (r.Float64()-0.5)*0.6
	zoom := 1.0 + r.Float64()*2.5 // 1×..3.5× the default span
	const maxIter = 100
	img := blankCanvas()
	spanX := 3.0 / zoom
	spanY := spanX * float64(CanvasH) / float64(CanvasW)
	for py := 0; py < CanvasH; py++ {
		for px := 0; px < CanvasW; px++ {
			x0 := cx0 + (float64(px)/float64(CanvasW)-0.5)*spanX
			y0 := cy0 + (float64(py)/float64(CanvasH)-0.5)*spanY
			x, y := 0.0, 0.0
			iter := 0
			for iter < maxIter && x*x+y*y <= 4.0 {
				xt := x*x - y*y + x0
				y = 2*x*y + y0
				x = xt
				iter++
			}
			var c color.RGBA
			if iter == maxIter {
				c = GruvBgHard
			} else {
				c = gruvPalette[iter%len(gruvPalette)]
			}
			img.SetRGBA(px, py, c)
		}
	}
	return img
}
