package main

import (
	"flag"
	"fmt"
	"os"
	"sync"

	"github.com/nlopes/slack"
	"github.com/spf13/viper"

	"github.com/project8/swarm/Go/authentication"
	"github.com/project8/swarm/Go/utility"
)

var monitorStarted bool = false

type eventRecipient struct {
	channelID string
	eventChan chan *slack.MessageEvent
}

func main() {
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

	// load config
	viper.SetConfigFile(configFile)
	if parseErr := viper.ReadInConfig(); parseErr != nil {
		fmt.Printf("%v", parseErr)
		os.Exit(1)
	}
	fmt.Printf("Config file loaded\n")

	userName := viper.GetString("username")

	// check authentication for desired username
	if authErr := authentication.Load(); authErr != nil {
		fmt.Printf("Error in loading authenticators: %v\n", authErr)
		os.Exit(1)
	}

	if ! authentication.SlackAvailable(userName) {
		fmt.Printf("Authentication for user <%s> is not available\n", userName)
		os.Exit(1)
	}
	authToken := authentication.SlackToken(userName)

	fmt.Printf("Slack username: %s\n", userName)
	fmt.Printf("Slack token: %s\n", authToken)

	// get the slack API object
	api := slack.New(authToken)
	if api == nil {
		fmt.Printf("Unable to make a new Slack API\n")
		os.Exit(1)
	}
	fmt.Printf("Created Slack API\n")
	// get list of users and then the user ID
	userID := ""
	users, usersErr := api.GetUsers()
	if usersErr != nil {
		fmt.Printf("Unable to get users: %s\n", usersErr)
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
		fmt.Printf("Could not get user ID for user <%s>\n", userName)
		os.Exit(1)
	}
	fmt.Printf("User ID: %s\n", userID)

	// get map of channel IDs
	var allChannelMap map[string]string
	channels, chanErr := api.GetChannels(true)
	if chanErr != nil {
		fmt.Printf("Unable to get channels: %s\n", chanErr)
		return
	} else {
		allChannelMap = make(map[string]string, len(channels))
		for _, aChan := range channels {
			allChannelMap[aChan.Name] = aChan.ID
			fmt.Printf("Channel %s has ID %s\n", aChan.Name, aChan.ID)
		}
	}

	var threadWait sync.WaitGroup
	recipientChan := make(chan eventRecipient, 10)

	// loop over channels
	channelsRaw := viper.GetStringMap("channels")
	for channelName, _ := range channelsRaw {
		// get channel ID
		if channelID, channelExists := allChannelMap[channelName]; channelExists == true {
			fmt.Printf("Found request for channel %s (%s)\n", channelName, channelID)
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
				fmt.Printf("(%v) Invalid size limit", channelID)
				continue
			}

			doLogging := false // future feature

			msgQueue := utility.NewQueue()

			var buildHistLock sync.Mutex
			histCond := sync.NewCond(&buildHistLock)

			if monitorSize || doLogging {
				if ! monitorStarted {
					threadWait.Add(1)
					go monitorSlack(api, &threadWait, recipientChan)
					monitorStarted = true
				}

				recipient := eventRecipient{
					channelID: channelID,
					eventChan: make(chan *slack.MessageEvent, 100),
				}
				recipientChan <- recipient

				threadWait.Add(1)
				go monitorChannel(channelID, api, recipient.eventChan, msgQueue, monitorSize, doLogging, histCond, &threadWait)
	
				// If we're monitoring the channel, then we increase the size limit for cleanHistory by 1.
				// This is because the latest message will be passed to the monitor as the first message received.
				// So the monitor will then remove an old message; so we leave one extra old message to be removed
				// once the monitoring begins.
				sizeLimit++
			}

			fmt.Printf("(%v) Launching cleanHistory\n", channelID)
			threadWait.Add(1)
			go cleanHistory(channelID, api, msgQueue, sizeLimit, histCond, &threadWait)

		} else {
			fmt.Printf("Warning: Channel <%s> does not exist\n", channelName)
		}
	}

	fmt.Printf("Waiting for threads to finish\n")
	threadWait.Wait()
	fmt.Printf("Threads complete\n")

	return
}


func cleanHistory(channelID string, api *slack.Client, msgQueue *utility.Queue, histSize int, histCond *sync.Cond, threadWait *sync.WaitGroup) {
	defer threadWait.Done()
	defer histCond.Signal()

	histCond.L.Lock()
	defer histCond.L.Unlock()

	fmt.Printf("(%v) Starting cleanHistory on %s\n", channelID, channelID)
	histParams := slack.NewHistoryParameters()
	histParams.Inclusive = true

	histCountMax := 1000

	// build history with histSize messages
	fmt.Printf("(%v) Building history with %v messages\n", channelID, histSize)
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
			fmt.Printf("(%v) Unable to get the channel history: %v\n", channelID, histErr)
			return
		}

		iLastMsg := len(history.Messages) - 1
		//fmt.Printf("0: %v, %v: %v\n", history.Messages[0].Timestamp, iLastMsg, history.Messages[iLastMsg].Timestamp)

		fmt.Printf("(%v) In skip loop; obtained history with %v messages\n", channelID, len(history.Messages))

		for iMsg := iLastMsg; iMsg >= 0; iMsg-- {
			msgQueue.Push(history.Messages[iMsg].Timestamp)
			fmt.Printf("(%v) Pushing to queue: %s\n", channelID, history.Messages[iMsg].Timestamp)
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
			fmt.Printf("(%v) Unable to get the channel history: %v\n", channelID, histErr)
			return
		}

		fmt.Printf("(%v) Deleting %v items (latest: %v)\n", channelID, len(history.Messages), history.Latest)

		for _/*iMsg*/, message := range history.Messages {
			fmt.Printf("(%v) Deleting: %s\n", channelID, message.Timestamp)
			_, _, /*respChan, respTS,*/ respErr := api.DeleteMessage(channelID, message.Timestamp)
			if respErr != nil {
				fmt.Printf("(%v) Unable to delete message: %v", respErr)
			}
			//fmt.Printf("(%v) Deletion response: %s, %s, %v\n", channelID, respChan, respTS, respErr)
			nDeleted++
		}
		histParams.Latest = history.Messages[len(history.Messages)-1].Timestamp
	}
	fmt.Printf("(%v) Deleted %v messages\n", channelID, nDeleted)

	return
}


func monitorSlack(api *slack.Client, threadWait *sync.WaitGroup, recipientChan chan eventRecipient) {
	defer threadWait.Done()
	defer fmt.Printf("Finished monitoring Slack\n")

	eventRecipients := make(map[string]chan *slack.MessageEvent)

	fmt.Printf("Connecting to RTM\n")
	rtm := api.NewRTM()
	fmt.Printf("%v", rtm)
	go rtm.ManageConnection()
	defer func() {
		fmt.Printf("Disconnecting from RTM\n")
		if discErr := rtm.Disconnect(); discErr != nil {
			fmt.Printf("Error while disconnecting from the Slack RTM\n")
		}
	}()

	fmt.Printf("Waiting for events\n")
monitorLoop:
	for {
		select {
		case recipient := <-recipientChan:
			eventRecipients[recipient.channelID] = recipient.eventChan
			fmt.Printf("Added event recipient for %v\n", recipient.channelID)

		case event, chanOpen := <-rtm.IncomingEvents:
			if ! chanOpen {
				fmt.Printf("Incoming events channel is closed\n")
				break monitorLoop
			}
			switch evData := event.Data.(type) {
			case *slack.HelloEvent:
				fmt.Printf("Slack says \"hello\"\n")

			case *slack.ConnectedEvent:
				//fmt.Println("Infos:", evData.Info)
				fmt.Printf("Connected to Slack\n")
				fmt.Printf("Connection counter: %v\n", evData.ConnectionCount)
				// Replace #general with your Channel ID
				//rtm.SendMessage(rtm.NewOutgoingMessage("Hello world", "#general"))

			case *slack.MessageEvent:
				//fmt.Printf("Message: %v\n", evData)
				if evData.SubType != "" {
					break
				}
				if eventChan, hasEventChan := eventRecipients[evData.Channel]; hasEventChan {
					eventChan <- evData
				}

			//case *slack.PresenceChangeEvent:
			//	fmt.Printf("Presence Change: %v\n", evData)

			case *slack.LatencyReport:
				fmt.Printf("Current latency: %v\n", evData.Value)

			case *slack.RTMError:
				fmt.Printf("Error: %s\n", evData.Error())

			case *slack.InvalidAuthEvent:
				fmt.Printf("Invalid credentials\n")
				break monitorLoop

			default:

				// Ignore other events..
				//fmt.Printf("Unexpected: %v\n", event.Data)
			}
		}
	}

	return
}

func monitorChannel(channelID string, api *slack.Client, messageChan <-chan *slack.MessageEvent, msgQueue *utility.Queue, monitorSize, doLogging bool, histCond *sync.Cond, threadWait *sync.WaitGroup) {
	defer threadWait.Done()
	defer fmt.Printf("(%v) Finished monitoring channel\n", channelID)

	fmt.Printf("(%v) Waiting for history\n", channelID)
	histCond.L.Lock()
	histCond.Wait()
	histCond.L.Unlock()

	fmt.Printf("(%v) Message queue has %v items\n", channelID, msgQueue.Len())

	fmt.Printf("(%v) Monitor size: %v\n", channelID, monitorSize)
	fmt.Printf("(%v) Do logging: %v\n", channelID, doLogging)

	fmt.Printf("(%v) Waiting for events\n", channelID)
monitorLoop:
	for {
		select {
		case message, chanOpen := <-messageChan:
			if ! chanOpen {
				fmt.Printf("(%v) Incoming message channel is closed", channelID)
				break monitorLoop
			}
			/*
			fmt.Printf("(%v) Received message\n", channelID)
			fmt.Printf("\tUser: %s\n", message.User)
			fmt.Printf("\tChannel: %s\n", message.Channel)
			fmt.Printf("\tTimestamp: %s\n", message.Timestamp)
			fmt.Printf("\tText: %s\n", message.Text)
			fmt.Printf("\tSubtype: %s\n", message.SubType)
			*/

			msgQueue.Push(message.Timestamp)
			toDelete := msgQueue.Poll().(string)
			//fmt.Printf("(%v) Adding to queue: %s; Removing from queue: %s\n", channelID, message.Timestamp, toDelete)
			api.DeleteMessage(channelID, toDelete)

		}
	}

	return
}
