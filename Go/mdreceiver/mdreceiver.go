package main

import (
	"flag"
	"os"
	"os/user"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"
	//"sync"
	"unsafe"

	"github.com/kardianos/osext"
	"github.com/spf13/viper"
	"github.com/ugorji/go/codec"

	"github.com/project8/dripline/go/dripline"

	"github.com/project8/swarm/Go/authentication"
	"github.com/project8/swarm/Go/logging"
	"github.com/project8/swarm/Go/utility"
)

var MasterSenderInfo dripline.SenderInfo
func fillMasterSenderInfo() (e error) {
	MasterSenderInfo.Package = "hornet"
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
	viper.SetDefault("queue", "metadata")

	// load config
	viper.SetConfigFile(configFile)
	if parseErr := viper.ReadInConfig(); parseErr != nil {
		logging.Log.Critical("%v", parseErr)
		os.Exit(1)
	}
	logging.Log.Notice("Config file loaded")
	logging.ConfigureLogging(viper.GetString("log-level"))
	logging.Log.Info("Log level: %v", viper.GetString("log-level"))

	broker := viper.GetString("broker")
	queueName := viper.GetString("queue")

	// check authentication for desired username
	if authErr := authentication.Load(); authErr != nil {
		logging.Log.Critical("Error in loading authenticators: %v", authErr)
		os.Exit(1)
	}

	if ! authentication.AmqpAvailable() {
		logging.Log.Critical("Authentication for AMQP is not available")
		os.Exit(1)
	}

	amqpUser := authentication.AmqpUsername()
	amqpPassword := authentication.AmqpPassword()

	url := "amqp://" + amqpUser + ":" + amqpPassword + "@" + broker

	service := dripline.StartService(url, queueName)
	if (service == nil) {
		logging.Log.Critical("AMQP service did not start")
		os.Exit(1)
	}
	logging.Log.Info("AMQP service started")

	if subscribeErr := service.SubscribeToRequests(queueName); subscribeErr != nil {
		logging.Log.Critical("Could not subscribe to requests: %v", subscribeErr)
		os.Exit(1)
	}

	if msiErr := fillMasterSenderInfo(); msiErr != nil {
		logging.Log.Critical("Could not fill out master sender info: %v", MasterSenderInfo)
		os.Exit(1)
	}




	//context := build.defaultContext()

	//os.Exit(2)



receiverLoop:
	for {
		select {
		case request, chanOpen := <-service.Receiver.RequestChan:
			if ! chanOpen {
				logging.Log.Error("Incoming request channel is closed")
				break receiverLoop
			}
			
			logging.Log.Debug("Received request")
			switch request.MsgOp {
			case dripline.MOCommand:
				var instruction string
				if request.Message.Target != queueName {
					instruction = strings.TrimPrefix(request.Message.Target, queueName + ".")
				}
				logging.Log.Debug("Command instruction: %s", instruction)
				switch instruction {
				//case "write_metadata":
				case "":
					logging.Log.Debug("Received \"write_metadata\" instruction")
					logging.Log.Warning("type: %v", reflect.TypeOf(request.Message.Payload))
					logging.Log.Warning("try printing the payload? \n%v", request.Message.Payload)
					payloadAsMap, okPAM := request.Message.Payload.(map[interface{}]interface{})
					if ! okPAM {
						if sendErr := PrepareAndSendReply(service, request, dripline.RCErrDripPayload, "Unable to convert payload to map; aborting message", MasterSenderInfo); sendErr != nil {
							break receiverLoop
						}
						continue receiverLoop
					}
					logging.Log.Warning("chips? %v", payloadAsMap["chips"])
					filenameIfc, hasFN := payloadAsMap["filename"]
					if ! hasFN {
						if sendErr := PrepareAndSendReply(service, request, dripline.RCErrDripPayload, "No filename present in message; aborting", MasterSenderInfo); sendErr != nil {
							break receiverLoop
						}
						continue receiverLoop
					}
					thePath, okFP := utility.TryConvertToString(filenameIfc)
					if okFP != nil {
						if sendErr := PrepareAndSendReply(service, request, dripline.RCErrDripPayload, "Unable to convert filename to string; aborting message", MasterSenderInfo); sendErr != nil {
							break receiverLoop
						}
						continue receiverLoop
					}
					logging.Log.Debug("Filename to write: %s", thePath)

					dir, _ := filepath.Split(thePath)
					if mkdirErr := os.MkdirAll(dir, os.ModeDir); mkdirErr != nil {
						if sendErr := PrepareAndSendReply(service, request, dripline.RCErrHW, "Unable to create directory; aborting", MasterSenderInfo); sendErr != nil {
							break receiverLoop
						}
						continue receiverLoop
					}

					metadataIfc, hasMetadata := payloadAsMap["metadata"]
					if ! hasMetadata {
						if sendErr := PrepareAndSendReply(service, request, dripline.RCErrDripPayload, "No metadata present in message; aborting", MasterSenderInfo); sendErr != nil {
							break receiverLoop
						}
						continue receiverLoop
					}

					encoded := make([]byte, 0, unsafe.Sizeof(metadataIfc))
					handle := new(codec.JsonHandle)
					encoder := codec.NewEncoderBytes(&(encoded), handle)
					jsonErr := encoder.Encode(metadataIfc)
					if jsonErr != nil {
						if sendErr := PrepareAndSendReply(service, request, dripline.RCErrDripPayload, "Unable to convert metadata to JSON", MasterSenderInfo); sendErr != nil {
							break receiverLoop
						}
						continue receiverLoop
					}

					theFile, fileErr := os.Create(thePath)
					if fileErr != nil {
						if sendErr := PrepareAndSendReply(service, request, dripline.RCErrHW, "Unable to create file for the metadata", MasterSenderInfo); sendErr != nil {
							break receiverLoop
						}
						continue receiverLoop
					}

					_, writeErr := theFile.Write(encoded)
					if writeErr != nil {
						if sendErr := PrepareAndSendReply(service, request, dripline.RCErrHW, "Unable to write metadata to file", MasterSenderInfo); sendErr != nil {
							break receiverLoop
						}
						continue receiverLoop
					}

					closeErr := theFile.Close()
					if closeErr != nil {
						if sendErr := PrepareAndSendReply(service, request, dripline.RCErrHW, "Unable to close the metadata file", MasterSenderInfo); sendErr != nil {
							break receiverLoop
						}
						continue receiverLoop
					}

					if sendErr := PrepareAndSendReply(service, request, dripline.RCSuccess, "Metadata file written", MasterSenderInfo); sendErr != nil {
						break receiverLoop
					}

				default:
					message := "Incoming request operation instruction not handled: " + instruction
					if sendErr := PrepareAndSendReply(service, request, dripline.RCErrDripMethod, message, MasterSenderInfo); sendErr != nil {
						break receiverLoop
					}
					continue receiverLoop
				}
			default:
				message := "Incoming request operation type not handled: " + strconv.FormatUint(uint64(request.MsgOp), 10)
				if sendErr := PrepareAndSendReply(service, request, dripline.RCErrDripMethod, message, MasterSenderInfo); sendErr != nil {
					break receiverLoop
				}
				continue receiverLoop
			}
		}
	}

	logging.Log.Info("MdReceiver is finished")
}

func PrepareAndSendReply(service *dripline.AmqpService, request dripline.Request, retCode dripline.MsgCodeT, returnMessage string, senderInfo dripline.SenderInfo) (e error) {
	e = nil
	reply := dripline.PrepareReplyToRequest(request, retCode, returnMessage, senderInfo)
	e = service.SendReply(reply);
	if e != nil {
		logging.Log.Error("Could not send the reply: %v", e)
	}
	return
}

