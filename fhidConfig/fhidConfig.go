package fhidConfig

import (
	"encoding/json"
	"os"
)

// Config is the exported configuration
// that other packages can use during
// runtime
var Config *Configuration

// Version is globally accessible version.
var Version string

// Entitlement holds info about a certain
// type of access such as read/write
type Entitlement struct {
	Type string
}

// AuthGroup stores a basic struct for
// holding group names and gids that are
// authorized to access services.
type AuthGroup struct {
	GroupID      string
	FriendlyName string
	Entitlements []*Entitlement
}

// Authentication holds info about
// the authentication mechanisms.
type Authentication struct {
	AuthEnabled           bool
	AuthURL               string
	AuthHeaderKey         string
	AuthHeaderGroup       string
	AuthMemberCheckMethod string
	AuthorizedGroups      []*AuthGroup
}

// Configuration is a struct used
// to build the exported Config variable
type Configuration struct {
	RedisEndpoint      string
	RedisImageIndexSet string
	ListenPort         string
	ListenHost         string
	Authentication     *Authentication
}

// ShowConfig returns a string of log formatted
// config for debug purposes
func (c *Configuration) ShowConfig() string {
	bs, err := json.Marshal(c)
	if err != nil {
		msg := "Error Marshalling configuration."
		return msg
	}
	return string(bs)
}

// SetConfig parses a config json file and returns
// and sets a package exported configuration object
// for use within other packages
func SetConfig(filename string) error {
	file, err := os.Open(filename)
	if err != nil {
		return err
	}
	decoder := json.NewDecoder(file)
	err = decoder.Decode(&Config)
	return err
}
