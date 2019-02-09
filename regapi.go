//////////////////////////////////////////////////////////////////////////
// DN42 Registry API Server
//////////////////////////////////////////////////////////////////////////

package main

//////////////////////////////////////////////////////////////////////////

import (
	"encoding/json"
	"github.com/gorilla/mux"
	log "github.com/sirupsen/logrus"
	"net/http"
	"strings"
	//	"time"
)

//////////////////////////////////////////////////////////////////////////
// register the api

func init() {
	RegisterAPIEndpoint(InitRegAPI)
}

//////////////////////////////////////////////////////////////////////////
// called from main to initialise the API routing

func InitRegAPI(router *mux.Router) {

	s := router.
		Methods("GET").
		PathPrefix("/registry").
		Subrouter()

	s.HandleFunc("/", regRootHandler)
	//s.HandleFunc("/.schema", rTypeListHandler)
	//s.HandleFunc("/.meta/", rTypeListHandler)

	s.HandleFunc("/{type}", regTypeHandler)
	s.HandleFunc("/{type}/{object}", regObjectHandler)

	log.Info("Registry API installed")
}

//////////////////////////////////////////////////////////////////////////
// handler utility funcs

func responseJSON(w http.ResponseWriter, v interface{}) {

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
	w.Write(data)
}

//////////////////////////////////////////////////////////////////////////
// root handler, lists all types within the registry

func regRootHandler(w http.ResponseWriter, r *http.Request) {

	response := make(map[string]int)
	for _, rType := range RegistryData.Types {
		response[rType.Ref] = len(rType.Objects)
	}
	responseJSON(w, response)

}

//////////////////////////////////////////////////////////////////////////
// type handler returns list of objects that match the type

func regTypeHandler(w http.ResponseWriter, r *http.Request) {

	// request parameters
	vars := mux.Vars(r)
	query := r.URL.Query()

	typeName := vars["type"] // type name to list
	match := query["match"]  // single query or match

	// special case to return all types
	all := false
	if typeName == "*" {
		match = []string{}
		all = true
	}

	// results will hold the types to return
	var results []*RegType

	//	check match type
	if match == nil {
		// exact match

		// check the type object exists
		rType := RegistryData.Types[typeName]
		if rType == nil {
			http.Error(w, "No types matching '"+typeName+"' found", http.StatusNotFound)
			return
		}

		// return just a single result
		results = []*RegType{rType}

	} else {
		// substring match

		// comparisons are lower case
		typeName = strings.ToLower(typeName)

		// walk through the types and filter to the results list
		results = make([]*RegType, 0)
		for key, rType := range RegistryData.Types {
			if all || strings.Contains(strings.ToLower(key), typeName) {
				// match found, add to the list
				results = append(results, rType)
			}
		}

	}

	// construct the response
	response := make(map[string][]string)
	for _, rType := range results {

		objects := make([]string, 0, len(rType.Objects))
		for key := range rType.Objects {
			objects = append(objects, key)
		}

		response[rType.Ref] = objects
	}

	responseJSON(w, response)
}

//////////////////////////////////////////////////////////////////////////
// object handler returns object data

// per object response structure
type RegObjectResponse struct {
	Attributes [][2]string
	Backlinks  []string
}

func regObjectHandler(w http.ResponseWriter, r *http.Request) {

	// request parameters
	vars := mux.Vars(r)
	query := r.URL.Query()

	typeName := vars["type"]  // object type
	objName := vars["object"] // object name or match
	match := query["match"]   // single query or match
	raw := query["raw"]       // raw or decorated results

	// special case to return all objects
	all := false
	if objName == "*" {
		match = []string{}
		all = true
	}

	// verify the type exists
	rType := RegistryData.Types[typeName]
	if rType == nil {
		http.Error(w, "No types matching '"+typeName+"' found",
			http.StatusNotFound)
		return
	}

	// results will hold the objects to return
	var results []*RegObject

	// check match type
	if match == nil {
		// exact match

		// check the object exists
		object := rType.Objects[objName]
		if object == nil {
			http.Error(w, "No objects matching '"+objName+"' found",
				http.StatusNotFound)
			return
		}

		// then just create a results list with one object
		results = []*RegObject{object}

	} else {
		// substring matching

		// comparisons are lower case
		objName = strings.ToLower(objName)

		// walk through the type objects and filter to the results list
		results = make([]*RegObject, 0)
		for key, object := range rType.Objects {
			if all || strings.Contains(strings.ToLower(key), objName) {
				// match found, add to the list
				results = append(results, object)
			}
		}
	}

	// collate the results in to the response data
	if raw == nil {
		// provide a decorated response
		response := make(map[string]RegObjectResponse)

		// for each object in the results
		for _, object := range results {

			// copy the raw attributes
			attributes := make([][2]string, len(object.Data))
			for ix, attribute := range object.Data {
				attributes[ix] = [2]string{attribute.Key, attribute.Value}
			}

			// construct the backlinks
			backlinks := make([]string, len(object.Backlinks))
			for ix, object := range object.Backlinks {
				backlinks[ix] = object.Ref
			}

			// add to the response
			response[object.Ref] = RegObjectResponse{
				Attributes: attributes,
				Backlinks:  backlinks,
			}
		}

		responseJSON(w, response)

	} else {
		// provide a response with just the raw registry data
		response := make(map[string][][2]string)

		// for each object in the results
		for _, object := range results {

			attributes := make([][2]string, len(object.Data))
			response[object.Ref] = attributes

			// copy the raw attributes
			for ix, attribute := range object.Data {
				attributes[ix] = [2]string{attribute.Key, attribute.RawValue}
			}
		}

		responseJSON(w, response)
	}

}

//////////////////////////////////////////////////////////////////////////
// end of code
