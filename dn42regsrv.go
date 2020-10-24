//////////////////////////////////////////////////////////////////////////
// DN42 Registry API Server
//////////////////////////////////////////////////////////////////////////

package main

//////////////////////////////////////////////////////////////////////////

import (
	"context"
	"encoding/json"
	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
	log "github.com/sirupsen/logrus"
	flag "github.com/spf13/pflag"
	"net/http"
	"os"
	"os/signal"
	"time"
)

//////////////////////////////////////////////////////////////////////////
// simple event bus

type NotifyFunc func(...interface{})
type SimpleEventBus map[string][]NotifyFunc

var EventBus = make(SimpleEventBus)

// add a listener to an event
func (bus SimpleEventBus) Listen(event string, nfunc NotifyFunc) {
	bus[event] = append(bus[event], nfunc)
}

// fire notifications for an event
func (bus SimpleEventBus) Fire(event string, params ...interface{}) {
	funcs := bus[event]
	if funcs != nil {
		for _, nfunc := range funcs {
			nfunc(params...)
		}
	}
}

//////////////////////////////////////////////////////////////////////////
// utility func for returning JSON from an API endpoint

func ResponseJSON(w http.ResponseWriter, v interface{}) {

	// for response time testing
	//time.Sleep(time.Second)

	// marshal the JSON string
	data, err := json.Marshal(v)
	if err != nil {
		log.WithFields(log.Fields{
			"error": err,
		}).Error("Failed to marshal JSON")
	}

	// write back to http handler
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Write(data)
}

//////////////////////////////////////////////////////////////////////////
// utility function to set the log level

func setLogLevel(levelStr string) {

	if level, err := log.ParseLevel(levelStr); err != nil {
		// failed to set the level

		// set a sensible default and, of course, log the error
		log.SetLevel(log.InfoLevel)
		log.WithFields(log.Fields{
			"loglevel": levelStr,
			"error":    err,
		}).Error("Failed to set requested log level")

	} else {

		// set the requested level
		log.SetLevel(level)

	}
}

//////////////////////////////////////////////////////////////////////////
// http request logger

func requestLogger(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		log.WithFields(log.Fields{
			"method": r.Method,
			"URL":    r.URL.String(),
			"Remote": r.RemoteAddr,
		}).Debug("HTTP Request")

		next.ServeHTTP(w, r)
	})
}

//////////////////////////////////////////////////////////////////////////
// everything starts here

func main() {

	// set a default log level, so that logging can be used immediately
	// the level will be overidden later on once the command line
	// options are loaded
	log.SetLevel(log.InfoLevel)
	log.Info("DN42 Registry API Server Starting")

	// declare cmd line options
	var (
		logLevel        = flag.StringP("LogLevel", "l", "Info", "Log level")
		regDir          = flag.StringP("RegDir", "d", "registry", "Registry data directory")
		bindAddress     = flag.StringP("BindAddress", "b", "[::]:8042", "Server bind address")
		staticRoot      = flag.StringP("StaticRoot", "s", "StaticRoot", "Static page directory")
		refreshInterval = flag.StringP("Refresh", "i", "60m", "Refresh interval")
		gitPath         = flag.StringP("GitPath", "g", "/usr/bin/git", "Path to git executable")
		autoPull        = flag.BoolP("AutoPull", "a", true, "Automatically pull the registry")
		branch          = flag.StringP("Branch", "p", "master", "git branch to pull")
	)
	flag.Parse()

	// now initialise logging properly based on the cmd line options
	setLogLevel(*logLevel)

	// parse the refreshInterval and start data collection
	interval, err := time.ParseDuration(*refreshInterval)
	if err != nil {
		log.WithFields(log.Fields{
			"error":    err,
			"interval": *refreshInterval,
		}).Fatal("Unable to parse registry refresh interval")
	}

	InitialiseRegistryData(*regDir, interval,
		*gitPath, *autoPull, *branch)

	// initialise router
	router := mux.NewRouter()
	// global handers, log all requests and allow compression
	router.Use(requestLogger)
	router.Use(handlers.CompressHandler)

	// add API routes
	subr := router.PathPrefix("/api").Subrouter()
	EventBus.Fire("APIEndpoint", subr)

	// initialise static routes
	InstallStaticRoutes(router, *staticRoot)

	// initialise http server
	server := &http.Server{
		Addr:         *bindAddress,
		WriteTimeout: time.Second * 15,
		ReadTimeout:  time.Second * 15,
		IdleTimeout:  time.Second * 60,
		Handler:      router,
	}

	// run the server in a non-blocking goroutine

	log.WithFields(log.Fields{
		"BindAddress": *bindAddress,
	}).Info("Starting server")

	go func() {
		if err := server.ListenAndServe(); err != nil {
			log.WithFields(log.Fields{
				"error":       err,
				"BindAddress": *bindAddress,
			}).Fatal("Unable to start server")
		}
	}()

	// graceful shutdown via SIGINT (^C)
	csig := make(chan os.Signal, 1)
	signal.Notify(csig, os.Interrupt)

	// and block
	<-csig

	log.Info("Server shutting down")

	// deadline for server to shutdown
	ctx, cancel := context.WithTimeout(context.Background(), 10)
	defer cancel()

	// shutdown the server
	server.Shutdown(ctx)

	// nothing left to do
	log.Info("Shutdown complete, all done")
	os.Exit(0)
}

//////////////////////////////////////////////////////////////////////////
// end of code
