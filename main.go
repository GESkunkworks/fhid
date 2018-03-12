package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"regexp"

	"github.build.ge.com/212601587/fhid/fhid"
	"github.build.ge.com/212601587/fhid/fhidConfig"
	"github.build.ge.com/212601587/fhid/fhidLogger"
)

var version string

func main() {
	// set up configuration
	var configFile string
	var logLevel string
	var logFile string
	var versionOverride string
	var versionDefault string
	var versionMajMin string
	var versionFlag bool
	var daemonFlag bool
	var noLogFile bool
	versionDefault = "v1.0"
	flag.StringVar(&configFile, "c", "./config.json", "Path to config file.")
	flag.StringVar(&logFile, "logfile", "fhid.log.json", "JSON logfile location")
	flag.StringVar(&logLevel, "loglevel", "info", "Log level (info or debug)")
	flag.StringVar(&versionOverride, "vlo", "", "If you wish to override the version listener reported during runtime. Affects the API route handlers (e.g., '/v2.3/healthcheck' in the format of vN.N where N is an int")
	flag.BoolVar(&versionFlag, "version", false, "print version and exit")
	flag.BoolVar(&daemonFlag, "daemon", false, "run as daemon with no stdout")
	flag.BoolVar(&noLogFile, "nologfile", false, "Indicates whether or not to skip writing of a filesystem log file.")
	flag.Parse()

	if version == "" {
		version = versionDefault
	}
	// get a vX.X form of the version no matter what happens
	if versionOverride != "" {
		versionMajMin = versionSplitter(versionOverride, versionDefault)
	} else {
		versionMajMin = versionSplitter(version, versionDefault)
	}

	if versionFlag {
		log.Printf("fhid %s\n", version)
		log.Printf("major/minor handler version = %s\n", versionMajMin)
		os.Exit(0)
	}
	// if daemon just log to file
	fhidLogger.SetLogger(daemonFlag, noLogFile, logFile, logLevel)

	if versionFlag {
		fmt.Printf("fhid %s\n", version)
		os.Exit(0)
	}
	if configFile == "" {
		fhidLogger.Loggo.Crit("Please specify config file with -c flag")
		os.Exit(1)
	}
	err := fhidConfig.SetConfig(configFile)
	fhidConfig.Version = version
	fhidLogger.Loggo.Info("fhid: Fixham Harbour Image Database", "version", version)
	fhidLogger.Loggo.Info("Set fhidConfig.Version", "Version", fhidConfig.Version)
	if err != nil {
		fhidLogger.Loggo.Error("Error loading config file, check formatting.", "filename", configFile, "Error", err)
		os.Exit(1)
	}
	fhidLogger.Loggo.Info("Loaded config", "Config", fhidConfig.Config.ShowConfig())
	err = fhid.SetupConnection()
	if err != nil {
		fhidLogger.Loggo.Error("Error in Redis test connection", "Error", err)
		fhid.TeardownConnection()
		os.Exit(1)
	} else {
		fhidLogger.Loggo.Info("Successfully connected to Redis")
	}
	http.HandleFunc(fmt.Sprintf("/%s/images", versionMajMin), fhid.HandlerImages)
	http.HandleFunc(fmt.Sprintf("/%s/query", versionMajMin), fhid.HandlerImagesQuery)

	routeHealthcheckVersioned := fmt.Sprintf("/%s/healthcheck", versionMajMin)
	http.HandleFunc(routeHealthcheckVersioned, fhid.HealthCheck)
	fhidLogger.Loggo.Info("Listening on versioned healthcheck endpoint", "Endpoint", routeHealthcheckVersioned)

	http.HandleFunc("/healthcheck", fhid.HealthCheck)
	listenString := fhidConfig.Config.ListenHost + ":" + fhidConfig.Config.ListenPort

	http.ListenAndServe(listenString, nil)
	fhidLogger.Loggo.Info("Listening on host", "Host", listenString)

}

func versionSplitter(fullver string, override string) (verMinMaj string) {
	r := regexp.MustCompile(`(v[0-9]*\d\.[0-9]*)`)
	if len(fullver) < 3 {
		return override
	} else {
		s := r.FindString(fullver)
		if len(s) < 4 {
			return override
		} else {
			return s
		}
	}
}
