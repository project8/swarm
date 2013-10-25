package dripdb

import (
	"fmt"
)

const (
	DBFmtString = "%s/%s"
)

type DripDB struct {
	Host DripDBHost
	Name string
}

func (d *DripDB) URL() string {
	base := d.Host.URL()
	return fmt.Sprintf(DBFmtString, base, d.Name)
}
