package dripdb

type LoggedData struct {
	SensorName   string `json:"sensor_name"`
	Uncalibrated string `json:"uncalibrated_value"`
	Calibrated   string `json:"calibrated_value"`
	Timestamp    string `json:"timestamp_localstring"`
}

type LogViewDoc struct {
	Id    string     `json:"id"`
	Key   string     `json:"key"`
	Value LoggedData `json:"value"`
}

type LogViewResult struct {
	TotalRows uint         `json:"total_rows"`
	Offset    uint         `json:"offset"`
	Rows      []LogViewDoc `json:"rows"`
}
