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
	"net/http"
	"strings"
	"time"
)

//////////////////////////////////////////////////////////////////////////
// register the api

func init() {
	EventBus.Listen("APIEndpoint", InitDNSAPI)
	EventBus.Listen("RegistryUpdate", DNSUpdate)
}

//////////////////////////////////////////////////////////////////////////
// data model

// very simple DNS record data
type DNSRecord struct {
	Name    string
	Type    string
	Content string
	Comment string `json:",omitempty"`
}

type DNSZone struct {
	Records   []*DNSRecord
	Commit    string
	Generated time.Time
}

var DNSRootZone *DNSZone

//////////////////////////////////////////////////////////////////////////
// fixed set of authoritative zones

var DNSRootAuthZones = map[string]string{
	"dn42":                    "domain/dn42",
	"recursive-servers.dn42":  "domain/recursive-servers.dn42",
	"delegation-servers.dn42": "domain/delegation-servers.dn42",
	"d.f.ip6.arpa":            "inet6num/fd00::_8",
	"20.172.in-addr.arpa":     "inetnum/172.20.0.0_16",
	"21.172.in-addr.arpa":     "inetnum/172.21.0.0_16",
	"22.172.in-addr.arpa":     "inetnum/172.22.0.0_16",
	"23.172.in-addr.arpa":     "inetnum/172.23.0.0_16",
	"31.172.in-addr.arpa":     "inetnum/172.31.0.0_16",
	"10.in-addr.arpa":         "inetnum/10.0.0.0_8",
}

//////////////////////////////////////////////////////////////////////////
// called from main to initialise the API routing

func InitDNSAPI(params ...interface{}) {

	router := params[0].(*mux.Router)

	s := router.
		Methods("GET").
		PathPrefix("/dns").
		Subrouter()

	s.HandleFunc("/root-zone", dnsRZoneHandler)

	log.Info("DNS API installed")
}

//////////////////////////////////////////////////////////////////////////
// api handlers

// return records that should be included in a DN42 root zone
func dnsRZoneHandler(w http.ResponseWriter, r *http.Request) {

	var format []string
	query := r.URL.Query()
	format = query["format"]
	if format == nil || len(format) != 1 {
		format = []string{"json"}
	}

	// cache for up to a day
	w.Header().Set("Cache-Control", "public, max-age=7200, stale-if-error=86400")

	switch format[0] {
	case "bind":
		DNSRootZone.WriteBindFormat(w)

	case "json":
		ResponseJSON(w, DNSRootZone)

	default:
		ResponseJSON(w, DNSRootZone)
	}
}

//////////////////////////////////////////////////////////////////////////
// called whenever the registry is updated

func DNSUpdate(params ...interface{}) {

	registry := params[0].(*Registry)
	//	path := params[1].(string)

	zone := &DNSZone{
		Generated: time.Now(),
		Commit:    registry.Commit,
	}

	// add zones that are authoritative within DN42
	for name, object := range DNSRootAuthZones {
		zone.AddRecords(registry, name, object, "DN42 Authoritative Zone")
	}

	// search all domain objects and add stub records for each TLD
	rtype := registry.Types["domain"]
	for name, object := range rtype.Objects {
		// domain is a TLD if it doesn't contain a '.'
		if strings.IndexRune(name, '.') == -1 {
			// don't include zones which are authoritative within DN42
			if DNSRootAuthZones[name] == "" {
				zone.AddRecords(registry, name, object.Ref, "Forward Zone")
			}
		}
	}

	DNSRootZone = zone
}

//////////////////////////////////////////////////////////////////////////
// utility function to add a DNS record to a zone

func (zone *DNSZone) AddRecord(name string, t string,
	content string, comment string) {
	record := &DNSRecord{
		Name:    name,
		Type:    t,
		Content: content,
		Comment: comment,
	}
	zone.Records = append(zone.Records, record)
}

//////////////////////////////////////////////////////////////////////////
// add nserver and ds-rdata records from a registry object

func (zone *DNSZone) AddRecords(registry *Registry, name string,
	path string, comment string) {

	// use the registry metadata key index to find the appropriate values
	object := registry.GetObject(path)
	if object == nil {
		log.WithFields(log.Fields{
			"zone": name,
			"path": path,
		}).Error("DNS: unable to find object in registry")
		return
	}

	nserver := object.GetKey("nserver")
	for _, ns := range nserver {
		// check if stub record needs to be added
		fields := strings.Split(ns.RawValue, " ")
		if len(fields) == 2 {
			// add a record for the NS, together with a stub A or AAAA record

			var stubtype string
			if strings.IndexRune(fields[1], ':') == -1 {
				// no : so IPv4
				stubtype = "A"
			} else {
				// has : so IPv6
				stubtype = "AAAA"
			}

			zone.AddRecord(name, "NS", fields[0]+".", comment)
			zone.AddRecord(fields[0], stubtype, fields[1], comment)

		} else {
			// no, just add an NS record as it was presented
			zone.AddRecord(name, "NS", ns.RawValue+".", comment)
		}

	}

	dsrdata := object.GetKey("ds-rdata")
	for _, ds := range dsrdata {
		zone.AddRecord(name, "DS", ds.RawValue, comment)
	}

}

//////////////////////////////////////////////////////////////////////////
// Functions for outputting zone records in different formats

func (r *DNSRecord) ToBindString() string {
	var comment string
	if r.Comment == "" {
		comment = ""
	} else {
		comment = "\t; " + r.Comment
	}

	return fmt.Sprintf("%s\tIN\t%s\t%s%s",
		r.Name, r.Type, r.Content, comment,
	)
}

func (zone *DNSZone) WriteBindFormat(w http.ResponseWriter) {

	w.Header().Set("Content-Type", "text/plain")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	// provide a header
	fmt.Fprintf(w, ";; DN42 Root Zone Records\n"+
		";; Commit Reference: %s\n;; Generated: %s\n",
		zone.Commit, zone.Generated)

	// then simply output each record in turn
	for _, record := range zone.Records {
		fmt.Fprintln(w, record.ToBindString())
	}

}

//////////////////////////////////////////////////////////////////////////
// end of code
