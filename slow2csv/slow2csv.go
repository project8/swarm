/*
	slow2csv grabs data from the dripline_logged_data database
	and dumps it to a raw text file.
*/
package main

import (
	"flag"
	"fmt"
	"github.com/project8/swarm/dripdb"
	"time"
	"strings"
)

const (
	/*
		The header format is
		timestamp, uncalibrated (units), calibrated (units)
	*/
	HeaderFmtString = "timestamp, unix_ts, uncalibrated (%s), calibrated (%s)\n"
	LineFmtString   = "%s, %s, %s, %s\n"
)

type Config struct {
	Host string
	Port uint
}

func StripUnits(s string) (value, unit string) {
	res := strings.Fields(strings.Trim(s, "\r\n"))
	if len(res) == 1 {
		value, unit = res[0], ""
	} else {
		value, unit = strings.TrimSpace(res[0]), strings.TrimSpace(res[1])
	}
	return
}

func main() {
	var dbhost = flag.String("host",
		"myrna.phys.washington.edu",
		"dripline database host")

	var dbport = flag.Uint("port",
		5984,
		"dripline database port")

	var dbname = flag.String("database_name",
		"dripline_logged_data",
		"name of database where logged data is stored")

	// TODO this should be current time - 2 hrs
	var from = flag.String("from",
		time.Now().UTC().Add(-2.0*time.Hour).Format(dripdb.TimeFormat),
		"start time of data you want")

	// TODO this should be now
	var to = flag.String("to",
		time.Now().UTC().Format(dripdb.TimeFormat),
		"stop time of data you want")

	var channel = flag.String("channel",
		"terminator_temp",
		"name of the channel for which data is desired")

	var get_names = flag.Bool("get_names", false, "only list possible channels")

	flag.Parse()

	// parse time strings to Time types
	t0, t0_err := time.Parse(time.RFC3339,*from)
	if t0_err != nil {
		fmt.Println("ERR: from time must be specified in RFC3339 format.")
		return
	} 

	t1, t1_err := time.Parse(time.RFC3339, *to)
	if t1_err != nil {
		fmt.Println("ERR: to time must be specified in RFC3339 format.")
		return
	}

	host := dripdb.DripDBHost{Host: *dbhost, Port: *dbport}
	db := dripdb.DripDB{Host: host, Name: *dbname}
	view := dripdb.View{DB: db,
		Design: "log_access",
		Name:   "all_logged_data"}

	keys := dripdb.KeyRange{Start: t0, End: t1}

	var v dripdb.LogViewResult

	err := view.GetDataForRange(keys, &v)
	if err != nil {
		panic("couldn't unmarshal retrieved data!")
	}

	if *get_names {

		/*
			Only report each possible channel once.
		*/

		name_map := make(map[string]bool)

		for _, row := range v.Rows {
			_, exists := name_map[row.Value.SensorName]
			if exists == false {
				fmt.Println(row.Value.SensorName)
				name_map[row.Value.SensorName] = true
			}
		}

	} else {

		/*
			Use the first row to determine units and print the header.
		*/
		var discovered bool = false

		for _, row := range v.Rows {
			v := row.Value
			if row.Value.SensorName == *channel {
				if discovered == false {
					_, cal_units := StripUnits(v.Calibrated)
					_, uncal_units := StripUnits(v.Uncalibrated)
					if cal_units == "" {
						cal_units = "?"
					}
					if uncal_units == "" {
						uncal_units = "?"
					}
					fmt.Printf(HeaderFmtString, uncal_units, cal_units)
					discovered = true
				}

				c, _ := StripUnits(v.Calibrated)
				uc, _ := StripUnits(v.Uncalibrated)
				t, t_err := time.Parse(dripdb.TimeFormat, v.Timestamp)
				var unix_timestring string
				if t_err != nil {
					unix_timestring = ""
				} else {
					unix_timestring = fmt.Sprintf("%d", t.Unix())
				}
				fmt.Printf(LineFmtString,
					v.Timestamp,
					unix_timestring,
					uc,
					c)
			}
		}

	}

}
