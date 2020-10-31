//////////////////////////////////////////////////////////////////////////
// DN42 Registry API Server
//////////////////////////////////////////////////////////////////////////

package main

//////////////////////////////////////////////////////////////////////////

import (
	"fmt"
	"github.com/gorilla/mux"
	log "github.com/sirupsen/logrus"
	//	"math/big"
	"bufio"
	"net"
	"net/http"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"
)

//////////////////////////////////////////////////////////////////////////
// register the api

func init() {
	EventBus.Listen("APIEndpoint", InitROAAPI)
	EventBus.Listen("RegistryUpdate", ROAUpdate)
}

//////////////////////////////////////////////////////////////////////////
// data model

type PrefixROA struct {
	Prefix string `json:"prefix"`
	MaxLen uint8  `json:"maxLength"`
	ASN    string `json:"asn"`
}

type ROAFilter struct {
	Number  uint       `json:"nr"`
	Action  string     `json:"action"`
	Prefix  string     `json:"prefix"`
	MinLen  uint8      `json:"minlen"`
	MaxLen  uint8      `json:"maxlen"`
	Network *net.IPNet `json:"-"`
	IPType  uint8      `json:"-"`
}

type ROA struct {
	CTime   time.Time
	Commit  string
	Filters []*ROAFilter
	IPv4    []*PrefixROA
	IPv6    []*PrefixROA
}

var ROAData *ROA

// set validity period for one week
// this might appear to be a long time, but is intended to provide
// enough time to prevent expiry of the data between real registry
// updates (which may only happen infrequently)
const ROA_JSON_VALIDITY_PERIOD = (7 * 24)

type ROAMetaData struct {
	Counts        uint   `json:"counts"`
	Generated     uint32 `json:"generated"`
	Valid         uint32 `json:"valid"`
	Signature     string `json:"signature,omitempty"`
	SignatureDate string `json:"signatureDate,omitempty"`
}

type ROAJSON struct {
	MetaData ROAMetaData  `json:"metadata"`
	Roas     []*PrefixROA `json:"roas"`
}

var ROAJSONResponse *ROAJSON

//////////////////////////////////////////////////////////////////////////
// called from main to initialise the API routing

func InitROAAPI(params ...interface{}) {

	router := params[0].(*mux.Router)

	s := router.
		Methods("GET").
		PathPrefix("/roa").
		Subrouter()

	s.HandleFunc("/filter/{ipv}", roaFilterHandler)
	s.HandleFunc("/json", roaJSONHandler)
	s.HandleFunc("/bird/{birdv}/{ipv}", roaBirdHandler)

	log.Info("ROA API installed")
}

//////////////////////////////////////////////////////////////////////////
// api handlers

// return JSON formatted version of filter{,6}.txt
func roaFilterHandler(w http.ResponseWriter, r *http.Request) {

	vars := mux.Vars(r)
	ipv := vars["ipv"]

	// pre-create an array to hold the result
	filters := make([]*ROAFilter, 0, len(ROAData.Filters))

	// helper closure to select from the filter array
	fselect := func(a []*ROAFilter, t uint8) []*ROAFilter {
		for _, f := range ROAData.Filters {
			if f.IPType == t {
				a = append(a, f)
			}
		}
		return a
	}

	// add ipv4 filters if required
	if strings.ContainsRune(ipv, '4') {
		filters = fselect(filters, 4)
	}

	// add ipv6 filters if required
	if strings.ContainsRune(ipv, '6') {
		filters = fselect(filters, 6)
	}

	// cache for up to a week, but set etag to commit to catch changes
	w.Header().Set("Cache-Control", "public, max-age=7200, stale-if-error=604800")
	w.Header().Set("ETag", ROAData.Commit)

	ResponseJSON(w, filters)
}

// return JSON formatted ROA data suitable for use with GoRTR
func roaJSONHandler(w http.ResponseWriter, r *http.Request) {

	// check validity period of returned data
	tnow := uint32(time.Now().Unix())
	valid := ROAJSONResponse.MetaData.Valid

	// check if validity period is close to expiry
	if (tnow > valid) ||
		((valid - tnow) < (ROA_JSON_VALIDITY_PERIOD / 4)) {
		// if so extend the validity period
		ROAJSONResponse.MetaData.Valid += (ROA_JSON_VALIDITY_PERIOD * 3600)
	}

	// cache for up to a week, but set etag to commit to catch changes
	w.Header().Set("Cache-Control", "public, max-age=7200, stale-if-error=604800")
	w.Header().Set("ETag", ROAData.Commit)

	ResponseJSON(w, ROAJSONResponse)
}

// return the roa in bird format
func roaBirdHandler(w http.ResponseWriter, r *http.Request) {

	vars := mux.Vars(r)
	birdv := vars["birdv"]
	ipv := vars["ipv"]

	// bird 1 or bird 2 format
	birdf := "roa %s max %d as %s;\n"
	if birdv == "2" {
		birdf = "route %s max %d as %s;\n"
	}

	var roa []*PrefixROA
	if strings.ContainsRune(ipv, '4') {
		roa = append(roa, ROAData.IPv4...)
	}
	if strings.ContainsRune(ipv, '6') {
		roa = append(roa, ROAData.IPv6...)
	}

	w.Header().Set("Content-Type", "text/plain")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	// cache for up to a week, but set etag to commit to catch changes
	w.Header().Set("Cache-Control", "public, max-age=7200, stale-if-error=604800")
	w.Header().Set("ETag", ROAData.Commit)

	fmt.Fprintf(w, "#\n# dn42regsrv ROA Generator\n# Last Updated: %s\n"+
		"# Commit: %s\n#\n", ROAData.CTime.String(), ROAData.Commit)

	for _, r := range roa {
		fmt.Fprintf(w, birdf, r.Prefix, r.MaxLen, r.ASN[2:])
	}

}

//////////////////////////////////////////////////////////////////////////
// called whenever the registry is updated

func ROAUpdate(params ...interface{}) {

	registry := params[0].(*Registry)
	path := params[1].(string)

	// initiate new ROA data
	roa := &ROA{
		CTime:  time.Now(),
		Commit: registry.Commit,
	}

	// load filter{,6}.txt files
	if roa.loadFilter(path+"/filter.txt", 4) != nil {
		// error loading IPv4 filter, don't update
		return
	}

	if roa.loadFilter(path+"/filter6.txt", 6) != nil {
		// error loading IPv6 filter, don't update
		return
	}

	// compile ROA prefixes
	roa.IPv4 = roa.CompileROA(registry, "route")
	roa.IPv6 = roa.CompileROA(registry, "route6")

	// swap in the new data
	ROAData = roa

	log.WithFields(log.Fields{
		"ipv4": len(roa.IPv4),
		"ipv6": len(roa.IPv6),
	}).Debug("ROA data updated")

	// pre-compute the JSON return struct

	utime := uint32(roa.CTime.Unix())

	response := &ROAJSON{
		MetaData: ROAMetaData{
			Generated: utime,
			Valid:     utime + (ROA_JSON_VALIDITY_PERIOD * 3600),
		},
	}

	response.Roas = append(roa.IPv4, roa.IPv6...)
	response.MetaData.Counts = uint(len(response.Roas))

	ROAJSONResponse = response
}

//////////////////////////////////////////////////////////////////////////
// load network filter definitions from a filter file

func (roa *ROA) loadFilter(path string, iptype uint8) error {

	// open the file for reading
	file, err := os.Open(path)
	if err != nil {
		log.WithFields(log.Fields{
			"path":  path,
			"error": err,
		}).Error("Unable to open filter file")
		return err
	}
	defer file.Close()

	// helper closure to convert strings to numbers
	var cerr error
	convert := func(s string) int {
		if cerr != nil {
			return 0
		}
		val, cerr := strconv.Atoi(s)
		if cerr != nil {
			log.WithFields(log.Fields{
				"number": s,
				"error":  err,
			}).Error("Unable to parse number in filter file")
			return 0
		}
		return val
	}

	filters := make([]*ROAFilter, 0)

	// read the file line by line
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {

		line := strings.TrimSpace(scanner.Text())

		// remove any comments
		if ix := strings.IndexRune(line, '#'); ix != -1 {
			line = line[:ix]
		}

		fields := strings.Fields(line)
		if len(fields) >= 5 {

			// parse the prefix in to a NetIP structure
			prefix := fields[2]
			_, network, err := net.ParseCIDR(prefix)
			if err != nil {
				log.WithFields(log.Fields{
					"path":   path,
					"prefix": prefix,
					"error":  err,
				}).Error("Unable to parse CIDR in filter file")

			} else {

				// construct the filter object
				roaf := &ROAFilter{
					Number:  uint(convert(fields[0])),
					Action:  fields[1],
					Prefix:  prefix,
					MinLen:  uint8(convert(fields[3])),
					MaxLen:  uint8(convert(fields[4])),
					Network: network,
					IPType:  iptype,
				}

				// add to list if no strconv error
				if cerr == nil {
					filters = append(filters, roaf)
				}
			}
		}
	}

	// did something go wrong ?
	if err := scanner.Err(); err != nil {
		log.WithFields(log.Fields{
			"path":  path,
			"error": err,
		}).Error("Scanner error reading filter file")
		return err
	}

	// filter.txt should be in order,
	// but still sort by number just in case
	sort.Slice(filters, func(i, j int) bool {
		return filters[i].Number < filters[j].Number
	})

	// add to the roa object
	roa.Filters = append(roa.Filters, filters...)
	return nil
}

//////////////////////////////////////////////////////////////////////////
// return the filter object that matches an IP address

func (roa *ROA) MatchFilter(ip net.IP) *ROAFilter {
	for _, filter := range roa.Filters {
		if filter.Network.Contains(ip) {
			return filter
		}
	}

	log.WithFields(log.Fields{
		"IP": ip,
	}).Error("Couldn't match address to filter !")

	return nil
}

//////////////////////////////////////////////////////////////////////////
// compile ROA data

func (roa *ROA) CompileROA(registry *Registry,
	tname string) []*PrefixROA {

	// prepare indices to the route object keys
	stype := registry.Schema[tname]
	routeIX := stype.KeyIndex[tname]
	originIX := stype.KeyIndex["origin"]
	mlenIX := stype.KeyIndex["max-length"]

	roalist := make([]*PrefixROA, 0, len(routeIX.Objects))

	// for each object that has a route key
	for object, rattribs := range routeIX.Objects {

		if len(rattribs) > 1 {
			log.WithFields(log.Fields{
				"object": object.Ref,
			}).Warn("Found object with multiple route attributes")
		}

		// extract the prefix
		prefix := rattribs[0].RawValue
		prefIP, prefNet, err := net.ParseCIDR(prefix)
		if err != nil {
			log.WithFields(log.Fields{
				"object": object.Ref,
				"prefix": prefix,
				"error":  err,
			}).Error("Unable to parse CIDR in ROA")
			continue
		}

		// check for CIDR errors
		if !prefIP.Equal(prefNet.IP) {
			log.WithFields(log.Fields{
				"prefix": prefix,
			}).Warn("Denied ROA: invalid CIDR")
			continue
		}

		// match the prefix to the prefix filters
		filter := roa.MatchFilter(prefNet.IP)
		if filter == nil {
			continue
		}

		// don't allow routes that are denied in the filter rules
		if filter.Action == "deny" {
			log.WithFields(log.Fields{
				"object": object.Ref,
				"prefix": prefix,
				"filter": filter.Prefix,
			}).Warn("Denied ROA: through filter rule")
			continue
		}

		mlen := filter.MaxLen
		prefLen, _ := prefNet.Mask.Size()

		// calculate the max-length for this object

		// check if the attribute has max-length defined
		mattrib := mlenIX.Objects[object]
		if mattrib != nil {

			// use the local max-length value
			tmp, err := strconv.ParseUint(mattrib[0].RawValue, 10, 8)
			if err != nil {
				log.WithFields(log.Fields{
					"object":     object.Ref,
					"max-length": mattrib[0].RawValue,
					"error":      err,
				}).Warn("Unable to convert max-length attribute")
			} else {

				// filter rules still have precedence over local values
				if (uint8(tmp) < mlen) && (uint8(tmp) > filter.MinLen) {
					mlen = uint8(tmp)
				}

			}
		}

		// if the prefix is greater than the max length
		// then don't emit an ROA route (making the route invalid)
		if prefLen > int(mlen) {
			log.WithFields(log.Fields{
				"object": object.Ref,
				"prefix": prefix,
				"maxlen": mlen,
			}).Warn("Denied ROA: Prefix > filter MaxLen")
			continue
		}

		// look up the origin key for this object
		oattribs := originIX.Objects[object]
		if oattribs == nil {
			log.WithFields(log.Fields{
				"object": object.Ref,
			}).Warn("Route Object without Origin")
		} else {

			// then for origin that can announce this prefix
			for _, oattrib := range oattribs {

				// add the ROA
				roalist = append(roalist, &PrefixROA{
					Prefix: prefNet.String(),
					MaxLen: mlen,
					ASN:    oattrib.RawValue,
				})

			}
		}
	}

	return roalist
}

//////////////////////////////////////////////////////////////////////////
// end of code
