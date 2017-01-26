package main

import (
	"flag"
	"os"
	"strings"

	"github.com/nlopes/slack"
	"github.com/spf13/viper"

	"github.com/project8/swarm/Go/authentication"
	"github.com/project8/swarm/Go/logging"

)

var monitorStarted bool = false

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
		"display this dialog")
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
	viper.SetDefault("username", "project8")
	viper.SetDefault("log-level", "INFO")
	viper.SetDefault("channel", "test_op")

	// load config
	viper.SetConfigFile(configFile)
	if parseErr := viper.ReadInConfig(); parseErr != nil {
		logging.Log.Criticalf("%v", parseErr)
		os.Exit(1)
	}
	logging.Log.Notice("Config file loaded")
	logging.ConfigureLogging(viper.GetString("log-level"))
	logging.Log.Infof("Log level: %v", viper.GetString("log-level"))

	userName := viper.GetString("username")

	channelName := viper.GetString("channel")

	// check authentication for desired username
	if authErr := authentication.Load(); authErr != nil {
		logging.Log.Criticalf("Error in loading authenticators: %v", authErr)
		os.Exit(1)
	}

	if ! authentication.SlackAvailable(userName) {
		logging.Log.Criticalf("Authentication for user <%s> is not available", userName)
		os.Exit(1)
	}
	authToken := authentication.SlackToken(userName)

	logging.Log.Infof("Slack username: %s", userName)
	logging.Log.Infof("Slack token: %s", authToken)

	// get the slack API object
	api := slack.New(authToken)
	if api == nil {
		logging.Log.Critical("Unable to make a new Slack API")
		os.Exit(1)
	}
	logging.Log.Info("Created Slack API")
	// get list of users and then the user ID
	userID := ""
	users, usersErr := api.GetUsers()
	if usersErr != nil {
		logging.Log.Criticalf("Unable to get users: %s", usersErr)
		os.Exit(1)
	} else {
usernameLoop:
		for _, user := range users {
			if user.Name == userName {
				userID = user.ID
				break usernameLoop
			}
		}
	}
	if userID == "" {
		logging.Log.Criticalf("Could not get user ID for user <%s>", userName)
		os.Exit(1)
	}
	logging.Log.Infof("User ID: %s", userID)
	userTag := "<@" + userID + ">"

	// get map of channel IDs
	channelID := ""
	channels, chanErr := api.GetChannels(true)
	if chanErr != nil {
		logging.Log.Criticalf("Unable to get channels: %s", chanErr)
		os.Exit(1)
	} else {
		for _, aChan := range channels {
			if aChan.Name == channelName {
				channelID = aChan.ID
				logging.Log.Infof("Found ID for channel %s: %v", channelName, channelID)
			}
		}
	}
	if channelID == "" {
		logging.Log.Criticalf("Did not find channel ID for channel %s", channelName)
		os.Exit(1)
	}

	theOperator := "nsoblath"
	theOperatorTag := "(<@" + theOperator + ">)"

	logging.Log.Info("Connecting to RTM")
	rtm := api.NewRTM()
	go rtm.ManageConnection()
	defer func() {
		logging.Log.Info("Disconnecting from RTM")
		if discErr := rtm.Disconnect(); discErr != nil {
			logging.Log.Error("Error while disconnecting from the Slack RTM")
		}
	}()

	logging.Log.Info("Waiting for events")
monitorLoop:
	for {
		select {
		case event, chanOpen := <-rtm.IncomingEvents:
			if ! chanOpen {
				logging.Log.Warning("Incoming events channel is closed")
				break monitorLoop
			}
			switch evData := event.Data.(type) {
			case *slack.HelloEvent:
				logging.Log.Info("Slack says \"hello\"")

			case *slack.ConnectedEvent:
				//logging.Log.Info("Infos:", evData.Info)
				logging.Log.Info("Connected to Slack")
				logging.Log.Infof("Connection counter: %v", evData.ConnectionCount)
				// Replace #general with your Channel ID
				//rtm.SendMessage(rtm.NewOutgoingMessage("Hello world", "#general"))

			case *slack.MessageEvent:
				//logging.Log.Infof("Message: %v", evData)
				if evData.SubType != "" {
					break
				}

				logging.Log.Infof("Got a message: %s", evData.Text)

				if evData.Channel != channelID {
					continue
				}

				logging.Log.Info("Message is on the right channel")

				if strings.Contains(evData.Text, userTag) {
					logging.Log.Info("Attempting to notify the operator")
					notifyMsg := rtm.NewOutgoingMessage(theOperatorTag, channelID)
					rtm.SendMessage(notifyMsg)
				}


			//case *slack.PresenceChangeEvent:
			//	logging.Log.Infof("Presence Change: %v", evData)

			case *slack.LatencyReport:
				logging.Log.Infof("Current latency: %v", evData.Value)

			case *slack.RTMError:
				logging.Log.Warningf("RTM Error: %s", evData.Error())

			case *slack.InvalidAuthEvent:
				logging.Log.Error("Invalid credentials")
				break monitorLoop

			default:

				// Ignore other events..
				//logging.Log.Infof("Unexpected: %v", event.Data)
			}
		}
	}

	logging.Log.Info("All done!")

	return
}
