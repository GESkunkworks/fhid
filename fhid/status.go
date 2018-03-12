package fhid

import (
	"encoding/json"
	"fmt"

	"github.build.ge.com/212601587/fhid/fhidConfig"
)

// status is an object to hold system status
// to be returned by things like the healthcheck handler
type status struct {
	State   string `json:"State"`
	Version string `json:"Version"`
}

// GetStatus returns the current string export of the
// FhidStatus struct.
func (f *status) getStatus() (msg string) {
	b, err := json.Marshal(f)
	if err != nil {
		msg = fmt.Sprintf(`{"State":"Unhealthy","Version":"%s"}`, fhidConfig.Version)
	}
	msg = string(b)
	return msg
}
