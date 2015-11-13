package main

import (
	"flag"
	"io/ioutil"
	"os"
	"path/filepath"
	"time"

	"github.com/spf13/viper"

	"github.com/project8/swarm/Go/logging"
)

// processDir will remove empty directories older than maxAge, and recursively process children of non-empty directories
func processDir(dirInfo os.FileInfo, basePath string, maxAge time.Duration) error {
	dirName := filepath.Join(basePath, dirInfo.Name())
	logging.Log.Debug("Processing directory <%s>", dirName)
	dirContents, readDirErr := ioutil.ReadDir(dirName)
	if readDirErr != nil {
		logging.Log.Error("Unable to read directory <%s>: %v", dirName, readDirErr)
		return readDirErr
	}
	if len(dirContents) == 0 {

		// Directory is empty, check if we need to remove it
		logging.Log.Debug("Directory is empty; checking age")
		if time.Since(dirInfo.ModTime()) > maxAge {
			// Ok, then remove the directory
			if remErr := os.Remove(dirName); remErr != nil {
				logging.Log.Error("Unable to remove an empty directory <%s>: %v", dirName, remErr)
				return remErr
			}
			logging.Log.Info("Successfully removed directory <%s>", dirName)
		}

	} else {

		// Directory is not empty; process its contents
		for _, fileInfo := range dirContents {
			logging.Log.Debug("Directory <%s> is not empty; processing contents", dirName)
			if fileInfo.IsDir() {
				if procErr := processDir(fileInfo, dirName, maxAge); procErr != nil {
					logging.Log.Error("An error occurred while processing directory <%s>: %v", fileInfo.Name(), procErr)
					// pass errors back up through recursion chain
					return procErr
				}
			} // else it's a file; ignore it
		}

	}

	logging.Log.Debug("No action taken on directory <%s>", dirName)
	return nil
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
		"Display this dialog")
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
	viper.SetDefault("log-level", "INFO")
	viper.SetDefault("maximum-age", "1h")
	viper.SetDefault("root-dir", "") // this will be invalid, so it must be manually configured
	viper.SetDefault("wait-interval", "10m")

	// load config
	if configFile != "" {
		viper.SetConfigFile(configFile)
		if parseErr := viper.ReadInConfig(); parseErr != nil {
			logging.Log.Critical("%v", parseErr)
			os.Exit(1)
		}
		logging.Log.Notice("Config file loaded")
	}
	logging.ConfigureLogging(viper.GetString("log-level"))
	logging.Log.Info("Log level: %v", viper.GetString("log-level"))

	maxAge := viper.GetDuration("maximum-age")
	waitInterval := viper.GetDuration("wait-interval")

	rootDir, rdErr := filepath.Abs(filepath.Clean(viper.GetString("root-dir")))
	if rdErr != nil {
		logging.Log.Critical("Unable to get absolute form of the root directory <%s>", viper.GetString("root-dir"))
		os.Exit(1)
	}
	if rootDir == "" {
		logging.Log.Critical("Root directory (\"root-dir\") was not specified in the config file")
		os.Exit(1)
	}

	// Do a couple checks on the root directory
	rootDirInfo, statErr := os.Stat(rootDir)
	if statErr != nil {
		logging.Log.Critical("Unable to \"Stat\" the root directory <%s>", rootDir)
		os.Exit(1)
	}
	if ! rootDirInfo.IsDir() {
		logging.Log.Critical("\"root-dir\" provided <%s> is not a directory", rootDir)
		os.Exit(1)
	}

	if chdirErr := os.Chdir(rootDir); chdirErr != nil {
		logging.Log.Critical("Unable to change to the root directory <%s>: %v", rootDir, chdirErr)
		os.Exit(1)
	}

	logging.Log.Notice("Watching for stale directories.  Use ctrl-c to exit")

//mainLoop:
	for {
		// Loop over the contents of rootDir
		// We don't apply processDir() directly to rootDir because we don't want to delete rootDir if it's empty
		logging.Log.Debug("Processing directory <%s>", rootDir)
		dirContents, readDirErr := ioutil.ReadDir(rootDir)
		if readDirErr != nil {
			logging.Log.Critical("Unable to read directory <%s>", rootDir)
			os.Exit(1)
		}

		exitOnErrors := false

		for _, fileInfo := range dirContents {
			logging.Log.Debug("Directory <%s> is not empty; processing contents", rootDir)
			if fileInfo.IsDir() {
				if procErr := processDir(fileInfo, rootDir, maxAge); procErr != nil {
					logging.Log.Error("An error occurred while processing directory <%s>: %v", fileInfo.Name(), procErr)
					exitOnErrors = true
				}
			} // else it's a file; ignore it
		}
		logging.Log.Debug("Finished processing <%s>", rootDir)

		if exitOnErrors == true {
			logging.Log.Critical("Exiting due to directory-processing errors")
			break
		}

		// Wait the specified amount of time before running again
		time.Sleep(waitInterval)
	}

	logging.Log.Notice("DungBeetle says: \"My job here is done\"")
}
