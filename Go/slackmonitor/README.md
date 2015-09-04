## SlackMonitor

### Purpose

SlackMonitor will clean the history of any number of Slack channels, and monitor the channels continuously to maintain their respective limits.

For each channel, the size limit will first be applied to the existing history of the channel.  Beyond that limit, older messages will be deleted.  If monitoring of the channel is desired, as each new message is detected, the oldest existing message will be deleted.

### Usage

```
> /path/to/SlackMonitor --config [config file]
```

### Notes

Message deletion requires a real user, not a bot.  Therefore the username specified in the configuration file and the matching token in .p8_authentications.json must be for a real user.  Also note that this restriction (message deletion requiring a real user) contradicts what's stated in the Slack API [documentation](https://api.slack.com/bot-users).  As of yet, this issue has not been pursued.