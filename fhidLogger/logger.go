package fhidLogger

import (
	"os"

	"github.com/inconshreveable/log15"
)

// Loggo is the global logger
var Loggo log15.Logger

// SetLogger sets up logging globally for the packages involved
// in the fhid runtime.
func SetLogger(daemonFlag, noLogFile bool, logFileS, loglevel string) {
	Loggo = log15.New()
	if noLogFile && loglevel == "debug" {
		Loggo.SetHandler(
			log15.LvlFilterHandler(
				log15.LvlDebug,
				log15.StreamHandler(os.Stdout, log15.LogfmtFormat())))
	} else if noLogFile {
		Loggo.SetHandler(
			log15.LvlFilterHandler(
				log15.LvlInfo,
				log15.StreamHandler(os.Stdout, log15.LogfmtFormat())))
	} else if daemonFlag && loglevel == "debug" {
		Loggo.SetHandler(
			log15.LvlFilterHandler(
				log15.LvlDebug,
				log15.Must.FileHandler(logFileS, log15.JsonFormat())))
	} else if daemonFlag {
		Loggo.SetHandler(
			log15.LvlFilterHandler(
				log15.LvlInfo,
				log15.Must.FileHandler(logFileS, log15.JsonFormat())))
	} else if loglevel == "debug" {
		// log to stdout and file
		Loggo.SetHandler(log15.MultiHandler(
			log15.StreamHandler(os.Stdout, log15.LogfmtFormat()),
			log15.LvlFilterHandler(
				log15.LvlDebug,
				log15.Must.FileHandler(logFileS, log15.JsonFormat()))))
	} else {
		// log to stdout and file
		Loggo.SetHandler(log15.MultiHandler(
			log15.LvlFilterHandler(
				log15.LvlInfo,
				log15.StreamHandler(os.Stdout, log15.LogfmtFormat())),
			log15.LvlFilterHandler(
				log15.LvlInfo,
				log15.Must.FileHandler(logFileS, log15.JsonFormat()))))
	}
}
