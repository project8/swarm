package dripdb

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"time"
)

const (
	ViewFmtString   = "%s/_design/%s/_view/%s"
	RangedFmtString = "%s?startkey=\"%s\"&endkey=\"%s\""
	DripTimeFmt = time.RFC3339
)

type View struct {
	DB     DripDB
	Design string
	Name   string
}

type Viewer interface {
}

type ViewDoc struct {
	Id    string      `json:"id"`
	Key   string      `json:"key"`
	Value interface{} `json:"value"`
}

type GenericViewResult struct {
	TotalRows uint      `json:"total_rows"`
	Offset    uint      `json:"offset"`
	Rows      []ViewDoc `json:"rows"`
}

type KeyRange struct {
	Start, End time.Time
}

func (v *View) URL() string {
	base := v.DB.URL()
	return fmt.Sprintf(ViewFmtString, base, v.Design, v.Name)
}

func toDripFormat(t time.Time)  (s string) {
	s = t.Format(DripTimeFmt)
	return
}

func (v *View) GetDataForRange(r KeyRange, res Viewer) (e error) {
	base := v.URL()
	
	startString := toDripFormat(r.Start)
	endString := toDripFormat(r.End)

	url := fmt.Sprintf(RangedFmtString, base, startString, endString)
	fmt.Println(url)

	http_res, e := http.Get(url)
	if e != nil {
		return
	}

	data, e := ioutil.ReadAll(http_res.Body)
	if e != nil {
		return
	}

	json.Unmarshal(data, res)
	return
}
