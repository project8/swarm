/*
* logger.go
*
* Logging functions with severity
*
*/

package logging

import (
	"fmt"
	"os"

	"github.com/op/go-logging"
)

// global logger
var Log = logging.MustGetLogger("swarm.logging")
var LogBackendLvl logging.LeveledBackend
var format = logging.MustStringFormatter(
    "%{color}%{id:03x} %{time:15:04:05.000} %{level:.4s} [%{shortfunc}] ▶ %{message}%{color:reset}",
)

var currentBackends []logging.Backend
func AddBackend(backend logging.Backend) {
	currentBackends = append(currentBackends, backend)
	logging.SetBackend(currentBackends...)
}

func InitializeLogging() {
	backend := logging.NewLogBackend(os.Stdout, "", 0)
	backendFormatter := logging.NewBackendFormatter(backend, format)
	LogBackendLvl = logging.AddModuleLevel(backendFormatter)
	LogBackendLvl.SetLevel(logging.INFO, "")
	AddBackend(LogBackendLvl)
}

func ConfigureLogging(level string) {
	if level, levelErr := logging.LogLevel(level); levelErr != nil {
		fmt.Printf("Warning: invalid logging-level configuration value: %s", level)
	} else {
		LogBackendLvl.SetLevel(level, "")
	}
}
