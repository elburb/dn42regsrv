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
	Number  uint
	Action  string
	Prefix  string
	MinLen  uint8
	MaxLen  uint8
	Network *net.IPNet
}

type ROA struct {
	CTime   time.Time
	Commit  string
	Filters []*ROAFilter
	IPv4    []*PrefixROA
	IPv6    []*PrefixROA
}

var ROAData *ROA

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

	s.HandleFunc("/json", roaJSONHandler)
	s.HandleFunc("/bird/{birdv}/{ipv}", roaBirdHandler)

	log.Info("ROA API installed")
}

//////////////////////////////////////////////////////////////////////////
// api handlers

// return JSON formatted ROA data suitable for use with GoRTR
func roaJSONHandler(w http.ResponseWriter, r *http.Request) {
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
	if roa.loadFilter(path+"/filter.txt") != nil {
		// error loading IPv4 filter, don't update
		return
	}

	if roa.loadFilter(path+"/filter6.txt") != nil {
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
			Valid:     utime + (12 * 3600), // valid for 12 hours
		},
	}

	response.Roas = append(roa.IPv4, roa.IPv6...)
	response.MetaData.Counts = uint(len(response.Roas))

	ROAJSONResponse = response
}

//////////////////////////////////////////////////////////////////////////
// load network filter definitions from a filter file

func (roa *ROA) loadFilter(path string) error {

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

	// sort the filters based on prefix length (largest first)
	sort.Slice(filters, func(i, j int) bool {
		leni, _ := filters[i].Network.Mask.Size()
		lenj, _ := filters[j].Network.Mask.Size()
		return leni > lenj
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
		_, pnet, err := net.ParseCIDR(prefix)
		if err != nil {
			log.WithFields(log.Fields{
				"object": object.Ref,
				"prefix": prefix,
				"error":  err,
			}).Error("Unable to parse CIDR in ROA")
			continue
		}

		// match the prefix to the prefix filters
		filter := roa.MatchFilter(pnet.IP)
		if filter == nil {
			continue
		}

		if filter.Action == "deny" {
			log.WithFields(log.Fields{
				"object": object.Ref,
				"prefix": prefix,
				"filter": filter.Prefix,
			}).Warn("Denied ROA through filter rule")
			continue
		}

		// calculate the max-length for this object

		mlen := filter.MaxLen

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
					Prefix: prefix,
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
