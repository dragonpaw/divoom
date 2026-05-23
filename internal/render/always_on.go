// Bakes the always-on header (day name, time, footer, weekend status)
// into a rendered scene JPG. Used only by `divoom render` so the
// screenshot tree in dist/scenes/ looks like a wall-installed frame
// — at runtime these elements are installed as device Text/Time/Week
// elements by the daemon and live atop an otherwise-empty header.

package render

import (
	"bytes"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/jpeg"
	"strings"
	"time"

	"golang.org/x/image/font"
	"golang.org/x/image/font/opentype"
)

// BakeAlwaysOnHeaderJPEG decodes a scene JPG, paints the always-on
// header onto it (day name, time, footer, weekend-countdown), and
// re-encodes. Mirrors the layout produced at runtime by `alwaysOn`
// in cmd/divoom/scenes.go.
func BakeAlwaysOnHeaderJPEG(in []byte, now time.Time) ([]byte, error) {
	src, err := jpeg.Decode(bytes.NewReader(in))
	if err != nil {
		return nil, fmt.Errorf("decode jpeg: %w", err)
	}
	img := image.NewRGBA(src.Bounds())
	draw.Draw(img, img.Bounds(), src, image.Point{}, draw.Src)
	if err := bakeAlwaysOnHeader(img, now); err != nil {
		return nil, err
	}
	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, img, &jpeg.Options{Quality: 95}); err != nil {
		return nil, fmt.Errorf("encode jpeg: %w", err)
	}
	return buf.Bytes(), nil
}

// gruvWeekday picks a gruvbox accent per weekday, mirroring dayColors
// in cmd/divoom/scenes.go. Kept here (not exported from main) so the
// render package can bake the header without a circular import.
func gruvWeekday(d time.Weekday) color.RGBA {
	switch d {
	case time.Sunday:
		return GruvPurple
	case time.Monday:
		return GruvRed
	case time.Tuesday:
		return GruvOrange
	case time.Wednesday:
		return GruvYellow
	case time.Thursday:
		return GruvGreen
	case time.Friday:
		return GruvAqua
	default: // Saturday
		return GruvBlue
	}
}

// gruvTimeColor mirrors timeColor: aqua mornings, orange afternoons.
func gruvTimeColor(now time.Time) color.RGBA {
	if now.Hour() < 12 {
		return GruvAqua
	}
	return GruvOrange
}

// weekendCountdown mirrors weekendStatus in cmd/divoom/scenes.go.
func weekendCountdown(now time.Time) (text string, c color.RGBA) {
	wd := now.Weekday()
	hour := now.Hour()
	weekend := false
	switch wd {
	case time.Saturday, time.Sunday:
		weekend = true
	case time.Friday:
		weekend = hour >= 18
	case time.Monday:
		weekend = hour < 3
	}
	if weekend {
		return "weekend!", GruvYellow
	}
	n := 5 - int(wd)
	if n < 0 {
		n = 0
	}
	return fmt.Sprintf("weekend-%dd", n), GruvFgDark
}

func bakeAlwaysOnHeader(img *image.RGBA, now time.Time) error {
	mono, err := LoadFont("Iosevka-Regular.ttf")
	if err != nil {
		return fmt.Errorf("load mono: %w", err)
	}

	dayFace, err := opentype.NewFace(mono, &opentype.FaceOptions{
		Size: 64, DPI: 72, Hinting: font.HintingFull,
	})
	if err != nil {
		return err
	}
	defer dayFace.Close()
	// Weekday().String() returns title case ("Wednesday"); the device's
	// Week element renders lower case ("wednesday") — match that.
	drawLabelLeft(img, strings.ToLower(now.Weekday().String()), dayFace,
		110, 90, gruvWeekday(now.Weekday()))

	timeFace, err := opentype.NewFace(mono, &opentype.FaceOptions{
		Size: 160, DPI: 72, Hinting: font.HintingFull,
	})
	if err != nil {
		return err
	}
	defer timeFace.Close()
	drawLabelCentered(img, now.Format("15:04"), timeFace, 400, 290,
		gruvTimeColor(now))

	footerFace, err := opentype.NewFace(mono, &opentype.FaceOptions{
		Size: 28, DPI: 72, Hinting: font.HintingFull,
	})
	if err != nil {
		return err
	}
	defer footerFace.Close()
	footer := fmt.Sprintf("%s  doy:%d  w:%d",
		now.Format("2006-01-02"), now.YearDay(), isoWeek(now))
	drawLabelLeft(img, footer, footerFace, 40, 425, GruvFgDark)

	weekText, weekColor := weekendCountdown(now)
	drawLabelRight(img, weekText, footerFace, 720, 425, weekColor)
	return nil
}

// isoWeek is the ISO 8601 week number, mirroring the helper in
// cmd/divoom/scenes.go.
func isoWeek(now time.Time) int {
	_, w := now.ISOWeek()
	return w
}
