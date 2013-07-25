package cernox

import (
	"testing"
	"fmt"
	"math"
)

func TestLN2RTRange(t *testing.T) {
	eps := 1.0e-3
	c := Cernox{CalPts: Cernox87821}
	rt := c.Calibrate(56.21)
	rt_err := math.Abs((rt - 276.33)/276.33)
	if rt_err > eps {
		fmt.Printf("RT cal error exceeded tol: %v (%v%%)\n",rt,rt_err)
		t.Fail()
	} else {
		fmt.Printf("RT success (%v, %v=tgt, e=%v)",rt, 276.33,rt_err*100)
	}
}

func TestLHe2LNRange(t *testing.T) {
	eps := 1.0e-3
	c := Cernox{CalPts: Cernox87821}
	rt := c.Calibrate(133.62)
	rt_err := math.Abs((rt - 77.0)/77.0)
	if rt_err > eps {
		fmt.Printf("LN cal error exceeded tol: %v (%v%%)\n",rt,rt_err)
		t.Fail()
	} else {
		fmt.Printf("LN success (%v, %v=tgt, e=%v)",rt, 77.0,rt_err*100)
	}
}

func TestLHeRange(t *testing.T) {
	eps := 1.0e-3
	c := Cernox{CalPts: Cernox87821}
	rt := c.Calibrate(1764.0)
	rt_err := math.Abs((rt-4.2)/4.2)
	if rt_err > eps {
		fmt.Printf("LHe cal error exceeded tol: %v (%v%%)\n",rt,rt_err)
		t.Fail()
	} else {
		fmt.Printf("LHe success (%v, %v=tgt, e=%v)",rt, 4.2,rt_err*100)
	}
}
