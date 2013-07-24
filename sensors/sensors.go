package sensors

type Calibrator interface {
	Calibrate(float64) float64
}

type Point2d struct {
	X, Y float64
}
