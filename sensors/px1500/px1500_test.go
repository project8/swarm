package px1500

import(
	"fmt"
	"math"
	"testing"
)

func TestByteConv(t *testing.T) {
	d := PX1500{}
	eps := 1.0e-3
	min := d.Calibrate(byte(0))
	err := math.Abs((min - adc_min)/adc_min)
	if err > eps {
		fmt.Printf("minimum cal exceeded tol (%v, %v)\n",min, adc_min)
		t.Fail()
	} else {
		fmt.Printf("min cal success (%v ~ %v +/- %v)\n",min,adc_min,eps)
	}
	max := d.Calibrate(byte(255))
	err = math.Abs((max - adc_max)/adc_max)
	if err > eps {
		fmt.Printf("maximum cal exceeded tol (%v, %v)\n",max, adc_max)
		t.Fail()
	} else {
		fmt.Printf("max cal success (%v ~ %v +/- %v)\n",max,adc_max,eps)
	}

	zero := d.Calibrate(byte(127))
	err = math.Abs(zero)
	if zero > eps {
		fmt.Printf("zero cal exceeded tol (%v)\n",zero)
		t.Fail()
	} else {
		fmt.Printf("zero cal success (%v)\n", zero)
	}
}
