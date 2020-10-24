//////////////////////////////////////////////////////////////////////////
// DN42 Registry API Server
//////////////////////////////////////////////////////////////////////////

package main

//////////////////////////////////////////////////////////////////////////

import (
	"github.com/gorilla/mux"
	log "github.com/sirupsen/logrus"
	"net/http"
	"os"
)

//////////////////////////////////////////////////////////////////////////
// called from main to initialise the API routing

func InstallStaticRoutes(router *mux.Router, staticPath string) {

	// an empty path disables static route serving
	if staticPath == "" {
		log.Info("Disabling static route serving")
		return
	}

	// validate that the staticPath exists
	stat, err := os.Stat(staticPath)
	if err != nil {
		log.WithFields(log.Fields{
			"error": err,
			"path":  staticPath,
		}).Fatal("Unable to find static page directory")
	}

	// and it is a directory
	if !stat.IsDir() {
		log.WithFields(log.Fields{
			"error": err,
			"path":  staticPath,
		}).Fatal("Static path is not a directory")
	}

	// install a file server for the static route
	router.PathPrefix("/").Handler(staticHandler(staticPath))

	log.WithFields(log.Fields{
		"path": staticPath,
	}).Info("Static route installed")

}

//////////////////////////////////////////////////////////////////////////

func staticHandler(path string) http.Handler {

	server := http.FileServer(http.Dir(path))

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// allow up to a month of caching
		w.Header().Set("Cache-Control", "public, max-age=2592000, stale-if-error=86400")
		server.ServeHTTP(w, r)
	})
}

//////////////////////////////////////////////////////////////////////////
// end of code
