package fhid

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"os"
	"regexp"
	"testing"

	"github.build.ge.com/212601587/fhid/fhidConfig"
	"github.build.ge.com/212601587/fhid/fhidLogger"
	"github.com/alicebob/miniredis"
)

type imagePostResponse struct {
	Success string
	Data    string
}

const imageGood = `
{
"Version":"1.2.3.145",
"BaseOS":"Ubuntu14.04",
"BuildNotes":{
	"BuildLog": ["line one","line two"],
	"OutputAmis": [
		{"AmiID": "ami-12345","AmiRegion":"us-east-1","AmiTags":[{"Key":"test","Value":"test"}]},
		{"AmiID": "ami-54321","AmiRegion":"us-west-1","AmiTags":[{"Key":"test","Value":"test"}]}
	]
},
"ReleaseNotes":{}
}
`

const imageBad = `
{
"Version":"1.2.3.145",
"BaseOS":"Ubuntu14.04",
"BuildNotes":{
	"BuildLog": ["line one","line two"],
	"OutputAmis": [
		{"AmiID": "ami-12345","AmiRegion":"us-east-1","AmiTags":[{"Key":"test","Value":"test"}]},
		{"AmiID": "ami-54321","AmiRegion":"us-west-1","AmiTags":[{"Key":"test","Value":"test"}]}
	]
},
"ReleaseNotes":{}
`

const imageGoodReleaseUpdate = `
{
"ReleaseNotes":{
	"ReleaseNote": "Pushing out a thing to do that dingy",
	"Amis": [
		{"AmiID": "ami-54321","AmiRegion":"us-west-1", 
		 "AmiTags":[{"Key":"test","Value":"test"}],
		 "AmiSharedTo": ["1234567","7654321","67183674","10239485"]},
		{"AmiID": "ami-54322","AmiRegion":"us-east-1", 
		 "AmiTags":[{"Key":"test","Value":"test"}],
		 "AmiSharedTo": ["1234567","7654321","67183674","10239485"]}
	],
	"ReleaseDate": "2018-01-30 04:36:25"
}
}
`

const imageGood2 = `
{
"Version":"3.4.3.99",
"BaseOS":"Centos7",
"BuildNotes":{
	"Tags":[{"Key":"test","Value":"test"}],
	"BuildLog": ["line one","line two"],
	"OutputAmis": [
		{"AmiID": "ami-12345","AmiRegion":"us-east-1"},
		{"AmiID": "ami-54321","AmiRegion":"us-west-1"}
	]
}
}
`

const imageGoodExpected = `
{
"Results":[{
"ImageID":".*",
"Version":"1.2.3.145",
"BaseOS":"Ubuntu14.04",
"BuildNotes":{
	"Tags":[{"Key":"test","Value":"test"}],
	"BuildLog": ["line one","line two"],
	"OutputAmis": [
		{"AmiID": "ami-12345","AmiRegion":"us-east-1"},
		{"AmiID": "ami-54321","AmiRegion":"us-west-1"}
	]
}}]}`

const imageGoodNonMatcher = `
{
"Version":"9999999999",
"BaseOS":"Winders",
"BuildNotes":{
	"Tags":[{"Key":"test","Value":"test"}],
	"BuildLog": ["line one","line two"],
	"OutputAmis": [
		{"AmiID": "ami-12345","AmiRegion":"us-east-1"},
		{"AmiID": "ami-54321","AmiRegion":"us-west-1"}
	]
}
}
`

const imageWithReleaseNotes = `
{
"Version":"9999999999",
"BaseOS":"Arch",
"BuildNotes":{
	"BuildLog": ["line one","line two"],
	"OutputAmis": [
		{"AmiID": "ami-54321","AmiRegion":"us-west-1", 
		 "AmiTags":[{"Key":"test","Value":"test"}],
		 "AmiSharedTo": ["1234567","7654321"]}
	]
},
"ReleaseNotes":{
	"ReleaseNote": "Pushing out a thing to do that thingy",
	"Amis": [
		{"AmiID": "ami-54321","AmiRegion":"us-west-1", 
		 "AmiTags":[{"Key":"test","Value":"test"}],
		 "AmiSharedTo": ["1234567","7654321","67183674","10239485"]},
		{"AmiID": "ami-54322","AmiRegion":"us-east-1", 
		 "AmiTags":[{"Key":"test","Value":"test"}],
		 "AmiSharedTo": ["1234567","7654321","67183674","10239485"]}
	],
	"ReleaseDate": "2018-01-30 04:36:25"
}
}
`

const imageWithReleaseNotes2 = `
{
"Version":"9999999998",
"BaseOS":"Arch",
"BuildNotes":{
	"BuildLog": ["line one","line two"],
	"OutputAmis": [
		{"AmiID": "ami-54321","AmiRegion":"us-west-1", 
		 "AmiTags":[{"Key":"test","Value":"test"}],
		 "AmiSharedTo": ["1234567","7654321"]}
	]
},
"ReleaseNotes":{
	"ReleaseNote": "Pushing out a thing to do that dingy",
	"Amis": [
		{"AmiID": "ami-54321","AmiRegion":"us-west-1", 
		 "AmiTags":[{"Key":"test","Value":"test"}],
		 "AmiSharedTo": ["1234567","7654321","67183674","10239485"]},
		{"AmiID": "ami-54322","AmiRegion":"us-east-1", 
		 "AmiTags":[{"Key":"test","Value":"test"}],
		 "AmiSharedTo": ["1234567","7654321","67183674","10239485"]}
	],
	"ReleaseDate": "2018-01-30 04:36:25"
}
}
`

const ImageQueryVersion = `
{
	"Version": {"StringMatch": "1.2.3.145"}
}
`

const ImageQueryBuildNotes = `
{
	"BuildNotes": {"StringMatch": ".*ami-12345.*"}
}
`

const ImageQueryBaseOS = `
{
	"BaseOS": {"StringMatch": ".*Ubuntu.*"}
}
`

const ImageQueryReleaseNotes = `
{
	"ReleaseNotes": {"StringMatch": ".*thingy.*"}
}
`

func resultsMatchExpected(results, expected string) (match bool, err error) {
	fhidLogger.Loggo.Info("Checking to see if results match expected.")
	// unmarshal actual results
	var iqrGot ImageQueryResults
	bresults := []byte(results)
	err = json.Unmarshal(bresults, &iqrGot)
	if err != nil {
		fhidLogger.Loggo.Error("Error unmarshaling query results", "Error", err)
		return false, err
	}

	// unmarshal expected results
	var iqrWant ImageQueryResults
	bexpected := []byte(expected)
	err = json.Unmarshal(bexpected, &iqrWant)
	if err != nil {
		fhidLogger.Loggo.Error("Error unmarshaling expected results", "Error", err)
		return false, err
	}

	// now loop through the expected and match one for one to results
	// except for ImageID since it's GUUID and changes every time
	match = true
	if len(iqrWant.Results) != len(iqrWant.Results) {
		return false, err
	}
	for idx, exp := range iqrWant.Results {
		dateMatch, err := regexp.Match(`\d{4}-\d{2}-\d{2} \d{2}:\d{2}:\d{2}`,
			[]byte(iqrGot.Results[idx].CreateDate))
		switch {
		case !dateMatch:
			return false, err
		case exp.Version != iqrGot.Results[idx].Version:
			return false, err
		case exp.BuildNotes != iqrGot.Results[idx].BuildNotes:
			return false, err
		case exp.BaseOS != iqrGot.Results[idx].BaseOS:
			return false, err
		}
	}
	return match, err
}

func writeConfigFile() (*bytes.Buffer, error) {
	seed := `{
        "RedisEndpoint": "localhost:6379",
		"RedisImageIndexSet": "IMAGE_INDEX",
        "ListenPort": "8090",
		"ListenHost": "127.0.0.1",
		"Authentication": {
			"AuthEnabled": true,
            "AuthURL": "https://auth.me.com/v1.0",
            "AuthMemberCheckMethod": "/validmember",
            "AuthHeaderKey": "x-api-key",
            "AuthHeaderGroup" : "GroupID",
            "AuthorizedGroups": [
                {
                    "GroupID": "g01236390",
                    "FriendlyName": "@Digital CoreTech Cloud Build Prod",
                    "Entitlements" : [
                        {"Type": "write" },
                        {"Type": "read" }
                    ]
                },
                {
                    "GroupID": "g00919618",
                    "FriendlyName": "@GE Public Cloud",
                    "Entitlements" : [
                        {"Type": "read" }
                    ]
                }
            ]
        }
}`
	var b bytes.Buffer
	_, err := b.WriteString(seed)
	if err != nil {
		fmt.Println("error: ", err)
		return &b, err
	}
	return &b, err
}

func writeBufferToFile(filename string, b *bytes.Buffer) error {
	err := ioutil.WriteFile(filename, b.Bytes(), 0644)
	return err
}

func deleteFile(filename string) {
	err := os.Remove(filename)
	if err != nil {
		panic(err)
	}
}

func setup(fake bool, addr string) error {
	filename := "handlers_test_config_temp.json"
	configbuff, err := writeConfigFile()
	err = writeBufferToFile(filename, configbuff)
	if err != nil {
		fhidLogger.Loggo.Error("Error writing temp config file", "Error", err)
		return err
	}
	err = fhidConfig.SetConfig(filename)
	fhidLogger.Loggo.Debug("Config parsed.", "Config", fhidConfig.Config.ShowConfig())
	if err != nil {
		fhidLogger.Loggo.Error("Error parsing config file", "Error", err)
		return err
	}
	if fake == true {
		fhidConfig.Config.RedisEndpoint = addr
	}
	fhidLogger.Loggo.Debug("Connecting to Redis at address.", "Address", fhidConfig.Config.RedisEndpoint)
	err = SetupConnection()
	if err != nil {
		fhidLogger.Loggo.Error("Error in Redis test connection", "Error", err)
		TeardownConnection()
		return err
	}
	defer deleteFile(filename)
	return err

}

func initLog() {
	skipWriteToFile := false
	daemonMode := true
	fhidLogger.SetLogger(daemonMode, skipWriteToFile, "fhid_test.log.json", "debug")
}

func runFakeRedis() (addr string, err error) {
	s, err := miniredis.Run()
	if err != nil {
		return "", err
	}
	addr = s.Addr()
	return addr, err
}

func seedQueryData() error {
	// turn off auth so we can seed data
	fhidConfig.Config.Authentication.AuthEnabled = false
	postBody := bytes.NewBufferString(imageGood)
	req, err := http.NewRequest("POST", "/images/?Score=0", postBody)
	if err != nil {
		return err
	}
	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(HandlerImages)
	handler.ServeHTTP(rr, req)
	if status := rr.Code; status != http.StatusOK {
		fhidLogger.Loggo.Error("handler returned wrong status code",
			"Got", status, "Want", http.StatusOK)
	}
	var j imagePostResponse
	err = json.Unmarshal([]byte(rr.Body.String()), &j)
	if err != nil {
		fhidLogger.Loggo.Error("Unable to unmarshal response JSON",
			"Error", err)
		return err
	}

	// write a second entry
	postBody = bytes.NewBufferString(imageGood2)
	req, err = http.NewRequest("POST", "/images/?Score=1", postBody)
	if err != nil {
		return err
	}
	handler.ServeHTTP(rr, req)
	if status := rr.Code; status != http.StatusOK {
		fhidLogger.Loggo.Error("handler returned wrong status code",
			"Got", status, "Want", http.StatusOK)
	}

	// write a third entry that shouldn't match query
	postBody = bytes.NewBufferString(imageGoodNonMatcher)
	req, err = http.NewRequest("POST", "/images/?Score=2", postBody)
	if err != nil {
		return err
	}
	handler.ServeHTTP(rr, req)
	if status := rr.Code; status != http.StatusOK {
		fhidLogger.Loggo.Error("handler returned wrong status code",
			"Got", status, "Want", http.StatusOK)
	}

	// write another entry test new releasenotes struct
	postBody = bytes.NewBufferString(imageWithReleaseNotes)
	req, err = http.NewRequest("POST", "/images/?Score=3", postBody)
	if err != nil {
		return err
	}
	handler.ServeHTTP(rr, req)
	if status := rr.Code; status != http.StatusOK {
		fhidLogger.Loggo.Error("handler returned wrong status code",
			"Got", status, "Want", http.StatusOK)
	}

	// turn auth back on again
	fhidConfig.Config.Authentication.AuthEnabled = true
	return err

}

func TestStatus(t *testing.T) {
	initLog()
	req, err := http.NewRequest("GET", "/healthcheck", nil)
	if err != nil {
		t.Fatal(err)
	}
	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(HealthCheck)
	handler.ServeHTTP(rr, req)
	if status := rr.Code; status != http.StatusOK {
		fhidLogger.Loggo.Error("handler returned wrong status code",
			"Got", status, "Want", http.StatusOK)
	}
}

func TestBadMethods(t *testing.T) {
	initLog()
	// test DELETE method
	req, err := http.NewRequest("DELETE", "/images", nil)
	if err != nil {
		t.Fatal(err)
	}
	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(HandlerImages)
	handler.ServeHTTP(rr, req)
	if status := rr.Code; status != http.StatusMethodNotAllowed {
		fhidLogger.Loggo.Error("handler returned wrong status code",
			"Got", status, "Want", http.StatusMethodNotAllowed)
	}
	// test PUT method
	req, err = http.NewRequest("PUT", "/images", nil)
	if err != nil {
		t.Fatal(err)
	}
	rr = httptest.NewRecorder()
	handler = http.HandlerFunc(HandlerImages)
	handler.ServeHTTP(rr, req)
	if status := rr.Code; status != http.StatusMethodNotAllowed {
		fhidLogger.Loggo.Error("handler returned wrong status code",
			"Got", status, "Want", http.StatusMethodNotAllowed)
	}
	// test HEAD method
	req, err = http.NewRequest("HEAD", "/images", nil)
	if err != nil {
		t.Fatal(err)
	}
	rr = httptest.NewRecorder()
	handler = http.HandlerFunc(HandlerImages)
	handler.ServeHTTP(rr, req)
	if status := rr.Code; status != http.StatusMethodNotAllowed {
		fhidLogger.Loggo.Error("handler returned wrong status code",
			"Got", status, "Want", http.StatusMethodNotAllowed)
	}
}

func TestImageBad(t *testing.T) {
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
	// First we need to post some entries to the DB
	// We have to use the score URL query so we force order or results
	// since the Redis index set we're using is a sorted set and only
	// sorts if proper score weights are given.
	postBody := bytes.NewBufferString(imageGood)
	req, err := http.NewRequest("POST", "/images/?Score=0", postBody)
	if err != nil {
		t.Fatal(err)
	}
	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(HandlerImages)
	handler.ServeHTTP(rr, req)
	if status := rr.Code; status != http.StatusBadRequest {
		fhidLogger.Loggo.Error("handler returned wrong status code",
			"Got", status, "Want", http.StatusBadRequest)
	}
}

func TestImageUpdate(t *testing.T) {
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
	// turn off auth
	fhidConfig.Config.Authentication.AuthEnabled = false
	// First we need to post some entries to the DB
	// We have to use the score URL query so we force order or results
	// since the Redis index set we're using is a sorted set and only
	// sorts if proper score weights are given.
	postBody := bytes.NewBufferString(imageGood)
	req, err := http.NewRequest("POST", "/images/?Score=0", postBody)
	if err != nil {
		t.Fatal(err)
	}
	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(HandlerImages)
	handler.ServeHTTP(rr, req)
	if status := rr.Code; status != http.StatusOK {
		fhidLogger.Loggo.Error("handler returned wrong status code",
			"Got", status, "Want", http.StatusOK)
	}
	var j imagePostResponse
	err = json.Unmarshal([]byte(rr.Body.String()), &j)
	if err != nil {
		fhidLogger.Loggo.Error("Unable to unmarshal response JSON",
			"Error", err)
		t.Fatal(err)
	}
	imageID := j.Data
	// set up response recorder
	rr = httptest.NewRecorder()
	queryBody := bytes.NewBufferString(imageGoodReleaseUpdate)
	urlString := fmt.Sprintf("/image?ImageID=%s", imageID)
	req, err = http.NewRequest("PATCH", urlString, queryBody)
	if err != nil {
		t.Fatal(err)
	}
	rr = httptest.NewRecorder()
	handler = http.HandlerFunc(HandlerImages)
	handler.ServeHTTP(rr, req)
	match := (rr.Code == 200)
	if !match {
		t.Errorf("handler returned unexpected results: got '%v' want '%v'",
			rr.Code, 200)
	}
}

func TestImageQueryBuildNotes(t *testing.T) {
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
	// First we need to post some entries to the DB
	// We have to use the score URL query so we force order or results
	// since the Redis index set we're using is a sorted set and only
	// sorts if proper score weights are given.
	err = seedQueryData()
	if err != nil {
		t.Errorf("Error seeding query data. '%s'", err)
	}
	// turn off auth again
	fhidConfig.Config.Authentication.AuthEnabled = false
	// set up response recorder
	rr := httptest.NewRecorder()
	queryBody := bytes.NewBufferString(ImageQueryBuildNotes)
	req, err := http.NewRequest("POST", "/query", queryBody)
	if err != nil {
		t.Fatal(err)
	}
	rr = httptest.NewRecorder()
	handler := http.HandlerFunc(HandlerImagesQuery)
	handler.ServeHTTP(rr, req)
	// Check the number of results is what we expect
	var results ImageQueryResults
	err = json.Unmarshal(rr.Body.Bytes(), &results)
	expectedResults := 3
	match := (len(results.Results) == expectedResults)
	if !match {
		t.Errorf("handler returned unexpected number of results: got '%v' want '%v'",
			len(results.Results), expectedResults)
	}
}

func TestImageQueryReleaseNotes(t *testing.T) {
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
	// First we need to post some entries to the DB
	// We have to use the score URL query so we force order or results
	// since the Redis index set we're using is a sorted set and only
	// sorts if proper score weights are given.
	err = seedQueryData()
	if err != nil {
		t.Errorf("Error seeding query data. '%s'", err)
	}
	// turn off auth again
	fhidConfig.Config.Authentication.AuthEnabled = false
	// set up response recorder
	rr := httptest.NewRecorder()
	queryBody := bytes.NewBufferString(ImageQueryReleaseNotes)
	req, err := http.NewRequest("POST", "/query", queryBody)
	if err != nil {
		t.Fatal(err)
	}
	rr = httptest.NewRecorder()
	handler := http.HandlerFunc(HandlerImagesQuery)
	handler.ServeHTTP(rr, req)
	// Check the number of results is what we expect
	var results ImageQueryResults
	err = json.Unmarshal(rr.Body.Bytes(), &results)
	expectedResults := 1
	match := (len(results.Results) == expectedResults)
	if !match {
		t.Errorf("handler returned unexpected number of results: got '%v' want '%v'",
			len(results.Results), expectedResults)
	}
}

func TestImageQueryVersion(t *testing.T) {
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
	// First we need to post some entries to the DB
	// We have to use the score URL query so we force order or results
	// since the Redis index set we're using is a sorted set and only
	// sorts if proper score weights are given.
	err = seedQueryData()
	if err != nil {
		t.Errorf("Error seeding query data. '%s'", err)
	}
	// turn off auth again
	fhidConfig.Config.Authentication.AuthEnabled = false
	// set up response recorder
	rr := httptest.NewRecorder()
	queryBody := bytes.NewBufferString(ImageQueryVersion)
	req, err := http.NewRequest("POST", "/image_query", queryBody)
	if err != nil {
		t.Fatal(err)
	}
	rr = httptest.NewRecorder()
	handler := http.HandlerFunc(HandlerImagesQuery)
	handler.ServeHTTP(rr, req)
	// Check the number of results is what we expect
	var results ImageQueryResults
	err = json.Unmarshal(rr.Body.Bytes(), &results)
	expectedResults := 1
	match := (len(results.Results) == expectedResults)
	if !match {
		t.Errorf("handler returned unexpected number of results: got '%v' want '%v'",
			len(results.Results), expectedResults)
	}
}

func TestImageGetNone(t *testing.T) {
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

	// turn off auth
	fhidConfig.Config.Authentication.AuthEnabled = false
	req, err := http.NewRequest("GET", "/images", nil)
	if err != nil {
		t.Fatal(err)
	}
	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(HandlerImages)
	handler.ServeHTTP(rr, req)
	if status := rr.Code; status != http.StatusBadRequest {
		t.Errorf("handler returned wrong status code: got %v want %v",
			status, http.StatusBadRequest)
	}
	// Check the response body is what we expect.
	expected := `{"Error": "Key 'ImageID' not found in URL string."}` + "\n"
	if rr.Body.String() != expected {
		t.Errorf("handler returned unexpected body: got %v want %v",
			rr.Body.String(), expected)
	}
}

func TestImageQueryBaseOS(t *testing.T) {
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
	// First we need to post some entries to the DB
	// We have to use the score URL query so we force order or results
	// since the Redis index set we're using is a sorted set and only
	// sorts if proper score weights are given.
	err = seedQueryData()
	if err != nil {
		t.Errorf("Error seeding query data. '%s'", err)
	}
	// turn off auth again
	fhidConfig.Config.Authentication.AuthEnabled = false
	// set up response recorder
	rr := httptest.NewRecorder()
	queryBody := bytes.NewBufferString(ImageQueryBaseOS)
	req, err := http.NewRequest("POST", "/image_query", queryBody)
	if err != nil {
		t.Fatal(err)
	}
	rr = httptest.NewRecorder()
	handler := http.HandlerFunc(HandlerImagesQuery)
	handler.ServeHTTP(rr, req)
	// Check the number of results is what we expect
	var results ImageQueryResults
	err = json.Unmarshal(rr.Body.Bytes(), &results)
	expectedResults := 1
	match := (len(results.Results) == expectedResults)
	if !match {
		t.Errorf("handler returned unexpected number of results: got '%v' want '%v'",
			len(results.Results), expectedResults)
	}
}
