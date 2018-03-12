package fhid

import (
	"errors"
	"fmt"
	"net/http"

	"github.build.ge.com/212601587/fhid/fhidLogger"

	"github.build.ge.com/212601587/fhid/fhidConfig"
)

// callAuth calls out to the auth URL and checks to see if the provided
// authKey is a member of the provided groupID.
func callAuth(authKey string, groupID string) (member bool, err error) {
	member = false
	url := fhidConfig.Config.Authentication.AuthURL + fhidConfig.Config.Authentication.AuthMemberCheckMethod
	fhidLogger.Loggo.Debug("Build auth url.", "URL", url)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return member, err
	}
	// set authkey in request header
	req.Header.Set(fhidConfig.Config.Authentication.AuthHeaderKey, authKey)
	// set groupid in request header
	req.Header.Set(fhidConfig.Config.Authentication.AuthHeaderGroup, groupID)
	fhidLogger.Loggo.Info("Calling auth url", "URL", url)
	client := http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		fhidLogger.Loggo.Error("Got error from auth url", "Error", err)
		return member, err
	}
	fhidLogger.Loggo.Info("Got response from auth url", "Response", resp)
	if resp.StatusCode == http.StatusOK {
		member = true
	} else if resp.StatusCode == http.StatusUnauthorized {
		member = false
		err = errors.New("Unauthorized")
	}
	return member, err
}

// redacter just trims out chars from a sensitive input
// string
func redacter(pure string) (redacted string) {
	redacted = fmt.Sprintf("%.5s", pure)
	redacted = redacted + "...[REDACTED]..."
	return redacted
}

// requiresAuth takes a request and a desired entitlement and parses
// the config and then calls the auth url to see if the token belongs
// to an authorized user. Returns true if the user is entitled and an
// error.
func requiresAuth(r *http.Request, needs string) (err error) {
	fhidLogger.Loggo.Info("Entering requiresAuth")
	authKey := r.Header.Get(fhidConfig.Config.Authentication.AuthHeaderKey)
	authKeyRedacted := redacter(authKey)
	fhidLogger.Loggo.Debug("debug authkey", "authkeyRedacted", authKeyRedacted)
	hasEntitlement := false
	for _, group := range fhidConfig.Config.Authentication.AuthorizedGroups {
		fhidLogger.Loggo.Debug("working on group", "Group", group.GroupID)
		fhidLogger.Loggo.Debug("value of hasentitlement", "hasEntitlement", hasEntitlement)
		if !hasEntitlement {
			member, err := callAuth(authKey, group.GroupID)
			if err != nil {
				fhidLogger.Loggo.Error("Error from callAuth", "Error", err)
				return err
			}
			if member {
				for _, entitlement := range group.Entitlements {
					fhidLogger.Loggo.Info("comparing entitlements for group", "GroupID", group.GroupID, "Entitlement", entitlement.Type)

					if needs == entitlement.Type {
						fhidLogger.Loggo.Info("Match!")
						hasEntitlement = true
					}
				}
			}
		}
	}
	if !hasEntitlement {
		err = errors.New(messageUnauthorized())
		return err
	}

	return err
}

// messageUnauthorized generates a user friendly unauthorized
// return string.
func messageUnauthorized() string {
	msg := fmt.Sprintf("Unauthorized. Please make sure you have generated a token here '%s' and that the user who generated the token is a member of an authorized group. For help email cloudpod@ge.com.", fhidConfig.Config.Authentication.AuthURL)
	return msg
}
