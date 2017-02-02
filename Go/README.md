# swarm/Go

## Contents

### Executables

* **diopsid** -- Monitors disk usage and sends dripline alerts
* **dungbeetle** -- Cleans up empty folders
* **mdreceiver** -- Receives data via dripline and saves JSON files
* **operator** -- Slack bot to handle operator requests and other commands
* **SlackMonitor** -- Cleans channel histories and maintains history size limits

### Packages

* **authentication** -- Interface for the Project 8 authentications scheme
* **logging** -- Standardized terminal logging
* **utility** -- A collection of useful tools

## Installation

swarm/Go requires Golang 1.1 or better.

Once you have your Golang environemnt set up ([e.g.](http://golang.org/doc/code.html#Workspaces)), use this command to install all of swarm/Go:

```
> go get github.com/project8/swarm/...
```

The ellipses are important, as they tell Go that packages exist in subdirectories.

## Updating

Use `go get` to get, build, and install the updated package.  The `-u` option for `go get` must be included to update swarm and its dependencies:

```
> go get -u github.com/project8/swarm/...
```

If there are problems updating either swarm or some of its dependencies, simply delete the appropriate directory in the [go root]/src directory, and repeate the `go get -u` command.
