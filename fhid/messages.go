package fhid

import (
	"fmt"

	"github.build.ge.com/212601587/fhid/fhidLogger"
)

func messageInvalidRequest(err error) string {
	msg := fmt.Sprintf(`{"Msg":"Invalid Request","Error":"%v"}`, err)
	fhidLogger.Loggo.Error("Invalid Request", "Error", err)
	return msg
}

func messageErrorHandler(err error) string {
	msg := fmt.Sprintf(`{"Msg":"Internal Server Error","Error":"%v"}`, err)
	fhidLogger.Loggo.Error("Internal Server Error", "Error", err)
	return msg
}

func messageErrorHandlerQuery(err error) string {
	msg := fmt.Sprintf(`{"Msg":"Query failed.","Error":"%v"}`, err)
	fhidLogger.Loggo.Error("Query failed", "Error", err)
	return msg
}

func messageSuccess() string {
	return `{"Success": "True"}`
}

func messageSuccessData(s string) string {
	return fmt.Sprintf(`{"Success": "True", "Data": "%s"}`, s)
}

func messageFailData(s string) string {
	return fmt.Sprintf(`{"Success": "False", "Data": "%s"}`, s)
}

func messageMethodNotAllowed() string {
	return `{"Error":"Method not allowed"}`
}
