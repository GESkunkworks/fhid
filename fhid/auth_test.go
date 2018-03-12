package fhid

import (
	"bytes"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.build.ge.com/212601587/fhid/fhidConfig"
	"github.build.ge.com/212601587/fhid/fhidLogger"
	"github.com/jarcoal/httpmock"
)

func mockHandlerGaudiBad(w http.ResponseWriter, r *http.Request) {
	response := `{"Success":false,"Message":"User not found in group","UserID":"212601587","GroupID":"g00919618"}`
	http.Error(w, response, http.StatusUnauthorized)
}

// TestAuthGoodRead tests to make sure a valid user with a
// read entitlement can successfully get a 404 response from
// the api.
func TestAuthGoodRead(t *testing.T) {
	initLog()
	// we initialize the fake redis instance
	addr, err := runFakeRedis()
	fhidLogger.Loggo.Info("Done starting fake Redis.")
	if err != nil {
		t.Errorf("Unable to start fake Redis for testing: %s", err)
	}
	err = setup(true, addr)
	if err != nil {
		t.Errorf("Unable to connect to fake Redis for testing: %s", err)
	}
	if err != nil {
		t.Fatal(err)
	}
	// make sure auth is enabled
	fhidConfig.Config.Authentication.AuthEnabled = true
	// now that setup has been run we can launch fake gaudi
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()
	httpmock.RegisterResponder("GET", "https://auth.me.com/v1.0/validmember",
		httpmock.NewStringResponder(200, `{"Success":true,"Message":"User is currently valid and is member of group","UserID":"212601587","GroupID":"g00919618"}`))
	// now we build a request
	req, err := http.NewRequest("GET", "/images?ImageID=123-456", nil)

	if err != nil {
		t.Fatal(err)
	}
	req.Header.Add(fhidConfig.Config.Authentication.AuthHeaderKey, "12345")
	req.Header.Add(fhidConfig.Config.Authentication.AuthHeaderGroup, "g00919618")
	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(HandlerImages)
	handler.ServeHTTP(rr, req)
	fhidLogger.Loggo.Debug("Got response", "Response", rr.Body)
	if status := rr.Code; status != http.StatusNotFound {
		msg := fmt.Sprintf("handler returned wrong status code. Got %d, Want %d", status, http.StatusNotFound)
		t.Fatal(msg)
		fhidLogger.Loggo.Error("handler returned wrong status code",
			"Got", status, "Want", http.StatusNotFound)
	}
	httpmock.DeactivateAndReset()
}

// TestAuthBad makes sure that the api will not allow a user
// to access any protected endpoint if the user isn't a member
// of any groups.
func TestAuthBad(t *testing.T) {
	initLog()
	// we initialize the fake redis instance
	addr, err := runFakeRedis()
	fhidLogger.Loggo.Info("Done starting fake Redis.")
	if err != nil {
		t.Errorf("Unable to start fake Redis for testing: %s", err)
	}
	err = setup(true, addr)
	if err != nil {
		t.Errorf("Unable to connect to fake Redis for testing: %s", err)
	}
	// make sure auth is enabled
	fhidConfig.Config.Authentication.AuthEnabled = true
	// now that setup has been run we can launch fake gaudi
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()
	httpmock.RegisterResponder("GET", "https://auth.me.com/v1.0/validmember",
		httpmock.NewStringResponder(401, `{"Success":false,"Message":"User not found in group","UserID":"212601587","GroupID":"g00919618"}`))
	// now we build a request
	postBody := bytes.NewBufferString(imageGood)
	req, err := http.NewRequest("POST", "/images/?Score=0", postBody)
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Add(fhidConfig.Config.Authentication.AuthHeaderKey, "12345")
	req.Header.Add(fhidConfig.Config.Authentication.AuthHeaderGroup, "g1234566")
	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(HandlerImages)
	handler.ServeHTTP(rr, req)
	if status := rr.Code; status != http.StatusUnauthorized {
		msg := fmt.Sprintf("handler returned wrong status code. Got %d, Want %d", status, http.StatusUnauthorized)
		t.Fatal(msg)
		fhidLogger.Loggo.Error("handler returned wrong status code",
			"Got", status, "Want", http.StatusUnauthorized)
	}
	httpmock.DeactivateAndReset()
}

// TestAuthWrongGroup checks to make sure that even if a user
// is in a valid group they need to have the right entitlement.
func TestAuthWrongGroup(t *testing.T) {
	initLog()
	// we initialize the fake redis instance
	addr, err := runFakeRedis()
	fhidLogger.Loggo.Info("Done starting fake Redis.")
	if err != nil {
		t.Errorf("Unable to start fake Redis for testing: %s", err)
	}
	err = setup(true, addr)
	if err != nil {
		t.Errorf("Unable to connect to fake Redis for testing: %s", err)
	}
	// make sure auth is enabled
	fhidConfig.Config.Authentication.AuthEnabled = true
	// now that setup has been run we can launch fake gaudi
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()
	httpmock.RegisterResponder("GET", "https://auth.me.com/v1.0/validmember",
		func(req *http.Request) (resp *http.Response, err error) {
			if req.Header.Get(fhidConfig.Config.Authentication.AuthHeaderGroup) == "g00919618" {
				resp = httpmock.NewStringResponse(200, `{"Success":true,"Message":"User is currently valid and is member of group","UserID":"212601587","GroupID":"g00919618"}`)
				return resp, nil
			}
			resp = httpmock.NewStringResponse(401, `{"Success":false,"Message":"User not found in group","UserID":"212601587","GroupID":"g01236390"}`)
			return resp, nil
		})
	// now we build a request
	postBody := bytes.NewBufferString(imageGood)
	req, err := http.NewRequest("POST", "/images/?Score=0", postBody)
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Add(fhidConfig.Config.Authentication.AuthHeaderKey, "12345")
	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(HandlerImages)
	handler.ServeHTTP(rr, req)
	if status := rr.Code; status != http.StatusUnauthorized {
		msg := fmt.Sprintf("handler returned wrong status code. Got %d, Want %d", status, http.StatusUnauthorized)
		t.Fatal(msg)
		fhidLogger.Loggo.Error("handler returned wrong status code",
			"Got", status, "Want", http.StatusUnauthorized)
	}
	httpmock.DeactivateAndReset()
}
