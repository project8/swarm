package dripdb

const (
	TimeFormat = "2006-01-02 15:04:05"
)

type URLer interface {
	URL() string
}