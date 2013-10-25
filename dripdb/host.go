package dripdb

import (
	"fmt"
)

type DripDBHost struct {
	Host string
	Port uint
}

func (d *DripDBHost) URL() string {
	return fmt.Sprintf("http://%s:%d", d.Host, d.Port)
}
