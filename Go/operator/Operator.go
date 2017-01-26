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


func getOperatorTag(operator string) (string) {
	return "(<@" + operator + ">)"
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

	botUserName := viper.GetString("username")

	channelName := viper.GetString("channel")

	// check authentication for desired username
	if authErr := authentication.Load(); authErr != nil {
		logging.Log.Criticalf("Error in loading authenticators: %v", authErr)
		os.Exit(1)
	}

	if ! authentication.SlackAvailable(botUserName) {
		logging.Log.Criticalf("Authentication for user <%s> is not available", botUserName)
		os.Exit(1)
	}
	authToken := authentication.SlackToken(botUserName)

	logging.Log.Infof("Slack username: %s", botUserName)
	logging.Log.Infof("Slack token: %s", authToken)

	// get the slack API object
	api := slack.New(authToken)
	if api == nil {
		logging.Log.Critical("Unable to make a new Slack API")
		os.Exit(1)
	}
	logging.Log.Info("Created Slack API")

	// make a map from user ID to name, and a map from user name to ID
	var userIDMap map[string]string
	var userNameMap map[string]string
	userIDMap = make(map[string]string)
	userNameMap = make(map[string]string)
	botUserID := ""
	users, usersErr := api.GetUsers()
	if usersErr != nil {
		logging.Log.Criticalf("Unable to get users: %s", usersErr)
		os.Exit(1)
	} else {
		for _, user := range users {
			userIDMap[user.ID] = user.Name
			userNameMap[user.Name] = user.ID
			if user.Name == botUserName {
				botUserID = user.ID
			}
		}
	}
	if botUserID == "" {
		logging.Log.Criticalf("Could not get user ID for user <%s>", botUserName)
		os.Exit(1)
	}
	logging.Log.Infof("User ID: %s", botUserID)
	botUserTag := "<@" + botUserID + ">"

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

	theOperator := ""
	theOperatorTag := ""

	logging.Log.Info("Connecting to RTM")
	rtm := api.NewRTM()
	go rtm.ManageConnection()
	defer func() {
		logging.Log.Info("Disconnecting from RTM")
		if discErr := rtm.Disconnect(); discErr != nil {
			logging.Log.Error("Error while disconnecting from the Slack RTM")
		}
	}()

	// Setup the commands that are used
	var commandMap map[string]func(string, *slack.MessageEvent)
	commandMap = make(map[string]func(string, *slack.MessageEvent))
	commandMap["!hello"] = func(_ string, msg *slack.MessageEvent) {
		hiMsg := rtm.NewOutgoingMessage("Hi, " + userIDMap[msg.User], msg.Channel)
		rtm.SendMessage(hiMsg)
		return
	}
	commandMap["!help"] = func(_ string, msg *slack.MessageEvent) {
		msgText := "You can either address me with `@operator` or enter a command.\n\n" +
			"If you address me with `@operator` I'll pass a notification on to the current operator.\n\n" +
			"If you enter a command, I can take certain actions:\n" +
			"\t`!hello`: say hi\n" +
			"\t`!help`: display this help message\n" +
			"\t`!startshift`: manually start your shift, replacing the existing operator\n" +
			"\t`!endshift`: remove yourself as the operator\n" +
			"\t`!overrideshift [username (optional)]`: replace the current operator with a manually-specified operator; if no operator is specified, the current operator will be removed"
			logging.Log.Debug("Printing help message")
			helpMsg := rtm.NewOutgoingMessage(msgText, msg.Channel)
			rtm.SendMessage(helpMsg)
	}
	commandMap["!startshift"] = func(_ string, msg *slack.MessageEvent) {
		logging.Log.Info("Shift starting for user " + userIDMap[msg.User])
		theOperator = msg.User
		theOperatorTag = getOperatorTag(theOperator)
		ssMessage := rtm.NewOutgoingMessage("Happy operating, " + userIDMap[theOperator] + "!", msg.Channel)
		rtm.SendMessage(ssMessage)
		return
	}
	commandMap["!endshift"] = func(_ string, msg *slack.MessageEvent) {
		if msg.User != theOperator {
			logging.Log.Debug("Received end-shift command from non-operator")
			usMessage := rtm.NewOutgoingMessage("I can't end your shift, because you're not the operator", msg.Channel)
			rtm.SendMessage(usMessage)
			return
		}
		logging.Log.Info("Shift ended for user " + userIDMap[theOperator])
		//changeOperator <- ""
		theOperator = ""
		theOperatorTag = ""
		usMessage := rtm.NewOutgoingMessage("Your shift is over, thanks!", msg.Channel)
		rtm.SendMessage(usMessage)
		return
	}
	commandMap["!overrideshift"] = func(username string, msg *slack.MessageEvent) {
		if username != "" {
			logging.Log.Info("Shift starting for user " + username)
			newUserID, hasID := userNameMap[username]
			if ! hasID {
				logging.Log.Warningf("Unknown username: %s", username)
				osMessage := rtm.NewOutgoingMessage("I'm sorry, I don't recognize that username", msg.Channel)
				rtm.SendMessage(osMessage)
				return
			}
			theOperator = newUserID
			theOperatorTag = getOperatorTag(theOperator)
			osMessage := rtm.NewOutgoingMessage("Happy operating, " + userIDMap[theOperator] + "!", msg.Channel)
			rtm.SendMessage(osMessage)
		} else {
			theOperator = ""
			theOperatorTag = ""
			osMessage := rtm.NewOutgoingMessage("Operator has been removed", msg.Channel)
			rtm.SendMessage(osMessage)
		}
		return
	}

	arrivedMsg := rtm.NewOutgoingMessage("Have no fear, @operator is here!", channelID)
	rtm.SendMessage(arrivedMsg)

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

			case *slack.MessageEvent:
				//logging.Log.Infof("Message: %v", evData)
				if evData.SubType != "" {
					break
				}

				logging.Log.Debugf("Got a message: %s", evData.Text)

				if evData.Channel != channelID || evData.User == botUserID {
					continue
				}

				logging.Log.Debug("Message is on the right channel")

				if evData.Text[0] == '!' {
					logging.Log.Debug("Received an instruction")
					tokens := strings.SplitN(evData.Text, " ", 2)
					command := strings.ToLower(tokens[0])
					extraText := ""
					logging.Log.Infof("Received command %s", command)
					if len(tokens) > 1 {
						extraText = tokens[1]
						logging.Log.Infof("Extra text in the command: %s", extraText)
					}
					funcToRun, hasCommand := commandMap[command]
					if ! hasCommand {
						logging.Log.Warningf("Received unknown command: %s", command)
						errorMsg := rtm.NewOutgoingMessage("I'm sorry, that's not something I know how to do", evData.Channel)
						rtm.SendMessage(errorMsg)
						continue
					}
					logging.Log.Debugf("Running function for command <%s>", command)
					funcToRun(extraText, evData)
					continue
				}

				if strings.Contains(evData.Text, botUserTag) {
					if theOperator == "" {
						logging.Log.Info("Got operator message, but no operator is assigned")
						notifyMsg := rtm.NewOutgoingMessage("No operator assigned", evData.Channel)
						rtm.SendMessage(notifyMsg)
						continue
					}

					logging.Log.Info("Attempting to notify the operator")
					notifyMsg := rtm.NewOutgoingMessage(theOperatorTag, evData.Channel)
					rtm.SendMessage(notifyMsg)
					continue
				}


			//case *slack.PresenceChangeEvent:
			//	logging.Log.Infof("Presence Change: %v", evData)

			case *slack.LatencyReport:
				logging.Log.Infof("Current latency: %v", evData.Value)
				continue

			case *slack.RTMError:
				logging.Log.Warningf("RTM Error: %s", evData.Error())
				continue

			case *slack.InvalidAuthEvent:
				logging.Log.Error("Invalid credentials")
				break monitorLoop

			default:

				// Ignore other events..
				//logging.Log.Infof("Unexpected: %v", event.Data)
			}
		}
	}

	leavingMsg := rtm.NewOutgoingMessage("Signing off!", channelID)
	rtm.SendMessage(leavingMsg)

	logging.Log.Info("All done!")

	return
}
