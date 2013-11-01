package dripdb

import (
	"time"
)

const (
	TimeFormat = time.RFC3339
)

type URLer interface {
	URL() string
}
