package main

import (
	"flag"
	"fmt"
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


fmt.Println("                                         ...~.+=:,.I+...                        ")
fmt.Println("                                   ... .+$7II?I?IZ7$??+~?I~..                   ")
fmt.Println("                               .,.:M=.MINI++I7I??????7I$7I=+7~...               ")
fmt.Println("                              O7+8.IOIII??7+I+7Z?7$I?I?I+??+$??O?.              ")
fmt.Println("                          ..8MMMM8I??+?+=I?+I?=7++:7???I?==++??$?+?.            ")
fmt.Println("                        :MMMMM.:~+?I=++$I++???=+??I==~+++?~I=7+=+~?7.           ")
fmt.Println("                       MMM..++I~?I?$I$$??+?+~$I++=I?7==++I++I+=?+=~=~.          ")
fmt.Println("                     .MM..~.7?~==~I+?7=II=8~?++?~?=7+$~?$D?=~7?+?+~=I:.         ")
fmt.Println("                    .MM.?~~$7+=~?=:+?7I=???I7II?7I++DZ?+=+7===+=??=~+?+.        ")
fmt.Println("                   .MM..NZ?M87+??+7?++?7=I$I7=+=IMMMMD=+$~=?++?:??=~+77I.       ")
fmt.Println("                    MMNMMMMMMMMMN$Z++7I7+=+++==MMMI~~+I=?+=?++I+=I?=I7$$?:      ")
fmt.Println("                  .MMMMMMMMMMMMMM8I+++77++?:?MMM::?~=I?+??II?=+?+=?I++I?Z=      ")
fmt.Println("                 IDMMMMMMMMMMMMMMM$I?I$7==NMM=++~~=7+=I???7I??8I$+I+?I?I$??.    ")
fmt.Println("               .MMMMMMMMMMMMMMMMMMOI7?II~MMM+=7=+~7:+$?+II7I?+~I?++I??I??+I..   ")
fmt.Println("              .MMMMMMMMMMMMMMMMMMMD==:+~NMM+=+?:=+?+~=8=++?$7+I?7+=+?II$?+==.   ")
fmt.Println("             .MMMMMMMMMMMMDMMMMMMM8I?:MMM+?+~$,=:??7==???Z?7II$7II7??7++=7:I.   ")
fmt.Println("             MMMMMMMMMMMM8NMMMMDMMNMMMM??78~:,+=??~:~II?~+II?II7+I+??7I7=ZI=~.  ")
fmt.Println("             MMMMMMMMMMMNNNMMMMNMM$?II==?=?+:==++?I++==+II$I=77I??II?II8$??7I~  ")
fmt.Println("            $M7M8MMMMMMMMMMMMMMMM+?~~==+$~~7:$I+7I??~+++=?7I=?$??I?$III?7?I$7Z  ")
fmt.Println("            NDMMMMMMMMMMMMMMMMMMM7???$$I?+III787+?7?????==+++?~??I?8ZI77I?I77?. ")
fmt.Println("           NMMMMMMMMMMMMMMMMMMMMD==??=:~77?I=?77?7+I=?I7=I?~+??77~??+IIZ$7$7$7. ")
fmt.Println("          ZMMMMMMMMMMMMMMMMMMMMM8??+=,=I?IZ+++=7?I+?II???+?+?7?=?I$+IZ$O7$ZI7=. ")
fmt.Println("          MMMMMMMMMMMMMMMMMMM8MMMDIZ7+++?~?+MMMII$?~7II=I?I7ZI=7$?II7$77I$7I7Z  ")
fmt.Println("         .NMMMMMMMMDMMMMMMMMMOMO=77?=+++?MMMI=$+III??+IOO+?I?I=???ZI?I7I7$ZZZ?. ")
fmt.Println("           DMMMMMMMMMMMMMMNMMMM7?++$~+OMMMO~II?777III?I$Z=?I7I7?+II+I7II7787O.  ")
fmt.Println("           .8MMMMMMMMMMMMN7NMMZ+I=~~IOMMOD/?O=7$7+$7?777I+?I+I?I7Z?7?7I7$8OZ.   ")
fmt.Println("       . ..$MMNMMMMMMMMMMMMMNM7+7MMMMMNO+++I??IO77II+8$I?I=+?+I7?+?Z$I7I$$7~,   ")
fmt.Println("       .NNMMMMMMMMMMMMMMMMMIMMMMMMZM+?7+?+I+++7ZI7$777??I7I7II777$D8I?+II7Z,.   ")
fmt.Println("     .MMMMMMMMMMMMMNMMMMI+~,==?~~7~+++7II7=?II$?IZI7+?II?=IO7?I=+?7Z7O??$D.     ")
fmt.Println(".MMNMMM MMMMMMMMMNDMMNMMMZ+~==~::?~I$?I~~II78=7?77?I?II=7??I$II?II$$IZ7$$.      ")
fmt.Println(" .O    NMMMMMMMMMMD..~NMMM.7=,+~~+?7I?=$+I$7II7I7IIII888$Z$I?II7ZI$7II+$~       ")
fmt.Println("      .~NMNMMMMMDZ.   ..MMM.+~=+==+Z7777$7Z77I?7Z$II77ZII$O7I+II?7+7I+~I        ")
fmt.Println("         .MM$=. .M    .:NMM~.=+?II$III7IIIII$Z?+$O$$?I77777III77I?7?++:         ")
fmt.Println("           8NN.  MM  .ZMMM$ .:????8887+7III7O$$+Z$$$7Z$O77I??+$=????=           ")
fmt.Println("          .MM:.  MM. .DMM.    ..7+~==?+7$88$$OIOID$7I77???+$7===+I              ")
fmt.Println("            ,M.  +M,  .M,.       .~:~,~~=7$Z87I7II7Z7I$I=~~=+=?..               ")
fmt.Println("           ,MMZ,.       7.         Z~I+:?++==~7Z:7~=7OZ=+++,                    ")
fmt.Println("            .MMMMM,                    ..~:... .......                          ")
fmt.Println("           .D?.  ?,                                                             ")
fmt.Println("                                                                                ")

	// defult configuration
	viper.SetDefault("log-level", "INFO")
	viper.SetDefault("maximum-age", "1h")
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

	rootDirs := viper.GetStringSlice("root-dirs")
	if len(rootDirs) == 0 {
		logging.Log.Critical("No root directories were provided")
		os.Exit(1)
	}

	// Clean up and check the root directories
	for rdInd, rootDir := range rootDirs {
		rootDirAbs, rdErr := filepath.Abs(filepath.Clean(rootDir))
		if rdErr != nil {
			logging.Log.Critical("Unable to get absolute form of the root directory <%s>", rootDir)
			os.Exit(1)
		}

		// Do a couple checks on the root directory
		rootDirInfo, statErr := os.Stat(rootDirAbs)
		if statErr != nil {
			logging.Log.Critical("Unable to \"Stat\" the root directory <%s>", rootDirAbs)
			os.Exit(1)
		}
		if ! rootDirInfo.IsDir() {
			logging.Log.Critical("Root directory <%s> is not a directory", rootDirAbs)
			os.Exit(1)
		}

		rootDirs[rdInd] = rootDirAbs
	}


	logging.Log.Notice("Watching for stale directories.  Use ctrl-c to exit")

//mainLoop:
	for {
		// Loop over the contents of rootDirs
		// We don't apply processDir() directly to the rootDirs because we don't want to delete rootDir if it's empty
		for _, rootDir := range rootDirs {
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
		}

		// Wait the specified amount of time before running again
		time.Sleep(waitInterval)
	}

	logging.Log.Notice("DungBeetle says: \"My job here is done\"")
}
