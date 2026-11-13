package chart

import (
	"bytes"
	"image"
	"strings"
	"testing"

	"github.com/wcharczuk/go-chart/v2/drawing"
	"github.com/wcharczuk/go-chart/v2/testutil"
)

// renderStroke draws a single stroked path with the given style and returns
// the raw RGBA pixels.
func renderStroke(t *testing.T, width, height int, style Style, path func(r Renderer)) *image.RGBA {
	t.Helper()

	r, err := PNG(width, height)
	testutil.AssertNil(t, err)

	style.WriteDrawingOptionsToRenderer(r)
	path(r)
	r.Stroke()

	iw := &ImageWriter{}
	testutil.AssertNil(t, r.Save(iw))
	img, err := iw.Image()
	testutil.AssertNil(t, err)

	typed, isTyped := img.(*image.RGBA)
	testutil.AssertTrue(t, isTyped)
	return typed
}

func assertPixelFilled(t *testing.T, img *image.RGBA, x, y int) {
	t.Helper()
	_, _, _, a := img.At(x, y).RGBA()
	testutil.AssertTrue(t, a > 0x8000, "expected filled pixel at", x, y)
}

func assertPixelEmpty(t *testing.T, img *image.RGBA, x, y int) {
	t.Helper()
	_, _, _, a := img.At(x, y).RGBA()
	testutil.AssertTrue(t, a < 0x8000, "expected empty pixel at", x, y)
}

func horizontalLine(r Renderer) {
	r.MoveTo(20, 100)
	r.LineTo(180, 100)
}

// TestStrokeDefaultMatchesExplicitButtBevel proves that leaving the cap and
// join unset produces exactly the historical output: butt caps and bevel
// joins.
func TestStrokeDefaultMatchesExplicitButtBevel(t *testing.T) {
	zigzag := func(r Renderer) {
		r.MoveTo(20, 60)
		r.LineTo(100, 140)
		r.LineTo(180, 60)
	}

	unset := renderStroke(t, 200, 200, Style{
		StrokeColor: drawing.ColorBlack,
		StrokeWidth: 20,
	}, zigzag)

	explicit := renderStroke(t, 200, 200, Style{
		StrokeColor:    drawing.ColorBlack,
		StrokeWidth:    20,
		StrokeLineCap:  LineCapButt,
		StrokeLineJoin: LineJoinBevel,
	}, zigzag)

	testutil.AssertTrue(t, bytes.Equal(unset.Pix, explicit.Pix),
		"unset cap/join should render identically to explicit butt/bevel")
}

// TestStrokeButtCapFlushEnds proves a butt cap stops exactly at the end of
// the line.
func TestStrokeButtCapFlushEnds(t *testing.T) {
	img := renderStroke(t, 200, 200, Style{
		StrokeColor:   drawing.ColorBlack,
		StrokeWidth:   20,
		StrokeLineCap: LineCapButt,
	}, horizontalLine)

	assertPixelFilled(t, img, 22, 100)
	assertPixelFilled(t, img, 177, 100)
	assertPixelEmpty(t, img, 17, 100)
	assertPixelEmpty(t, img, 183, 100)
}

// TestStrokeSquareCapExtendsHalfLineWidth proves a square cap extends the
// stroke by half the line width (10px for a 20px stroke) past each endpoint.
func TestStrokeSquareCapExtendsHalfLineWidth(t *testing.T) {
	img := renderStroke(t, 200, 200, Style{
		StrokeColor:   drawing.ColorBlack,
		StrokeWidth:   20,
		StrokeLineCap: LineCapSquare,
	}, horizontalLine)

	// inside the cap extensions: endpoints at x=20 and x=180, half width 10
	assertPixelFilled(t, img, 14, 100)
	assertPixelFilled(t, img, 185, 100)
	// beyond the cap extensions (edges at x=10 and x=190)
	assertPixelEmpty(t, img, 7, 100)
	assertPixelEmpty(t, img, 192, 100)
}

// TestStrokeRoundCap proves a round cap bulges a semicircle of radius half
// the line width around each endpoint.
func TestStrokeRoundCap(t *testing.T) {
	img := renderStroke(t, 200, 200, Style{
		StrokeColor:   drawing.ColorBlack,
		StrokeWidth:   20,
		StrokeLineCap: LineCapRound,
	}, horizontalLine)

	// 8px past the endpoint, within the radius-10 semicircle
	assertPixelFilled(t, img, 188, 100)
	assertPixelFilled(t, img, 12, 100)
	// 12px past the endpoint, outside the semicircle
	assertPixelEmpty(t, img, 192, 100)
	assertPixelEmpty(t, img, 8, 100)
	// 12px off the axis below the endpoint, outside the semicircle
	assertPixelEmpty(t, img, 180, 112)
}

// zigzagPeak is a sharp peak at (100, 40) with 20px stroke; the outer
// bevel chord sits at y=35.53 and the miter tip at y=17.64.
func zigzagPeak(r Renderer) {
	r.MoveTo(50, 140)
	r.LineTo(100, 40)
	r.LineTo(150, 140)
}

// TestStrokeMiterJoinFillsTip proves a miter join fills the outer spike of
// a sharp corner that a bevel join cuts off.
func TestStrokeMiterJoinFillsTip(t *testing.T) {
	img := renderStroke(t, 200, 200, Style{
		StrokeColor:    drawing.ColorBlack,
		StrokeWidth:    20,
		StrokeLineJoin: LineJoinMiter,
	}, zigzagPeak)

	// between the bevel chord (y=35.53) and the miter tip (y=17.64)
	assertPixelFilled(t, img, 100, 22)
	// above the miter tip
	assertPixelEmpty(t, img, 100, 14)
}

// TestStrokeBevelJoinCutsTip proves a bevel join cuts the corner at the
// chord between the two outer offsets.
func TestStrokeBevelJoinCutsTip(t *testing.T) {
	img := renderStroke(t, 200, 200, Style{
		StrokeColor:    drawing.ColorBlack,
		StrokeWidth:    20,
		StrokeLineJoin: LineJoinBevel,
	}, zigzagPeak)

	// above the chord (y=35.53): cut off
	assertPixelEmpty(t, img, 100, 30)
	// below the chord: still stroked
	assertPixelFilled(t, img, 100, 38)
}

// TestStrokeRoundJoin proves a round join fills the corner out to a radius
// of half the line width around the vertex.
func TestStrokeRoundJoin(t *testing.T) {
	img := renderStroke(t, 200, 200, Style{
		StrokeColor:    drawing.ColorBlack,
		StrokeWidth:    20,
		StrokeLineJoin: LineJoinRound,
	}, zigzagPeak)

	// 7px from the vertex (100, 40): inside the radius-10 arc but above
	// the bevel chord
	assertPixelFilled(t, img, 100, 33)
	// 18px from the vertex: outside the arc, and no miter spike
	assertPixelEmpty(t, img, 100, 22)
}

// TestStrokeMiterLimitDegradesToBevel proves a miter join degrades to a
// bevel join when the miter ratio exceeds the configured limit, and that
// the limit is configurable. The peak has a miter ratio of ~2.24.
func TestStrokeMiterLimitDegradesToBevel(t *testing.T) {
	overLimit := renderStroke(t, 200, 200, Style{
		StrokeColor:      drawing.ColorBlack,
		StrokeWidth:      20,
		StrokeLineJoin:   LineJoinMiter,
		StrokeMiterLimit: 2.0,
	}, zigzagPeak)

	// ratio 2.24 > limit 2.0: bevel, the tip is cut at the chord
	assertPixelEmpty(t, overLimit, 100, 30)
	assertPixelFilled(t, overLimit, 100, 38)

	underLimit := renderStroke(t, 200, 200, Style{
		StrokeColor:      drawing.ColorBlack,
		StrokeWidth:      20,
		StrokeLineJoin:   LineJoinMiter,
		StrokeMiterLimit: 3.0,
	}, zigzagPeak)

	// ratio 2.24 < limit 3.0: miter, the tip is filled
	assertPixelFilled(t, underLimit, 100, 22)
}

// TestStrokeClosedPathHasNoCaps proves closed subpaths get joins, not caps,
// at the closure vertex: a square cap must not extend the outline there.
func TestStrokeClosedPathHasNoCaps(t *testing.T) {
	square := func(r Renderer) {
		r.MoveTo(60, 60)
		r.LineTo(140, 60)
		r.LineTo(140, 140)
		r.LineTo(60, 140)
		r.Close()
	}

	beveled := renderStroke(t, 200, 200, Style{
		StrokeColor:    drawing.ColorBlack,
		StrokeWidth:    20,
		StrokeLineCap:  LineCapSquare,
		StrokeLineJoin: LineJoinBevel,
	}, square)

	// outside the bevel chord (x+y=110) at the closure corner (60, 60);
	// a stray square cap would fill this
	assertPixelEmpty(t, beveled, 52, 52)
	// sanity: the stroke body is there
	assertPixelFilled(t, beveled, 70, 52)

	mitered := renderStroke(t, 200, 200, Style{
		StrokeColor:    drawing.ColorBlack,
		StrokeWidth:    20,
		StrokeLineCap:  LineCapSquare,
		StrokeLineJoin: LineJoinMiter,
	}, square)

	// the 90 degree closure corner miters exactly to (50, 50)
	assertPixelFilled(t, mitered, 52, 52)
}

// TestStrokeDashArrayCapsEachDash proves every dash of a dashed stroke is
// capped with the configured cap.
func TestStrokeDashArrayCapsEachDash(t *testing.T) {
	longLine := func(r Renderer) {
		r.MoveTo(20, 100)
		r.LineTo(220, 100)
	}
	dashStyle := Style{
		StrokeColor:     drawing.ColorBlack,
		StrokeWidth:     20,
		StrokeDashArray: []float64{40, 40},
		StrokeLineCap:   LineCapSquare,
	}
	// dashes cover [20,60], [100,140], [180,220]; square caps extend each
	// by 10 on both ends: [10,70], [90,150], [170,230]
	img := renderStroke(t, 240, 200, dashStyle, longLine)

	assertPixelFilled(t, img, 12, 100) // square cap of first dash
	assertPixelFilled(t, img, 95, 100) // square cap of second dash
	assertPixelEmpty(t, img, 80, 100)  // gap between capped dashes
	assertPixelEmpty(t, img, 160, 100) // gap between capped dashes

	// with butt caps the dash ends stay flush with the dash pattern
	buttStyle := dashStyle
	buttStyle.StrokeLineCap = LineCapButt
	butt := renderStroke(t, 240, 200, buttStyle, longLine)

	assertPixelEmpty(t, butt, 12, 100)
	assertPixelEmpty(t, butt, 95, 100)
	assertPixelFilled(t, butt, 25, 100)
}

// TestVectorRendererStrokeLineCapJoin proves the SVG backend emits the
// same-named attributes matching the style.
func TestVectorRendererStrokeLineCapJoin(t *testing.T) {
	vr, err := SVG(200, 200)
	testutil.AssertNil(t, err)

	style := Style{
		StrokeColor:      drawing.ColorBlack,
		StrokeWidth:      20,
		StrokeLineCap:    LineCapSquare,
		StrokeLineJoin:   LineJoinMiter,
		StrokeMiterLimit: 8,
	}
	style.WriteToRenderer(vr)

	vr.MoveTo(20, 100)
	vr.LineTo(180, 100)
	vr.Stroke()

	buffer := bytes.NewBuffer([]byte{})
	testutil.AssertNil(t, vr.Save(buffer))
	raw := buffer.String()

	testutil.AssertContains(t, raw, "stroke-width:20")
	testutil.AssertContains(t, raw, "stroke-linecap:square")
	testutil.AssertContains(t, raw, "stroke-linejoin:miter")
	testutil.AssertContains(t, raw, "stroke-miterlimit:8")
}

// TestVectorRendererStrokeLineCapJoinUnset proves the SVG output is
// unchanged when no cap, join or miter limit is set.
func TestVectorRendererStrokeLineCapJoinUnset(t *testing.T) {
	vr, err := SVG(200, 200)
	testutil.AssertNil(t, err)

	vr.SetStrokeColor(drawing.ColorBlack)
	vr.SetStrokeWidth(20)
	vr.MoveTo(20, 100)
	vr.LineTo(180, 100)
	vr.Stroke()

	buffer := bytes.NewBuffer([]byte{})
	testutil.AssertNil(t, vr.Save(buffer))
	raw := buffer.String()

	testutil.AssertFalse(t, strings.Contains(raw, "stroke-linecap"),
		"unset line cap should not emit stroke-linecap")
	testutil.AssertFalse(t, strings.Contains(raw, "stroke-linejoin"),
		"unset line join should not emit stroke-linejoin")
	testutil.AssertFalse(t, strings.Contains(raw, "stroke-miterlimit"),
		"unset miter limit should not emit stroke-miterlimit")
}

// TestStrokeBackendsAgreeOnCaps proves the PNG and SVG backends carry the
// same cap and join settings from one style.
func TestStrokeBackendsAgreeOnCaps(t *testing.T) {
	style := Style{
		StrokeColor:      drawing.ColorBlack,
		StrokeWidth:      20,
		StrokeLineCap:    LineCapRound,
		StrokeLineJoin:   LineJoinRound,
		StrokeMiterLimit: 6,
	}

	// raster side: the style reaches the graphic context
	rr, err := PNG(200, 200)
	testutil.AssertNil(t, err)
	style.WriteDrawingOptionsToRenderer(rr)
	rr.MoveTo(20, 100)
	rr.LineTo(180, 100)
	rr.Stroke()
	iw := &ImageWriter{}
	testutil.AssertNil(t, rr.Save(iw))
	img, err := iw.Image()
	testutil.AssertNil(t, err)
	rgba, isRGBA := img.(*image.RGBA)
	testutil.AssertTrue(t, isRGBA)
	// round cap: 8px past the endpoint is filled
	assertPixelFilled(t, rgba, 188, 100)

	// vector side: the same style emits the matching attributes
	vr, err := SVG(200, 200)
	testutil.AssertNil(t, err)
	style.WriteDrawingOptionsToRenderer(vr)
	vr.MoveTo(20, 100)
	vr.LineTo(180, 100)
	vr.Stroke()
	buffer := bytes.NewBuffer([]byte{})
	testutil.AssertNil(t, vr.Save(buffer))
	raw := buffer.String()
	testutil.AssertContains(t, raw, "stroke-linecap:round")
	testutil.AssertContains(t, raw, "stroke-linejoin:round")
	testutil.AssertContains(t, raw, "stroke-miterlimit:6")
}
