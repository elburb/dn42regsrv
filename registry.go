//////////////////////////////////////////////////////////////////////////
// DN42 Registry API Server
//////////////////////////////////////////////////////////////////////////

package main

//////////////////////////////////////////////////////////////////////////

import (
	"bufio"
	//	"errors"
	"fmt"
	log "github.com/sirupsen/logrus"
	"io/ioutil"
	"os"
	"os/exec"
	"strings"
	"time"
)

//////////////////////////////////////////////////////////////////////////
// registry data model

// registry data

// Attributes within Objects
type RegAttribute struct {
	Key      string
	Value    string // this is a post-processed, or decorated value
	RawValue string // the raw value as read from the registry
}

type RegObject struct {
	Ref       string          // the ref contains the full path for this object
	Data      []*RegAttribute // the key/value data for this object
	Backlinks []*RegObject    // other objects that reference this one
}

// types are collections of objects
type RegType struct {
	Ref     string                // full path for this type
	Objects map[string]*RegObject // the objects in this type
}

// registry meta data

type RegAttributeSchema struct {
	Fields    []string
	Relations []*RegType
}

type RegTypeSchema struct {
	Ref        string
	Attributes map[string]*RegAttributeSchema
	KeyIndex   map[string]map[*RegObject][]*RegAttribute
}

// the registry itself

type Registry struct {
	Schema map[string]*RegTypeSchema
	Types  map[string]*RegType
}

// and a variable for the actual data
var RegistryData *Registry

// store the current commit has
var previousCommit string

//////////////////////////////////////////////////////////////////////////
// utility and manipulation functions

// general functions

func RegistryMakePath(t string, o string) string {
	return t + "/" + o
}

// attribute functions

// nothing here

// object functions

// return attributes exactly matching a specific key
func (object *RegObject) GetKey(key string) []*RegAttribute {

	attributes := make([]*RegAttribute, 0)
	for _, a := range object.Data {
		if a.Key == key {
			attributes = append(attributes, a)
		}
	}

	return attributes
}

// return a single key
func (object *RegObject) GetSingleKey(key string) *RegAttribute {

	attributes := object.GetKey(key)
	if len(attributes) != 1 {
		log.WithFields(log.Fields{
			"key":    key,
			"object": object.Ref,
		}).Error("Unable to find unique key in object")

		// can't register the object
		return nil
	}
	return attributes[0]
}

// schema functions

// validate a set of attributes against a schema
func (schema *RegTypeSchema) validate(attributes []*RegAttribute) []*RegAttribute {

	validated := make([]*RegAttribute, 0, len(attributes))
	for _, attribute := range attributes {

		// keys beginning with 'x-' are user defined, skip validation
		if !strings.HasPrefix(attribute.Key, "x-") {
			if schema.Attributes[attribute.Key] == nil {
				// couldn't find a schema attribute

				log.WithFields(log.Fields{
					"key":    attribute.Key,
					"schema": schema.Ref,
				}).Error("Schema validation failed")

				// don't add to the validated list
				continue
			}
		}

		// all ok
		validated = append(validated, attribute)
	}

	return validated
}

// add an attribute to the key map
func (schema *RegTypeSchema) addKeyIndex(object *RegObject,
	attribute *RegAttribute) {

	objmap := schema.KeyIndex[attribute.Key]
	// create a new object map if it didn't exist
	if objmap == nil {
		objmap = make(map[*RegObject][]*RegAttribute)
		schema.KeyIndex[attribute.Key] = objmap
	}

	// add the object/attribute reference
	objmap[object] = append(objmap[object], attribute)
}

// object functions

// add a backlink to an object
func (object *RegObject) addBacklink(ref *RegObject) {

	// check if the backlink already exists, this could be the case
	// if an object is referenced multiple times (e.g. admin-c & tech-c)
	for _, blink := range object.Backlinks {
		if blink == ref {
			// already exists, just return as nothing to do
			return
		}
	}

	// didn't find a match, add the backlink
	object.Backlinks = append(object.Backlinks, ref)

}

//////////////////////////////////////////////////////////////////////////
// reload the registry

func reloadRegistry(path string) {

	log.Debug("Reloading registry")

	// r will become the new registry data
	registry := &Registry{
		Schema: make(map[string]*RegTypeSchema),
		Types:  make(map[string]*RegType),
	}

	// bootstrap the schema registry type
	registry.Types["schema"] = &RegType{
		Ref:     "schema",
		Objects: make(map[string]*RegObject),
	}
	registry.loadType("schema", path)

	// and parse the schema to get the remaining types
	registry.parseSchema()

	// now load the remaining types
	for _, rType := range registry.Types {
		registry.loadType(rType.Ref, path)
	}

	// mark relationships
	registry.decorate()

	// swap in the new registry data
	RegistryData = registry
}

//////////////////////////////////////////////////////////////////////////
// create and load the raw data for a registry type

func (registry *Registry) loadType(typeName string, path string) {

	// the type will already have been created
	rType := registry.Types[typeName]

	// as will the schema (unless attempting to load the schema itself)
	schema := registry.Schema[typeName]

	// special case for DNS as the directory
	// doesn't match the type name
	if typeName == "domain" {
		path += "/dns"
	} else {
		path += "/" + typeName
	}

	// and load all the objects in this type
	rType.loadObjects(schema, path)

}

//////////////////////////////////////////////////////////////////////////
// load all the objects associated with a type

func (rType *RegType) loadObjects(schema *RegTypeSchema, path string) {

	entries, err := ioutil.ReadDir(path)
	if err != nil {
		log.WithFields(log.Fields{
			"error": err,
			"path":  path,
			"type":  rType.Ref,
		}).Error("Failed to read registry type directory")
		return
	}

	// for each entry in the directory
	for _, entry := range entries {

		// each file maps to a registry object
		if !entry.IsDir() {

			filename := entry.Name()
			// ignore dotfiles
			if !strings.HasPrefix(filename, ".") {

				// load the attributes from file
				attributes := loadAttributes(path + "/" + filename)

				// basic validation of attributes against the schema
				// schema may be nil if we are actually loading the schema itself
				if schema != nil {
					attributes = schema.validate(attributes)
				}

				// make the object
				object := &RegObject{
					Ref:       RegistryMakePath(rType.Ref, filename),
					Data:      attributes,
					Backlinks: make([]*RegObject, 0),
				}

				// add to type
				rType.Objects[filename] = object
			}
		}
	}

	log.WithFields(log.Fields{
		"ref":   rType.Ref,
		"path":  path,
		"count": len(rType.Objects),
	}).Debug("Loaded registry type")

}

//////////////////////////////////////////////////////////////////////////
// read attributes from a file

func loadAttributes(path string) []*RegAttribute {

	attributes := make([]*RegAttribute, 0)

	// open the file to start reading it
	file, err := os.Open(path)
	if err != nil {
		log.WithFields(log.Fields{
			"error": err,
			"path":  path,
		}).Error("Failed to read attributes from file")
		return attributes
	}
	defer file.Close()

	// read the file line by line using the bufio scanner
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {

		line := strings.TrimRight(scanner.Text(), "\r\n")
		runes := []rune(line)

		// lines starting with '+' denote an empty line
		if runes[0] == rune('+') {

			// concatenate a \n on to the previous attribute value
			attributes[len(attributes)-1].RawValue += "\n"

		} else {

			// look for a : separator in the first 20 characters
			ix := strings.IndexByte(line, ':')
			if ix == -1 || ix >= 20 {
				// couldn't find one

				if len(runes) <= 20 {
					// hmmm, the line was shorter than 20 characters
					// something is amiss

					log.WithFields(log.Fields{
						"length": len(runes),
						"path":   path,
						"line":   line,
					}).Warn("Short line detected")

				} else {

					// line is a continuation of the previous line, so
					// concatenate the value on to the previous attribute value
					attributes[len(attributes)-1].RawValue +=
						"\n" + string(runes[20:])

				}
			} else {
				// found a key and : separator

				// is there actually a value ?
				var value string
				if len(runes) <= 20 {
					// blank value
					value = ""
				} else {
					value = string(runes[20:])
				}

				// create a new attribute
				a := &RegAttribute{
					Key:      string(runes[:ix]),
					RawValue: value,
				}
				attributes = append(attributes, a)
			}
		}
	}

	return attributes
}

//////////////////////////////////////////////////////////////////////////
// parse schema files to extract keys and for attribute relations

func (registry *Registry) parseSchema() {

	// for each object in the schema type
	for _, object := range registry.Types["schema"].Objects {

		// look up the ref attribute
		ref := object.GetSingleKey("ref")
		if ref == nil {
			log.WithFields(log.Fields{
				"object": object.Ref,
			}).Error("Schema record without ref")

			// can't process this object
			continue
		}

		// create the type schema object
		typeName := strings.TrimPrefix(ref.RawValue, "dn42.")
		typeSchema := &RegTypeSchema{
			Ref:        typeName,
			Attributes: make(map[string]*RegAttributeSchema),
			KeyIndex:   make(map[string]map[*RegObject][]*RegAttribute),
		}

		// ensure the type exists
		rType := registry.Types[typeName]
		if rType == nil {
			rType := &RegType{
				Ref:     typeName,
				Objects: make(map[string]*RegObject),
			}
			registry.Types[typeName] = rType
		}

		// for each key attribute in the schema
		attributes := object.GetKey("key")
		for _, attribute := range attributes {

			// split the value on whitespace
			fields := strings.Fields(attribute.RawValue)
			keyName := fields[0]

			typeSchema.Attributes[keyName] = &RegAttributeSchema{
				Fields: fields[1:],
			}
		}

		// register the type schema
		registry.Schema[typeName] = typeSchema

	}

	// scan the fields of each schema attribute to determine relationships
	// this needs to be second step to allow pre-creation of the types
	for _, typeSchema := range registry.Schema {
		for attribName, attribSchema := range typeSchema.Attributes {
			for _, field := range attribSchema.Fields {
				if strings.HasPrefix(field, "lookup=") {

					// the relationships may be a multivalue, separated by ,
					rels := strings.Split(strings.
						TrimPrefix(field, "lookup="), ",")

					// map to a regtype
					relations := make([]*RegType, 0, len(rels))
					for ix := range rels {
						relName := strings.TrimPrefix(rels[ix], "dn42.")
						relation := registry.Types[relName]

						// log if unable to look up the type
						if relation == nil {
							// log unless this is the schema def lookup=str '>' [spec]...
							if typeSchema.Ref != "schema" {
								log.WithFields(log.Fields{
									"relation":  relName,
									"attribute": attribName,
									"type":      typeSchema.Ref,
								}).Error("Relation to type that does not exist")
							}

						} else {
							// store the relationship
							relations = append(relations, relation)
						}
					}

					// register the relations
					attribSchema.Relations = relations

					// assume only 1 lookup= per key
					break
				}
			}
		}
	}

	log.Debug("Schema parsing complete")
}

//////////////////////////////////////////////////////////////////////////
// parse all attributes and decorate them

func (registry *Registry) decorate() {

	cattribs := 0
	cmatched := 0

	// walk each attribute value
	for _, rType := range registry.Types {
		schema := registry.Schema[rType.Ref]
		for _, object := range rType.Objects {
			for _, attribute := range object.Data {
				cattribs += 1

				// add this attribute to the key map
				schema.addKeyIndex(object, attribute)

				attribSchema := schema.Attributes[attribute.Key]
				// are there relations defined for this attribute ?
				// attribSchema may be null if this attribute is user defined (x-*)
				if (attribSchema != nil) &&
					attribute.matchRelation(object, attribSchema.Relations) {
					// matched
					cmatched += 1
				} else {
					// no match, just copy the attribute data
					attribute.Value = attribute.RawValue
				}
			}
		}
	}

	log.WithFields(log.Fields{
		"attributes": cattribs,
		"matched":    cmatched,
	}).Debug("Decoration complete")

}

//////////////////////////////////////////////////////////////////////////
// match an attribute against schema relations

func (attribute *RegAttribute) matchRelation(parent *RegObject,
	relations []*RegType) bool {

	// it's not going to match if relations is empty
	if relations == nil {
		return false
	}

	// check each relation
	for _, relation := range relations {

		object := relation.Objects[attribute.RawValue]
		if object != nil {
			// found a match !

			// decorate the attribute value
			attribute.Value = fmt.Sprintf("[%s](%s)",
				attribute.RawValue, object.Ref)

			// and add a back reference to the related object
			object.addBacklink(parent)

			return true
		}

	}

	// didn't find anything
	return false
}

//////////////////////////////////////////////////////////////////////////
// fetch the current commit hash

func getCommitHash(regDir string, gitPath string) string {

	// run git to get the latest commit hash
	cmd := exec.Command(gitPath, "log", "-1", "--format=%H")
	cmd.Dir = regDir
	// execute
	out, err := cmd.Output()
	if err != nil {
		log.WithFields(log.Fields{
			"error":   err,
			"gitPath": gitPath,
			"regDir":  regDir,
		}).Error("Failed to execute git log")
	}

	return strings.TrimSpace(string(out))
}

//////////////////////////////////////////////////////////////////////////
// refresh the registry

func refreshRegistry(regDir string, gitPath string, pullURL string) {

	// run git to get the latest commit hash
	cmd := exec.Command(gitPath, "pull", pullURL)
	cmd.Dir = regDir
	// execute
	out, err := cmd.Output()
	if err != nil {
		log.WithFields(log.Fields{
			"error":   err,
			"gitPath": gitPath,
			"regDir":  regDir,
			"pullURL": pullURL,
		}).Error("Failed to execute git log")
	}

	fmt.Println(string(out))
}

//////////////////////////////////////////////////////////////////////////
// called from main to initialse the registry data and syncing

func InitialiseRegistryData(regDir string, refresh time.Duration,
	gitPath string, autoPull bool, pullURL string) {

	// validate that the regDir/data path exists
	dataPath := regDir + "/data"
	regStat, err := os.Stat(dataPath)
	if err != nil {
		log.WithFields(log.Fields{
			"error": err,
			"path":  dataPath,
		}).Fatal("Unable to find registry directory")
	}

	// and it is a directory
	if !regStat.IsDir() {
		log.WithFields(log.Fields{
			"error": err,
			"path":  dataPath,
		}).Fatal("Registry path is not a directory")
	}

	// check that git exists
	_, err = os.Stat(gitPath)
	if err != nil {
		log.WithFields(log.Fields{
			"error": err,
			"path":  gitPath,
		}).Fatal("Unable to find git executable")
	}

	// enforce a minimum update time
	minTime := 10 * time.Minute
	if refresh < minTime {
		log.WithFields(log.Fields{
			"interval": refresh,
		}).Error("Enforcing minimum update time of 10 minutes")

		refresh = minTime
	}

	// initialise the previous commit hash
	// and do initial load from registry
	previousCommit = getCommitHash(regDir, gitPath)
	reloadRegistry(dataPath)

	go func() {

		// every refresh interval
		for range time.Tick(refresh) {
			log.Debug("Refresh Timer")

			// automatically try to refresh the registry ?
			if autoPull {
				refreshRegistry(regDir, gitPath, pullURL)
			}

			// get the latest hash
			currentCommit := getCommitHash(regDir, gitPath)

			// has the registry been updated ?
			if currentCommit != previousCommit {
				log.WithFields(log.Fields{
					"current":  currentCommit,
					"previous": previousCommit,
				}).Info("Registry has changed, refresh started")

				// refresh
				reloadRegistry(dataPath)

				// update commit
				previousCommit = currentCommit
			}

		}
	}()

}

//////////////////////////////////////////////////////////////////////////
// end of code
