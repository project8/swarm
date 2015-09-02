package main

import (
	"flag"
	"os"
	"sync"

	"github.com/nlopes/slack"
	"github.com/spf13/viper"

	"github.com/project8/swarm/Go/authentication"
	"github.com/project8/swarm/Go/logging"
	"github.com/project8/swarm/Go/utility"
)

var monitorStarted bool = false

type eventRecipient struct {
	channelID string
	eventChan chan *slack.MessageEvent
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

	// load config
	viper.SetConfigFile(configFile)
	if parseErr := viper.ReadInConfig(); parseErr != nil {
		logging.Log.Critical("%v", parseErr)
		os.Exit(1)
	}
	logging.Log.Notice("Config file loaded")
	logging.ConfigureLogging(viper.GetString("log-level"))
	logging.Log.Info("Log level: %v", viper.GetString("log-level"))

	userName := viper.GetString("username")

	if ! viper.IsSet("channels") {
		logging.Log.Critical("No channel configuration found")
		os.Exit(1)
	}

	// check authentication for desired username
	if authErr := authentication.Load(); authErr != nil {
		logging.Log.Critical("Error in loading authenticators: %v", authErr)
		os.Exit(1)
	}

	if ! authentication.SlackAvailable(userName) {
		logging.Log.Critical("Authentication for user <%s> is not available", userName)
		os.Exit(1)
	}
	authToken := authentication.SlackToken(userName)

	logging.Log.Info("Slack username: %s", userName)
	logging.Log.Info("Slack token: %s", authToken)

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
		logging.Log.Critical("Unable to get users: %s", usersErr)
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
		logging.Log.Critical("Could not get user ID for user <%s>", userName)
		os.Exit(1)
	}
	logging.Log.Info("User ID: %s", userID)

	// get map of channel IDs
	var allChannelMap map[string]string
	channels, chanErr := api.GetChannels(true)
	if chanErr != nil {
		logging.Log.Critical("Unable to get channels: %s", chanErr)
		os.Exit(1)
	} else {
		allChannelMap = make(map[string]string, len(channels))
		for _, aChan := range channels {
			allChannelMap[aChan.Name] = aChan.ID
			logging.Log.Debug("Channel %s has ID %s", aChan.Name, aChan.ID)
		}
	}

	var threadWait sync.WaitGroup
	recipientChan := make(chan eventRecipient, 10)

	// loop over channels
	channelsRaw := viper.GetStringMap("channels")
	for channelName, _ := range channelsRaw {
		// get channel ID
		if channelID, channelExists := allChannelMap[channelName]; channelExists == true {
			logging.Log.Info("(%s) Found request for channel %s", channelID, channelName)
			//channelInfo := channelInfoRaw.(map[string](interface{}))

			channelConfigName := "channels." + channelName

			sizeLimitCN := channelConfigName + ".size-limit"
			sizeLimit := -1
			monitorSize := false
			// size will not be monitored if size-limit is not set in the configuration
			if viper.IsSet(sizeLimitCN) {
				sizeLimit = viper.GetInt(sizeLimitCN)
				monitorSize = viper.GetBool(channelConfigName + ".monitor-size")
			}

			if sizeLimit < 0 {
				logging.Log.Error("(%s) Invalid size limit", channelID)
				continue
			}

			doLogging := false // future feature

			msgQueue := utility.NewQueue()

			var buildHistLock sync.Mutex
			histCond := sync.NewCond(&buildHistLock)

			if monitorSize || doLogging {
				if ! monitorStarted {
					logging.Log.Notice("Launching monitorSlack")
					threadWait.Add(1)
					go monitorSlack(api, &threadWait, recipientChan)
					monitorStarted = true
				}

				recipient := eventRecipient{
					channelID: channelID,
					eventChan: make(chan *slack.MessageEvent, 100),
				}
				recipientChan <- recipient

				logging.Log.Notice("(%s) Launching monitorChannel", channelID)
				threadWait.Add(1)
				go monitorChannel(channelID, api, recipient.eventChan, msgQueue, monitorSize, doLogging, histCond, &threadWait)
	
				// If we're monitoring the channel, then we increase the size limit for cleanHistory by 1.
				// This is because the latest message will be passed to the monitor as the first message received.
				// So the monitor will then remove an old message; so we leave one extra old message to be removed
				// once the monitoring begins.
				sizeLimit++
			}

			logging.Log.Notice("(%s) Launching cleanHistory", channelID)
			threadWait.Add(1)
			go cleanHistory(channelID, api, msgQueue, sizeLimit, histCond, &threadWait)

		} else {
			logging.Log.Warning("Channel <%s> does not exist", channelName)
		}
	}

	logging.Log.Notice("Waiting for threads to finish")
	threadWait.Wait()
	logging.Log.Notice("Threads complete")

	return
}


func cleanHistory(channelID string, api *slack.Client, msgQueue *utility.Queue, histSize int, histCond *sync.Cond, threadWait *sync.WaitGroup) {
	defer threadWait.Done()
	defer histCond.Signal()

	histCond.L.Lock()
	defer histCond.L.Unlock()

	logging.Log.Info("(%s) Starting cleanHistory", channelID)
	histParams := slack.NewHistoryParameters()
	histParams.Inclusive = true

	histCountMax := 1000

	// build history with histSize messages
	logging.Log.Info("(%s) Building history with %v messages", channelID, histSize)
	var history *slack.History
	var histErr error
	nRemaining := histSize
	for nRemaining > 0 {
		if nRemaining > histCountMax {
			histParams.Count = histCountMax
		} else {
			histParams.Count = nRemaining
		}

		history, histErr = api.GetChannelHistory(channelID, histParams)
		if histErr != nil {
			logging.Log.Error("(%s) Unable to get the channel history: %v", channelID, histErr)
			return
		}

		iLastMsg := len(history.Messages) - 1
		//logging.Log.Debug("0: %v, %v: %v", history.Messages[0].Timestamp, iLastMsg, history.Messages[iLastMsg].Timestamp)

		logging.Log.Debug("(%s) In skip loop; obtained history with %v messages", channelID, len(history.Messages))

		for iMsg := iLastMsg; iMsg >= 0; iMsg-- {
			msgQueue.Push(history.Messages[iMsg].Timestamp)
			//logging.Log.Debug("(%s) Pushing to queue: %s", channelID, history.Messages[iMsg].Timestamp)
		}

		if ! history.HasMore {
			return
		}
		histParams.Latest = history.Messages[iLastMsg].Timestamp
		histParams.Inclusive = false
		nRemaining -= histCountMax
	}

	histParams.Count = histCountMax
	nDeleted := 0
	for history.HasMore == true {
		history, histErr = api.GetChannelHistory(channelID, histParams)
		if histErr != nil {
			logging.Log.Error("(%s) Unable to get the channel history: %v", channelID, histErr)
			return
		}

		logging.Log.Debug("(%s) Deleting %v items (latest: %v)", channelID, len(history.Messages), history.Latest)

		for _/*iMsg*/, message := range history.Messages {
			//logging.Log.Debug("(%s) Deleting: %s", channelID, message.Timestamp)
			_, _, /*respChan, respTS,*/ respErr := api.DeleteMessage(channelID, message.Timestamp)
			if respErr != nil {
				logging.Log.Warning("(%s) Unable to delete message: %v", respErr)
			}
			//logging.Log.Debug("(%s) Deletion response: %s, %s, %v", channelID, respChan, respTS, respErr)
			nDeleted++
		}
		histParams.Latest = history.Messages[len(history.Messages)-1].Timestamp
	}
	logging.Log.Notice("(%s) Deleted %v messages", channelID, nDeleted)

	return
}


func monitorSlack(api *slack.Client, threadWait *sync.WaitGroup, recipientChan chan eventRecipient) {
	defer threadWait.Done()
	defer logging.Log.Notice("Finished monitoring Slack")

	eventRecipients := make(map[string]chan *slack.MessageEvent)

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
		case recipient := <-recipientChan:
			eventRecipients[recipient.channelID] = recipient.eventChan
			logging.Log.Debug("Added event recipient for %v", recipient.channelID)

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
				logging.Log.Info("Connection counter: %v", evData.ConnectionCount)
				// Replace #general with your Channel ID
				//rtm.SendMessage(rtm.NewOutgoingMessage("Hello world", "#general"))

			case *slack.MessageEvent:
				//logging.Log.Info("Message: %v", evData)
				if evData.SubType != "" {
					break
				}
				if eventChan, hasEventChan := eventRecipients[evData.Channel]; hasEventChan {
					eventChan <- evData
				}

			//case *slack.PresenceChangeEvent:
			//	logging.Log.Info("Presence Change: %v", evData)

			case *slack.LatencyReport:
				logging.Log.Info("Current latency: %v", evData.Value)

			case *slack.RTMError:
				logging.Log.Warning("RTM Error: %s", evData.Error())

			case *slack.InvalidAuthEvent:
				logging.Log.Error("Invalid credentials")
				break monitorLoop

			default:

				// Ignore other events..
				//logging.Log.Info("Unexpected: %v", event.Data)
			}
		}
	}

	return
}

func monitorChannel(channelID string, api *slack.Client, messageChan <-chan *slack.MessageEvent, msgQueue *utility.Queue, monitorSize, doLogging bool, histCond *sync.Cond, threadWait *sync.WaitGroup) {
	defer threadWait.Done()
	defer logging.Log.Notice("(%s) Finished monitoring channel", channelID)

	logging.Log.Info("(%s) Waiting for history", channelID)
	histCond.L.Lock()
	histCond.Wait()
	histCond.L.Unlock()

	logging.Log.Debug("(%s) Message queue has %v items", channelID, msgQueue.Len())

	logging.Log.Info("(%s) Monitor size: %v", channelID, monitorSize)
	logging.Log.Info("(%s) Do logging: %v", channelID, doLogging)

	logging.Log.Info("(%s) Waiting for events", channelID)
monitorLoop:
	for {
		select {
		case message, chanOpen := <-messageChan:
			if ! chanOpen {
				logging.Log.Error("(%s) Incoming message channel is closed", channelID)
				break monitorLoop
			}
			/*
			logging.Log.Debug("(%s) Received message", channelID)
			logging.Log.Debug("\tUser: %s", message.User)
			logging.Log.Debug("\tChannel: %s", message.Channel)
			logging.Log.Debug("\tTimestamp: %s", message.Timestamp)
			logging.Log.Debug("\tText: %s", message.Text)
			logging.Log.Debug("\tSubtype: %s", message.SubType)
			*/

			msgQueue.Push(message.Timestamp)
			toDelete := msgQueue.Poll().(string)
			//logging.Log.Debug("(%s) Adding to queue: %s; Removing from queue: %s", channelID, message.Timestamp, toDelete)
			api.DeleteMessage(channelID, toDelete)

		}
	}

	return
}
