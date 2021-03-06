/* This program will look at specified folder on the machine where it is running and get used space.
It will then send an alert to the sensor_value.disks_{machine}_{diskname} queue about these two pieces of information.
It will then go to sleep for a specified amount of time.

Created: Mathieu Guigue, Dec 1 2016
Last update: Walter Pettus, Oct 1 2017
*/
package main

import (
	"flag"
	// "fmt"
	"os"
	"os/user"
	// "path/filepath"
	//"reflect"
	// "strconv"
	"strings"
	"syscall"
	//"sync"
	"time"

	"github.com/kardianos/osext"
	"github.com/spf13/viper"
	// could not install the following go program so I copied the code in the program here
	// "github.com/lunny/diskinfo.go"

	"github.com/project8/dripline-go/dripline"

	"github.com/project8/swarm/Go/authentication"
	"github.com/project8/swarm/Go/logging"
	// "github.com/project8/swarm/Go/utility"
)

type DiskStatus struct {
	Used     uint64  `json:"used"`
	Fraction float64 `json:"fraction"`
}

// disk usage of path/disk
func DiskUsage(path string) (disk DiskStatus) {
	fs := syscall.Statfs_t{}
	err := syscall.Statfs(path, &fs)
	if err != nil {
		return
	}
	disk.Used = (fs.Blocks - fs.Bfree) * uint64(fs.Bsize)
	disk.Fraction = float64(fs.Blocks-fs.Bfree) / float64(fs.Blocks-fs.Bfree+fs.Bavail)
	return
}

const (
	B  = 1
	KB = 1024 * B
	MB = 1024 * KB
	GB = 1024 * MB
)

var MasterSenderInfo dripline.SenderInfo

func fillMasterSenderInfo() (e error) {
	MasterSenderInfo.Package = "diopsid"
	MasterSenderInfo.Exe, e = osext.Executable()
	if e != nil {
		return
	}

	//MasterSenderInfo.Version = gogitver.Tag()
	//MasterSenderInfo.Commit = gogitver.Git()

	MasterSenderInfo.Hostname, e = os.Hostname()
	if e != nil {
		return
	}

	user, userErr := user.Current()
	e = userErr
	if e != nil {
		return
	}
	MasterSenderInfo.Username = user.Username
	return
}

func main() {
	logging.InitializeLogging()

	// user needs help
	var needHelp bool

	// configuration file
	var configFile string

	// set up flag to point at conf, parse arguments and then verify
	flag.BoolVar(&needHelp,
		"help",
		false,
		"Display this dialog")
	flag.StringVar(&configFile,
		"config",
		"",
		"JSON configuration file")
	flag.Parse()

	if needHelp {
		flag.Usage()
		os.Exit(1)
	}

	// defult configuration
	viper.SetDefault("log-level", "INFO")
	viper.SetDefault("broker", "localhost")
	viper.SetDefault("wait-interval", "1m")
	viper.SetDefault("subscribe-queue", "diopsid-queue")
	viper.SetDefault("alerts-queue-base", "sensor_value.disks_machinename_")

	// load config
	if configFile != "" {
		viper.SetConfigFile(configFile)
		if parseErr := viper.ReadInConfig(); parseErr != nil {
			logging.Log.Criticalf("%v", parseErr)
			os.Exit(1)
		}
		logging.Log.Notice("Config file loaded")
	}
	logging.ConfigureLogging(viper.GetString("log-level"))
	logging.Log.Infof("Log level: %v", viper.GetString("log-level"))

	wheretolook := viper.GetStringSlice("where-to-look")
	if len(wheretolook) == 0 {
		logging.Log.Critical("No directories were provided")
		os.Exit(1)
	}
	for i, dir := range wheretolook {
		wheretolook[i] = strings.TrimSuffix(dir, "/")
	}
	logging.Log.Debugf("wheretolook after trimming trailing '/' : %q", wheretolook)

	broker := viper.GetString("broker")
	queueName := viper.GetString("subscribe-queue")
	alertsQueueBase := viper.GetString("alerts-queue-base")
	waitInterval := viper.GetDuration("wait-interval")

	// check authentication for desired username
	if authErr := authentication.Load(); authErr != nil {
		logging.Log.Criticalf("Error in loading authenticators: %v", authErr)
		os.Exit(1)
	}

	if !authentication.AmqpAvailable() {
		logging.Log.Critical("Authentication for AMQP is not available")
		os.Exit(1)
	}

	amqpUser := authentication.AmqpUsername()
	amqpPassword := authentication.AmqpPassword()
	url := "amqp://" + amqpUser + ":" + amqpPassword + "@" + broker

	service := dripline.StartService(url, queueName)
	if service == nil {
		logging.Log.Critical("AMQP service did not start")
		os.Exit(1)
	}
	logging.Log.Info("AMQP service started")

	// add .# to the queue name for the subscription
	// the queue name does not have to be the same as the queue where to send the alerts!
	// it is better to define a proper queueName in the config file to prevent
	// conflict between services subscribing to the same queue (which is not allowed!)
	subscriptionKey := queueName + ".#"
	if subscribeErr := service.SubscribeToAlerts(subscriptionKey); subscribeErr != nil {
		logging.Log.Criticalf("Could not subscribe to alerts at <%v>: %v", subscriptionKey, subscribeErr)
		os.Exit(1)
	}

	if msiErr := fillMasterSenderInfo(); msiErr != nil {
		logging.Log.Criticalf("Could not fill out master sender info: %v", MasterSenderInfo)
		os.Exit(1)
	}

	for {
		for _, dir := range wheretolook {
			diskname := strings.Split(dir, "/")
			alert := dripline.PrepareAlert(alertsQueueBase+diskname[len(diskname)-1], "application/json", MasterSenderInfo)
			disk := DiskUsage(dir)
			var payload map[string]interface{}
			payload = make(map[string]interface{})
			payload["value_raw"] = float64(disk.Used) / float64(GB)
			payload["value_cal"] = disk.Fraction
			alert.Message.Payload = payload

			e := service.SendAlert(alert)
			if e != nil {
				logging.Log.Errorf("Could not send the alert: %v", e)
			}
			logging.Log.Infof("Alert sent: [%s] Used: %d KB, Use Fraction: %.3f", dir, disk.Used/KB, disk.Fraction)
			time.Sleep(2 * time.Second)
		}
		logging.Log.Infof("Sleeping now")
		time.Sleep(waitInterval)
	}
}
