package px1500

const (
	adc_min float64 = -0.25
	adc_max float64 = 0.25
	adc_range float64 = adc_max - adc_min
	n_levels int = 1 << 8
)

type PX1500 struct {
}

func (p *PX1500) Calibrate(v interface{}) (f float64) {
	// The PX1500 can calibrate:
	//   ADC counts (bytes)
	//   Power (float64)
	switch v.(type) {
	case byte:
		f = adc_count_to_voltage(v.(byte))
	default:
	}
	return
}

func adc_count_to_voltage(b byte) float64 {
	return adc_min + adc_range*float64(b)/float64(n_levels - 1)
}
