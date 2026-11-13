package chart

import "github.com/wcharczuk/go-chart/v2/drawing"

// LineCap is the style used for the ends of open subpaths.
// Each dash of a dashed stroke is its own subpath and is capped
// according to this setting as well.
type LineCap string

const (
	// LineCapUnset leaves the cap style up to the renderer, which
	// historically renders as LineCapButt.
	LineCapUnset LineCap = ""
	// LineCapButt ends the stroke exactly at the end of the line.
	LineCapButt LineCap = "butt"
	// LineCapRound ends the stroke with a semicircle of radius half the
	// stroke width centered on the end of the line.
	LineCapRound LineCap = "round"
	// LineCapSquare ends the stroke with a square extending half the
	// stroke width beyond the end of the line.
	LineCapSquare LineCap = "square"
)

// LineJoin is the style used where two segments of a subpath meet.
type LineJoin string

const (
	// LineJoinUnset leaves the join style up to the renderer, which
	// historically renders as LineJoinBevel.
	LineJoinUnset LineJoin = ""
	// LineJoinBevel cuts the corner between segments with a straight chord.
	LineJoinBevel LineJoin = "bevel"
	// LineJoinRound rounds the corner between segments with an arc of
	// radius half the stroke width.
	LineJoinRound LineJoin = "round"
	// LineJoinMiter extends the outer edges of the segments to a point,
	// degrading to LineJoinBevel when the miter length exceeds the
	// configured miter limit.
	LineJoinMiter LineJoin = "miter"
)

// drawingLineCap maps the chart line cap to the drawing package equivalent.
// Unset values map to butt, which matches the historical raster output.
func (c LineCap) drawingLineCap() drawing.LineCap {
	switch c {
	case LineCapRound:
		return drawing.RoundCap
	case LineCapSquare:
		return drawing.SquareCap
	default:
		return drawing.ButtCap
	}
}

// drawingLineJoin maps the chart line join to the drawing package
// equivalent. Unset values map to bevel, which matches the historical
// raster output.
func (j LineJoin) drawingLineJoin() drawing.LineJoin {
	switch j {
	case LineJoinRound:
		return drawing.RoundJoin
	case LineJoinMiter:
		return drawing.MiterJoin
	default:
		return drawing.BevelJoin
	}
}
