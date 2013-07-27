package cernox

import (
	"math"
	s "github.com/project8/swarm/sensors"
)

type Cernox struct {
	CalPts []s.Point2d
}

func (c *Cernox) Calibrate(Ω float64) (K float64) {
	logΩ := math.Log(Ω)
	pt1, pt2 := find_interval(logΩ, c.CalPts)
	slope, icept := linear_fit(pt1, pt2)
	logT := interpolate(slope,icept, logΩ)
	return math.Exp(logT)
}

func interpolate(m, b, x float64) float64 {
	return (m*x + b)
}

func linear_fit(pt1, pt2 s.Point2d) (m, b float64) {
	m = (pt2.Y - pt1.Y)/(pt2.X - pt1.X)
	b = pt2.Y - m*pt2.X
	return
}

func find_interval(pt float64, points []s.Point2d) (s.Point2d,s.Point2d) {
	// Assumes points are sorted from low to high T!
	// If the first point is larger than the first element
	// in points, return the first element.  If not, iterate
	// through until we find the appropriate one.  If we get
	// all the way to the end, return the last element.
	if pt <= points[0].X {
		return points[0], points[1]
	} else {
		for i := 1; i < len(points) - 1; i++ {
			if (pt >= points[i].X && pt < points[i+1].X ) {
				return points[i], points[i+1]
			} 
		}
	}
	return points[len(points)-2], points[len(points)-1]
}
