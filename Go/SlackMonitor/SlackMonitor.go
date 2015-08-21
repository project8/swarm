package main

import (
	"fmt"

	"github.com/project8/swarm/Go/authentication"

	"github.com/nlopes/slack"
)

func main() {
	if authErr := authentication.Load(); authErr != nil {
		fmt.Printf("Error in loading authenticators: %v\n", authErr)
		return
	}

	fmt.Printf("Authenticators loaded\n%v\n", authentication.Authenticators)

	slackUser := "project8"

	if authentication.SlackAvailable(slackUser) == false {
		fmt.Printf("Do not have authentication for user <%s>", slackUser)
		return
	}

    api := slack.New(authentication.SlackToken(slackUser))

    authResp, atErr := api.AuthTest()
    if atErr != nil {
    	fmt.Printf("Unable to complete AuthTest: %s\n", atErr)
    	return
    }
    fmt.Printf("Auth test: \n%v\n", authResp)

    chanName := "github"
    var chanID string
    channels, chanErr := api.GetChannels(true)
    if chanErr != nil {
    	fmt.Printf("Unable to get channels: %s\n", chanErr)
    	return
    } else {
    	fmt.Printf("All channels:\n")
    	for index, aChan := range channels {
    		fmt.Printf("%v: ID=%v, Name=%v\n", index, aChan.ID, aChan.Name)
    		if aChan.Name == chanName {
    			chanID = aChan.ID
    		}
    	}
    }

    chanInfo, chanInfoErr := api.GetChannelInfo(chanID)
    if chanInfoErr != nil {
    	fmt.Printf("Unable to get channel info: %s\n", chanInfoErr)
    	return
    } else {
    	fmt.Printf("Channel info: \n%v\n", chanInfo)
    }

    userName := "nsoblath"
    var userID string
    users, usersErr := api.GetUsers()
    if usersErr != nil {
    	fmt.Printf("Unable to get users: %s\n", usersErr)
    	return
    } else {
    	fmt.Printf("Users:\n")
    	for index, user := range users {
    		fmt.Printf("%v: ID=%v, Name=%v\n", index, user.ID, user.Name)
    		if user.Name == userName {
    			userID = user.ID
    		}
    	}
    }

    user, userErr := api.GetUserInfo(userID)
    if userErr != nil {
        fmt.Printf("Unable to get user info: %s\n", userErr)
        return
    }
    fmt.Printf("ID: %s, Fullname: %s, Email: %s\n", user.ID, user.Profile.RealName, user.Profile.Email)

    return
}