package sensors

type Calibrator interface {
	Calibrate(interface{}) float64
}

type Point2d struct {
	X, Y float64
}
