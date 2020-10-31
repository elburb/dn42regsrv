//////////////////////////////////////////////////////////////////////////
// DN42 Registry API Server
//////////////////////////////////////////////////////////////////////////

package main

//////////////////////////////////////////////////////////////////////////

import (
	//	"fmt"
	"github.com/gorilla/mux"
	log "github.com/sirupsen/logrus"
	"net/http"
	"strings"
	//	"time"
)

//////////////////////////////////////////////////////////////////////////
// data structures

type RegMetaReturn struct {
	Commit string
}

//////////////////////////////////////////////////////////////////////////
// register the api

func init() {
	EventBus.Listen("APIEndpoint", InitRegistryAPI)
}

//////////////////////////////////////////////////////////////////////////
// called from main to initialise the API routing

func InitRegistryAPI(params ...interface{}) {

	router := params[0].(*mux.Router)

	s := router.
		Methods("GET").
		PathPrefix("/registry").
		Subrouter()

	s.HandleFunc("/", regRootHandler)
	//s.HandleFunc("/.schema", rTypeListHandler)
	//s.HandleFunc("/.meta/", rTypeListHandler)

	s.HandleFunc("/.meta", regMetaHandler)
	s.HandleFunc("/{type}", regTypeHandler)
	s.HandleFunc("/{type}/{object}", regObjectHandler)
	s.HandleFunc("/{type}/{object}/{key}", regKeyHandler)
	s.HandleFunc("/{type}/{object}/{key}/{attribute}", regAttributeHandler)

	log.Info("Registry API installed")
}

//////////////////////////////////////////////////////////////////////////
// return registry metadata

func regMetaHandler(w http.ResponseWriter, r *http.Request) {

	rv := RegMetaReturn{
		Commit: RegistryData.Commit,
	}

	// don't cache
	w.Header().Set("Cache-Control", "no-store")
	ResponseJSON(w, rv)
}

//////////////////////////////////////////////////////////////////////////
// filter functions

// return a list of types that match the filter
func filterTypes(filter string) []*RegType {

	var rtypes []*RegType = nil

	// check if filter starts with '*'
	if filter[0] == '*' {
		// try and match the filter against all reg types

		filter = strings.ToLower(filter[1:])

		// special case, if the filter was '*' return all types
		if len(filter) == 0 {

			rtypes = make([]*RegType, 0, len(RegistryData.Types))
			for _, rtype := range RegistryData.Types {
				rtypes = append(rtypes, rtype)
			}

		} else {

			// otherwise substring match the types
			for _, rtype := range RegistryData.Types {
				lname := strings.ToLower(rtype.Ref)
				if strings.Contains(lname, filter) {
					// matched, add it to the list
					rtypes = append(rtypes, rtype)
				}
			}

		}

	} else {
		// perform an exact match with one entry

		rtype := RegistryData.Types[filter]
		if rtype != nil {
			// return a single answer
			rtypes = []*RegType{rtype}
		}

	}

	return rtypes
}

// return a list of objects from a set of types that match a filter
func filterObjects(rtypes []*RegType, filter string) []*RegObject {

	var objects []*RegObject = nil

	// check if filter starts with '*'
	if filter[0] == '*' {
		// try and match objects against the filter

		filter = strings.ToLower(filter[1:])

		// for each type
		for _, rtype := range rtypes {

			// special case, if the filter was '*' return all objects
			if len(filter) == 0 {

				objs := make([]*RegObject, 0, len(rtype.Objects))
				for _, object := range rtype.Objects {
					objs = append(objs, object)
				}
				objects = append(objects, objs...)

			} else {
				// otherwise substring match the object names

				for _, object := range rtype.Objects {
					lname := strings.ToLower(object.Ref)
					if strings.Contains(lname, filter) {
						// matched, add it to the list
						objects = append(objects, object)
					}
				}

			}

		}

	} else {
		// perform an exact match against one object for each type

		for _, rtype := range rtypes {

			object := rtype.Objects[filter]
			if object != nil {
				// add the object
				objects = append(objects, object)
			}
		}

	}

	return objects
}

// return a list of key indices matching the filter
func filterKeys(rtypes []*RegType, filter string) []*RegKeyIndex {

	var ix []*RegKeyIndex = nil

	// check if filter starts with '*'
	if filter[0] == '*' {
		// try and match keys against the filter

		filter = strings.ToLower(filter[1:])

		// for each type
		for _, rtype := range rtypes {
			ref := rtype.Ref
			schema := RegistryData.Schema[ref]

			// special case, if the filter was '*' return all indices
			if len(filter) == 0 {

				tmp := make([]*RegKeyIndex, 0, len(schema.KeyIndex))
				for _, keyix := range schema.KeyIndex {
					tmp = append(tmp, keyix)
				}
				ix = append(ix, tmp...)

			} else {
				// otherwise substring match the key names

				for kname, keyix := range schema.KeyIndex {
					kname = strings.ToLower(kname)
					if strings.Contains(kname, filter) {
						ix = append(ix, keyix)
					}
				}

			}
		}

	} else {
		// perform an exact match, one key for each type

		for _, rtype := range rtypes {
			ref := rtype.Ref
			schema := RegistryData.Schema[ref]
			keyix := schema.KeyIndex[filter]
			if keyix != nil {
				// add the index
				ix = append(ix, keyix)
			}
		}

	}

	return ix
}

// helper func to determine if an attribute matches a filter
func matchAttribute(attribute *RegAttribute,
	filter string, isExact bool) bool {

	if isExact {

		return filter == attribute.RawValue

	} else {

		l := strings.ToLower(attribute.RawValue)
		return strings.Contains(l, filter)

	}
}

// return a map of objects and attribute values that match the filter
func filterAttributes(ix []*RegKeyIndex, objects []*RegObject,
	filter string, raw bool) map[string]map[string][]string {

	result := make(map[string]map[string][]string)

	// pre-calculate the search type
	isExact := true
	isAll := false

	if filter[0] == '*' {
		isExact = false
		filter = strings.ToLower(filter[1:])
		if len(filter) == 0 {
			isAll = true
		}
	}

	// for each key index
	for _, keyix := range ix {

		// for each object
		for _, object := range objects {

			// attributes in this object that match this key
			attributes := keyix.Objects[object]
			if attributes != nil {
				// this object has at least one relevant key

				// match the attributes
				for _, attribute := range attributes {
					if isAll || matchAttribute(attribute, filter, isExact) {
						// match found !

						objmap := result[object.Ref]
						if objmap == nil {
							objmap = make(map[string][]string)
							result[object.Ref] = objmap
						}

						// append the result
						var value *string
						if raw {
							value = &attribute.RawValue
						} else {
							value = &attribute.Value
						}

						objmap[keyix.Ref] = append(objmap[keyix.Ref], *value)
					}
				}
			}

		}

	}

	return result
}

//////////////////////////////////////////////////////////////////////////
// root handler, lists all types within the registry

func regRootHandler(w http.ResponseWriter, r *http.Request) {

	response := make(map[string]int)
	for _, rType := range RegistryData.Types {
		response[rType.Ref] = len(rType.Objects)
	}

	// cache for up to a day, but set etag to commit to catch changes
	w.Header().Set("Cache-Control", "public, max-age=7200, stale-if-error=86400")
	w.Header().Set("ETag", RegistryData.Commit)

	ResponseJSON(w, response)

}

//////////////////////////////////////////////////////////////////////////
// type handler returns list of objects that match the type

func regTypeHandler(w http.ResponseWriter, r *http.Request) {

	// request parameters
	vars := mux.Vars(r)
	tFilter := vars["type"] // type filter

	// match registry types against the filter
	rtypes := filterTypes(tFilter)
	if rtypes == nil {
		http.Error(w, "No objects matching '"+tFilter+"' found",
			http.StatusNotFound)
		return
	}

	// construct the response
	response := make(map[string][]string)
	for _, rtype := range rtypes {

		objects := make([]string, 0, len(rtype.Objects))
		for key := range rtype.Objects {
			objects = append(objects, key)
		}

		response[rtype.Ref] = objects
	}

	// cache for up to a day, but set etag to commit to catch changes
	w.Header().Set("Cache-Control", "public, max-age=7200, stale-if-error=86400")
	w.Header().Set("ETag", RegistryData.Commit)

	ResponseJSON(w, response)
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

	tFilter := vars["type"]   // type filter
	oFilter := vars["object"] // object filter
	raw := query["raw"]       // raw or decorated results

	// select the type(s)
	rtypes := filterTypes(tFilter)
	if rtypes == nil {
		http.Error(w, "No objects matching '"+tFilter+"' found",
			http.StatusNotFound)
		return
	}

	// then select the objects
	objects := filterObjects(rtypes, oFilter)
	if objects == nil {
		http.Error(w, "No objects matching '"+tFilter+
			"/"+oFilter+"' found", http.StatusNotFound)
		return
	}

	// collate the results in to the response data
	if raw == nil {
		// provide a decorated response
		response := make(map[string]RegObjectResponse)

		// for each object in the results
		for _, object := range objects {

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

		// cache for up to a day, but set etag to commit to catch changes
		w.Header().Set("Cache-Control", "public, max-age=7200, stale-if-error=86400")
		w.Header().Set("ETag", RegistryData.Commit)

		ResponseJSON(w, response)

	} else {
		// provide a response with just the raw registry data
		response := make(map[string][][2]string)

		// for each object in the results
		for _, object := range objects {

			attributes := make([][2]string, len(object.Data))
			response[object.Ref] = attributes

			// copy the raw attributes
			for ix, attribute := range object.Data {
				attributes[ix] = [2]string{attribute.Key, attribute.RawValue}
			}
		}

		// cache for up to a day, but set etag to commit to catch changes
		w.Header().Set("Cache-Control", "public, max-age=7200 stale-if-error=86400")
		w.Header().Set("ETag", RegistryData.Commit)

		ResponseJSON(w, response)
	}

}

//////////////////////////////////////////////////////////////////////////
// key handler returns attribute data matching the key

func regKeyHandler(w http.ResponseWriter, r *http.Request) {

	// request parameters
	vars := mux.Vars(r)
	query := r.URL.Query()

	tFilter := vars["type"]   // type filter
	oFilter := vars["object"] // object filter
	kFilter := vars["key"]    // key filter
	raw := query["raw"]       // raw or decorated results

	// select the type(s)
	rtypes := filterTypes(tFilter)
	if rtypes == nil {
		http.Error(w, "No objects matching '"+tFilter+"' found",
			http.StatusNotFound)
		return
	}

	// select the key indices
	ix := filterKeys(rtypes, kFilter)
	if rtypes == nil {
		http.Error(w, "No objects matching '"+tFilter+"/*/"+
			kFilter+"' found", http.StatusNotFound)
		return
	}

	// select the objects
	objects := filterObjects(rtypes, oFilter)
	if objects == nil {
		http.Error(w, "No objects matching '"+tFilter+
			"/"+oFilter+"' found", http.StatusNotFound)
		return
	}

	// select objects that match the keys
	amap := filterAttributes(ix, objects, "*", (raw != nil))
	if len(amap) == 0 {
		http.Error(w, "No attributes matching '"+tFilter+"/"+
			oFilter+"/"+kFilter+"' found", http.StatusNotFound)
		return
	}

	// cache for up to a day, but set etag to commit to catch changes
	w.Header().Set("Cache-Control", "public, max-age=7200, stale-if-error=86400")
	w.Header().Set("ETag", RegistryData.Commit)

	ResponseJSON(w, amap)
}

//////////////////////////////////////////////////////////////////////////
// attribute handler returns attribute data matching the attribute

func regAttributeHandler(w http.ResponseWriter, r *http.Request) {

	// request parameters
	vars := mux.Vars(r)
	query := r.URL.Query()

	tFilter := vars["type"]      // type filter
	oFilter := vars["object"]    // object filter
	kFilter := vars["key"]       // key filter
	aFilter := vars["attribute"] // attribute filter
	raw := query["raw"]          // raw or decorated results

	// select the type(s)
	rtypes := filterTypes(tFilter)
	if rtypes == nil {
		http.Error(w, "No objects matching '"+tFilter+"' found",
			http.StatusNotFound)
		return
	}

	// select the key indices
	ix := filterKeys(rtypes, kFilter)
	if rtypes == nil {
		http.Error(w, "No objects matching '"+tFilter+"/*/"+
			kFilter+"' found", http.StatusNotFound)
		return
	}

	// then select the objects
	objects := filterObjects(rtypes, oFilter)
	if objects == nil {
		http.Error(w, "No objects matching '"+tFilter+
			"/"+oFilter+"' found", http.StatusNotFound)
		return
	}

	// select objects that match the keys
	amap := filterAttributes(ix, objects, aFilter, (raw != nil))
	if len(amap) == 0 {
		http.Error(w, "No attributes matching '"+tFilter+"/"+
			oFilter+"/"+kFilter+"/"+aFilter+"' found", http.StatusNotFound)
		return
	}

	// cache for up to a day, but set etag to commit to catch changes
	w.Header().Set("Cache-Control", "public, max-age=7200, stale-if-error=86400")
	w.Header().Set("ETag", RegistryData.Commit)

	ResponseJSON(w, amap)

}

//////////////////////////////////////////////////////////////////////////
// end of code
