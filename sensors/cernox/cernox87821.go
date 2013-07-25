package cernox

import (
	"math"
	s "github.com/project8/swarm/sensors"
)

var Cernox87821 = []s.Point2d{
	// log(R), log(T)
	s.Point2d{X: math.Log(56.21), Y: math.Log(276.33)},
	s.Point2d{X: math.Log(133.62), Y: math.Log(77.0)},
	s.Point2d{X: math.Log(1764.0), Y: math.Log(4.2)},
}
