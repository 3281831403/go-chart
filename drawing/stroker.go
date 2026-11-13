// Copyright 2010 The draw2d Authors. All rights reserved.
// created: 13/12/2010 by Laurent Le Goff

package drawing

import "math"

// DefaultMiterLimit is the miter limit used when none is set.
// It matches the SVG default for stroke-miterlimit.
const DefaultMiterLimit = 4.0

// roundArcTolerance is the maximum allowed distance (in pixels) between a
// round cap or join and its chord approximation.
const roundArcTolerance = 0.25

// NewLineStroker creates a new line stroker.
func NewLineStroker(c LineCap, j LineJoin, flattener Flattener) *LineStroker {
	l := new(LineStroker)
	l.Flattener = flattener
	l.HalfLineWidth = 0.5
	l.Cap = c
	l.Join = j
	l.MiterLimit = DefaultMiterLimit
	return l
}

// LineStroker draws the stroke portion of a line.
type LineStroker struct {
	Flattener     Flattener
	HalfLineWidth float64
	Cap           LineCap
	Join          LineJoin
	// MiterLimit is the maximum allowed ratio of the miter length (the
	// distance from a join vertex to the miter tip) to half the line width.
	// When the ratio exceeds the limit a miter join degrades to a bevel
	// join. Values <= 0 are interpreted as DefaultMiterLimit.
	MiterLimit float64

	vertices []float64
	rewind   []float64

	x, y   float64 // current point
	nx, ny float64 // current normal, scaled by HalfLineWidth
	dx, dy float64 // current segment unit direction

	sx, sy     float64 // subpath start point
	snx, sny   float64 // first segment normal
	sdx, sdy   float64 // first segment unit direction
	hasSegment bool
	closed     bool
}

// MoveTo implements the path builder interface.
func (l *LineStroker) MoveTo(x, y float64) {
	l.sx, l.sy = x, y
	l.x, l.y = x, y
	l.hasSegment = false
	l.closed = false
}

// LineTo implements the path builder interface.
func (l *LineStroker) LineTo(x, y float64) {
	l.line(l.x, l.y, x, y)
}

// LineJoin implements the path builder interface.
func (l *LineStroker) LineJoin() {}

func (l *LineStroker) line(x1, y1, x2, y2 float64) {
	dx := (x2 - x1)
	dy := (y2 - y1)
	d := vectorDistance(dx, dy)
	if d == 0 {
		return
	}
	ndx, ndy := dx/d, dy/d
	nx := ndy * l.HalfLineWidth
	ny := -(ndx * l.HalfLineWidth)
	if l.hasSegment {
		l.join(x1, y1, ndx, ndy, nx, ny)
	} else {
		l.snx, l.sny = nx, ny
		l.sdx, l.sdy = ndx, ndy
		l.hasSegment = true
	}
	l.appendVertex(x1+nx, y1+ny, x2+nx, y2+ny, x1-nx, y1-ny, x2-nx, y2-ny)
	l.x, l.y, l.nx, l.ny = x2, y2, nx, ny
	l.dx, l.dy = ndx, ndy
}

// join emits the join geometry at vertex (vx, vy) between the previous
// segment (tracked on the stroker) and a new segment with unit direction
// (ndx, ndy) and scaled normal (nnx, nny).
func (l *LineStroker) join(vx, vy, ndx, ndy, nnx, nny float64) {
	cross := l.dx*ndy - l.dy*ndx
	if cross == 0 {
		// colinear segments need no join
		return
	}
	// cross > 0 means the forward (vertices) side is the outer side of the
	// turn, cross < 0 means the rewind side is.
	side := 1.0
	if cross < 0 {
		side = -1.0
	}
	// outer offsets, each of length HalfLineWidth
	o1x, o1y := side*l.nx, side*l.ny
	o2x, o2y := side*nnx, side*nny
	switch l.Join {
	case MiterJoin:
		hw := l.HalfLineWidth
		cosTheta := (o1x*o2x + o1y*o2y) / (hw * hw)
		if cosTheta <= -1 {
			// 180 degree reversal, no sane miter
			return
		}
		// ratio of the miter length to half the line width
		ratio := math.Sqrt(2 / (1 + cosTheta))
		if ratio > l.miterLimit() {
			// over the limit: degrade to a bevel join, which needs
			// no extra vertices
			return
		}
		l.appendOuter(side, vx+(o1x+o2x)/(1+cosTheta), vy+(o1y+o2y)/(1+cosTheta))
	case RoundJoin:
		l.arc(side, vx, vy, o1x, o1y, o2x, o2y)
	default:
		// BevelJoin: the chord between the two outer offsets is already
		// part of the outline, nothing to add.
	}
}

func (l *LineStroker) miterLimit() float64 {
	if l.MiterLimit <= 0 {
		return DefaultMiterLimit
	}
	return l.MiterLimit
}

// appendOuter appends a point to the outer side of the outline; side > 0 is
// the forward (vertices) side, side < 0 is the rewind side.
func (l *LineStroker) appendOuter(side, x, y float64) {
	if side > 0 {
		l.vertices = append(l.vertices, x, y)
	} else {
		l.rewind = append(l.rewind, x, y)
	}
}

// arc emits a chord approximated arc of radius HalfLineWidth centered on
// (cx, cy) from offset (o1x, o1y) to offset (o2x, o2y), taking the short
// way around. The arc endpoints are excluded; they are already part of the
// outline.
func (l *LineStroker) arc(side, cx, cy, o1x, o1y, o2x, o2y float64) {
	a1 := math.Atan2(o1y, o1x)
	a2 := math.Atan2(o2y, o2x)
	delta := a2 - a1
	for delta <= -math.Pi {
		delta += 2 * math.Pi
	}
	for delta > math.Pi {
		delta -= 2 * math.Pi
	}
	segments := l.arcSegments(delta)
	for i := 1; i < segments; i++ {
		angle := a1 + delta*float64(i)/float64(segments)
		l.appendOuter(side,
			cx+math.Cos(angle)*l.HalfLineWidth,
			cy+math.Sin(angle)*l.HalfLineWidth)
	}
}

// arcSegments returns the number of chords used to approximate an arc
// sweeping delta radians, keeping the chord error under roundArcTolerance.
func (l *LineStroker) arcSegments(delta float64) int {
	step := math.Pi
	if hw := l.HalfLineWidth; hw > roundArcTolerance {
		step = 2 * math.Acos(1-roundArcTolerance/hw)
	}
	segments := int(math.Ceil(math.Abs(delta) / step))
	if segments < 1 {
		segments = 1
	}
	return segments
}

// Close implements the path builder interface.
func (l *LineStroker) Close() {
	if !l.hasSegment || l.closed {
		return
	}
	// join the last segment back into the first one at the subpath start
	l.join(l.sx, l.sy, l.sdx, l.sdy, l.snx, l.sny)
	l.appendVertex(l.vertices[0], l.vertices[1], l.rewind[0], l.rewind[1])
	l.closed = true
}

// End implements the path builder interface.
func (l *LineStroker) End() {
	if len(l.vertices) > 1 {
		capped := !l.closed
		if capped && l.Cap == SquareCap {
			l.applySquareCaps()
		}
		l.Flattener.MoveTo(l.vertices[0], l.vertices[1])
		for i, j := 2, 3; j < len(l.vertices); i, j = i+2, j+2 {
			l.Flattener.LineTo(l.vertices[i], l.vertices[j])
		}
		if capped && l.Cap == RoundCap {
			l.roundCap(l.x, l.y, l.nx, l.ny)
		}
		for i, j := len(l.rewind)-2, len(l.rewind)-1; j > 0; i, j = i-2, j-2 {
			l.Flattener.LineTo(l.rewind[i], l.rewind[j])
		}
		if capped && l.Cap == RoundCap {
			l.roundCap(l.sx, l.sy, -l.snx, -l.sny)
		}
		l.Flattener.LineTo(l.vertices[0], l.vertices[1])
	}
	l.Flattener.End()
	// reinit vertices
	l.vertices = l.vertices[0:0]
	l.rewind = l.rewind[0:0]
	l.x, l.y, l.nx, l.ny = 0, 0, 0, 0
	l.dx, l.dy = 0, 0
	l.sx, l.sy, l.snx, l.sny = 0, 0, 0, 0
	l.sdx, l.sdy = 0, 0
	l.hasSegment = false
	l.closed = false
}

// applySquareCaps extends the first and last outline points of the current
// subpath by HalfLineWidth along the segment directions.
func (l *LineStroker) applySquareCaps() {
	hw := l.HalfLineWidth
	l.vertices[0] -= l.sdx * hw
	l.vertices[1] -= l.sdy * hw
	l.rewind[0] -= l.sdx * hw
	l.rewind[1] -= l.sdy * hw
	vl := len(l.vertices) - 2
	l.vertices[vl] += l.dx * hw
	l.vertices[vl+1] += l.dy * hw
	rl := len(l.rewind) - 2
	l.rewind[rl] += l.dx * hw
	l.rewind[rl+1] += l.dy * hw
}

// roundCap emits a semicircular cap centered on (cx, cy) from offset
// (nx, ny) to its opposite, bulging outward. Rotating the normal by +90
// degrees yields the segment direction, so sweeping the normal angle by pi
// passes through the outward direction.
func (l *LineStroker) roundCap(cx, cy, nx, ny float64) {
	a1 := math.Atan2(ny, nx)
	segments := l.arcSegments(math.Pi)
	for i := 1; i < segments; i++ {
		angle := a1 + math.Pi*float64(i)/float64(segments)
		l.Flattener.LineTo(
			cx+math.Cos(angle)*l.HalfLineWidth,
			cy+math.Sin(angle)*l.HalfLineWidth)
	}
}

func (l *LineStroker) appendVertex(vertices ...float64) {
	s := len(vertices) / 2
	l.vertices = append(l.vertices, vertices[:s]...)
	l.rewind = append(l.rewind, vertices[s:]...)
}
