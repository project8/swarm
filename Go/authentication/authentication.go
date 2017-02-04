/*
* authentication.go
*
* Loads and stores the authentication credentials that can be used by Hornet
*
* The current set of authenticators is:
*   - AMQP (username/password)
*   - Slack (token)
*/

package authentication

import (
	"encoding/json"
	"io/ioutil"
	"os/user"
	"path/filepath"
)

type AmqpCredentialType struct {
	Username string `json:"username"`
	Password string `json:"password"`
	Available bool
}

type SlackCredentialType struct {
	Tokens map[string]string
	Available bool
}

type GoogleCredentialType struct {
	JSONKey []byte
	Available bool
}

type AuthenticatorsType struct {
	Amqp AmqpCredentialType `json:"amqp"`
	Slack SlackCredentialType
	Google GoogleCredentialType
}

var Authenticators AuthenticatorsType

func Load() (e error) {
	// Get the home directory, where the authenticators live
	usr, usrErr := user.Current()
	if usrErr != nil {
		e = usrErr
		return
	}
	//log.Println( usr.HomeDir )

	// Read in the authenticators file
	authFilePath := filepath.Join(usr.HomeDir, ".project8_authentications.json")
	authFileData, fileErr := ioutil.ReadFile(authFilePath)
	if fileErr != nil {
		e = fileErr
		return
	}

	// Unmarshal the JSON data
	var authDecodedData map[string]interface{}
	if jsonErr := json.Unmarshal(authFileData, &authDecodedData); jsonErr != nil {
		e = jsonErr
		return
	}

	// Decode AMQP authentication
	Authenticators.Amqp.Available = false
	if amqpDecodedData_raw, hasAmqp := authDecodedData["amqp"]; hasAmqp {
		amqpDecodedData := amqpDecodedData_raw.(map[string]interface{})
		if username, hasUsername := amqpDecodedData["username"]; hasUsername {
			Authenticators.Amqp.Username = username.(string)
			if password, hasPassword := amqpDecodedData["password"]; hasPassword {
				Authenticators.Amqp.Password = password.(string)
				Authenticators.Amqp.Available = true
			}
		}
	}

	// Decode Slack authentication
	if slackDecodedData_raw, hasSlack := authDecodedData["slack"]; hasSlack {
		slackDecodedData := slackDecodedData_raw.(map[string]interface{})
		Authenticators.Slack.Tokens = make(map[string]string)
		for username, token := range slackDecodedData {
			Authenticators.Slack.Tokens[username] = token.(string)
		}
		if len(Authenticators.Slack.Tokens) > 0 {
			Authenticators.Slack.Available = true
		}
	}

	if googleDecodedData_raw, hasGoogle := authDecodedData["google"]; hasGoogle {
		var marshallErr error
		Authenticators.Google.JSONKey, marshallErr = json.Marshal(googleDecodedData_raw)
		if marshallErr != nil {
			e = marshallErr
			return
		}
		//Authenticators.Google.JSONKey = jsonKey
		Authenticators.Google.Available = true
	}

	return
}


// AMQP Convenience Functions

func AmqpAvailable() bool {
	return Authenticators.Amqp.Available
}

func AmqpUsername() string {
	if Authenticators.Amqp.Available {
		return Authenticators.Amqp.Username
	} else {
		return ""
	}
}

func AmqpPassword() string {
	if Authenticators.Amqp.Available {
		return Authenticators.Amqp.Password
	} else {
		return ""
	}
}

// Slack Convenience Functions

func SlackAvailable(username string) bool {
	_, hasUser := Authenticators.Slack.Tokens[username]
	return hasUser
}

func SlackToken(username string) string {
	token, hasUser := Authenticators.Slack.Tokens[username]
	if hasUser {
		return token
	} else {
		return ""
	}
}

// Google Convenience Functions

func GoogleAvailable() bool {
	return Authenticators.Google.Available
}

func GoogleJSONKey() []byte {
	return Authenticators.Google.JSONKey
}

