package main

import (
	"flag"
	"os"
	"os/user"
	"strings"
  "fmt"
  "runtime"
  "sync"
	"github.com/nlopes/slack"
	"github.com/spf13/viper"

	"github.com/project8/swarm/Go/authentication"
	"github.com/project8/swarm/Go/logging"

	"encoding/json"
	"io/ioutil"
	"net/http"
	"net/url"
	"path/filepath"
	"time"

	"golang.org/x/net/context"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/calendar/v3"

)

var monitorStarted bool = false


func getOperatorTag(operator string) (string) {
	return "(<@" + operator + ">)"
}
func inTimeSpan(start, end, check time.Time) bool {
    return check.After(start) && check.Before(end)
}
	// Setting Google Calendar interfacing

	// getClient uses a Context and Config to retrieve a Token
	// then generate a Client. It returns the generated Client.


func getClient(ctx context.Context, config *oauth2.Config) *http.Client {
	cacheFile, err := tokenCacheFile()
	if err != nil {
		logging.Log.Fatalf("Unable to get path to cached credential file. %v", err)
	}
	tok, err := tokenFromFile(cacheFile)
	if err != nil {
		tok = getTokenFromWeb(config)
		saveToken(cacheFile, tok)
	}
	return config.Client(ctx, tok)
}

// getTokenFromWeb uses Config to request a Token.
// It returns the retrieved Token.
func getTokenFromWeb(config *oauth2.Config) *oauth2.Token {
	authURL := config.AuthCodeURL("state-token", oauth2.AccessTypeOffline)
	fmt.Printf("Go to the following link in your browser then type the "+
		"authorization code: \n%v\n", authURL)

	var code string
	if _, err := fmt.Scan(&code); err != nil {
		logging.Log.Fatalf("Unable to read authorization code %v", err)
	}

	tok, err := config.Exchange(oauth2.NoContext, code)
	if err != nil {
		logging.Log.Fatalf("Unable to retrieve token from web %v", err)
	}
	return tok
}

// tokenCacheFile generates credential file path/filename.
// It returns the generated credential path/filename.
func tokenCacheFile() (string, error) {
	usr, err := user.Current()
	if err != nil {
		return "", err
	}
	tokenCacheDir := filepath.Join(usr.HomeDir, ".credentials")
	os.MkdirAll(tokenCacheDir, 0700)
	return filepath.Join(tokenCacheDir,
		url.QueryEscape("calendar-go-quickstart.json")), err
}

// tokenFromFile retrieves a Token from a given file path.
// It returns the retrieved Token and any read error encountered.
func tokenFromFile(file string) (*oauth2.Token, error) {
	f, err := os.Open(file)
	if err != nil {
		return nil, err
	}
	t := &oauth2.Token{}
	err = json.NewDecoder(f).Decode(t)
	defer f.Close()
	return t, err
}

// saveToken uses a file path to create a file and store the
// token in it.
func saveToken(file string, token *oauth2.Token) {
	fmt.Printf("Saving credential file to: %s\n", file)
	f, err := os.Create(file)
	if err != nil {
		logging.Log.Fatalf("Unable to cache oauth token: %v", err)
	}
	defer f.Close()
	json.NewEncoder(f).Encode(token)
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

	// Information about the operator and temporary operators
	theOperator := ""
	theOperatorTag := ""
	var tempOperators map[string]string
	tempOperators = make(map[string]string)

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
			"\t`!whoisop`: show who the current operator is, plus any temporary operators\n" +
			"\t`!startshift`: manually start your shift, replacing the existing operator\n" +
			"\t`!endshift`: remove yourself as the operator\n" +
			"\t`!overrideshift [username (optional)]`: replace the current operator with a manually-specified operator; if no operator is specified, the current operator will be removed\n" +
			"\t`!tempoperator [username (optional)]`: add yourself or someone else as a temporary operator; leave the username blank to add yourself\n" +
			"\t`!removetempoperator [username (optional)]`: remove yourself or someone else as temporary operator; leave the username blank to remove yourself"
			logging.Log.Debug("Printing help message")
			helpMsg := rtm.NewOutgoingMessage(msgText, msg.Channel)
			rtm.SendMessage(helpMsg)
	}
	commandMap["!whoisop"] = func(_ string, msg *slack.MessageEvent) {
		if theOperator == "" && len(tempOperators) == 0 {
			wioMessage := rtm.NewOutgoingMessage("There is no operator assigned right now", msg.Channel)
			rtm.SendMessage(wioMessage)
			return
		}
		var msgText string
		if theOperator != "" {
			msgText += "The operator is " + userIDMap[theOperator] + ".  "
		}
		if len(tempOperators) != 0 {
			msgText += "Temporary operators: "
			for userID, _ := range tempOperators {
				msgText += userIDMap[userID] + " "
			}
		}
		wioMessage := rtm.NewOutgoingMessage(msgText, msg.Channel)
		rtm.SendMessage(wioMessage)
		return
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
	commandMap["!tempoperator"] = func(username string, msg *slack.MessageEvent) {
		if username != "" {
			logging.Log.Info("Adding as temporary operator user " + username)
			newUserID, hasID := userNameMap[username]
			if ! hasID {
				logging.Log.Warningf("Unknown username: %s", username)
				osMessage := rtm.NewOutgoingMessage("I'm sorry, I don't recognize that username", msg.Channel)
				rtm.SendMessage(osMessage)
				return
			}
			tempOperators[newUserID] = getOperatorTag(newUserID)
			osMessage := rtm.NewOutgoingMessage("Use your powers wisely, " + userIDMap[newUserID] + "!", msg.Channel)
			rtm.SendMessage(osMessage)
		} else {
			logging.Log.Info("Adding as temporary operator user " + userIDMap[msg.User])
			tempOperators[msg.User] = getOperatorTag(msg.User)
			ssMessage := rtm.NewOutgoingMessage("Use your powers wisely, " + userIDMap[msg.User] + "!", msg.Channel)
			rtm.SendMessage(ssMessage)
		}
	}
	commandMap["!removetempoperator"] = func(username string, msg *slack.MessageEvent) {
		toRemove := msg.User
		if username != "" {
			logging.Log.Info("Removing " + username + " from temporary operatorship")
			toRemoveID, hasID := userNameMap[username]
			if ! hasID {
				logging.Log.Warningf("Unknown username: %s", username)
				osMessage := rtm.NewOutgoingMessage("I'm sorry, I don't recognize that username", msg.Channel)
				rtm.SendMessage(osMessage)
				return
			}
			toRemove = toRemoveID
		}

		_, isTempOp := tempOperators[toRemove]
		if ! isTempOp {
			logging.Log.Debug("Received remove-temp-operator instruction from non-temp-operator")
			usMessage := rtm.NewOutgoingMessage(userIDMap[toRemove] + " is not currently listed as a temporary operator", msg.Channel)
			rtm.SendMessage(usMessage)
			return
		}
		logging.Log.Info("Removing temporary operator " + userIDMap[toRemove])
		delete(tempOperators, toRemove)
		usMessage := rtm.NewOutgoingMessage("OK, you're all done, thanks!", msg.Channel)
		rtm.SendMessage(usMessage)
		return
	}

	arrivedMsg := rtm.NewOutgoingMessage("Have no fear, @operator is here!", channelID)
	rtm.SendMessage(arrivedMsg)


	// Google Calendar starts here
	ctx := context.Background()
		usr, usrErr := user.Current()
		if usrErr != nil {
				logging.Log.Fatalf("error")
		return
		}
		CalAuthPath:=filepath.Join(usr.HomeDir, "client_secret.json")
	  b, err := ioutil.ReadFile(CalAuthPath)
	  if err != nil {
	    logging.Log.Fatalf("Unable to read client secret file: %v", err)
	  }

	  // If modifying these scopes, delete your previously saved credentials
	  // at ~/.credentials/calendar-go-quickstart.json
	  config, err := google.ConfigFromJSON(b, calendar.CalendarReadonlyScope)
	  if err != nil {
	    logging.Log.Fatalf("Unable to parse client secret file to config: %v", err)
	  }
	  client := getClient(ctx, config)

	  srv, err := calendar.New(client)
	  if err != nil {
	    logging.Log.Fatalf("Unable to retrieve calendar Client %v", err)
	  }


// Setting the Multithreading
	runtime.GOMAXPROCS(2)

	var wg sync.WaitGroup
	wg.Add(2)

	fmt.Println("Starting Go Routines")

	temp :=0
// Starting the two loops
	go func() {
			defer wg.Done()

			for number := 1; number < 27; number++ {
					fmt.Printf("a%d ", number)
			}
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
				if strings.Contains(evData.Text, botUserTag) {
					if theOperator == "" && len(tempOperators) == 0 {
						logging.Log.Info("Got operator message, but no operator is assigned")
						notifyMsg := rtm.NewOutgoingMessage("No operator assigned", evData.Channel)
						rtm.SendMessage(notifyMsg)
						continue
					}

					logging.Log.Info("Attempting to notify the operator")
					notification := theOperatorTag
					for _, tag := range tempOperators {
						notification += " " + tag
					}
					notifyMsg := rtm.NewOutgoingMessage(notification, evData.Channel)
					rtm.SendMessage(notifyMsg)
					continue
				}
			}
		}
	}
}()


	go func() {
			defer wg.Done()

			for number := 1; number < 200; number++ {

						t := time.Now().Format(time.RFC3339)
						events, err := srv.Events.List("primary").ShowDeleted(false).
							SingleEvents(true).TimeMin(t).MaxResults(100).OrderBy("startTime").Do()
						if err != nil {
							logging.Log.Fatalf("Unable to retrieve next ten of the user's events. %v", err)
						}
						// var buffer bytes.Buffer
						// buffer.WriteString("Found ")
						// buffer.WriteString(strconv.FormatInt(int64(len(events.Items)),10))
						// buffer.WriteString(" events in the Calendar")
						//
						// errorMsg := rtm.NewOutgoingMessage(buffer.String(), channelID)
						// rtm.SendMessage(errorMsg)
						layout := "2006-01-02"
						logging.Log.Infof("Upcoming events:")
						if len(events.Items) > 0 {
							for _, i := range events.Items {
								var whenStart string
								var whenEnd string
								// If the DateTime is an empty string the Event is an all-day Event.
								// So only Date is available.

								if strings.Contains(i.Summary,"Operator:") {
									logging.Log.Infof("%s (%s -- %s)\n", i.Summary, whenStart, whenEnd)
									errorMsg := rtm.NewOutgoingMessage("Found the operator", channelID)
									rtm.SendMessage(errorMsg)
									if i.Start.DateTime != "" {
										whenStart = i.Start.DateTime
									} else {
										whenStart = i.Start.Date
									}
									if i.End.DateTime != "" {
										whenEnd = i.End.DateTime
									} else {
										whenEnd = i.End.Date
									}
									extractedTime, err := time.Parse(layout, str)

									if inTimeSpan(whenStart, whenEnd, t) {
        						logging.Log.Debugf(t, "is between", whenStart, "and", whenEnd, ".")
										logging.Log.Infof(input[len("Operator: "):]," is the operator.")
    							}

								}
							}
						} else {
							fmt.Printf("No upcoming events found.\n")
						}
						time.Sleep(10*time.Second)
			}
	}()

	fmt.Println("Waiting To Finish")
	wg.Wait()

	fmt.Println("\nTerminating Program")


	logging.Log.Info("Waiting for events")


	leavingMsg := rtm.NewOutgoingMessage("Signing off!", channelID)
	rtm.SendMessage(leavingMsg)

	logging.Log.Info("All done!")

	return
}
