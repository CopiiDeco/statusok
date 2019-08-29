package database

import (
	"context"
	"log"

	"cloud.google.com/go/logging"
)

type Stackdriver struct {
	ProjectId string `json:"projectId"`
	LogName   string `json:"logName"`
}

var (
	StackdriverCon *logging.Client
)

//Return database name
func (stackdriver Stackdriver) GetDatabaseName() string {
	var DatabaseName = "Stackdriver"
	return DatabaseName
}

func (stackdriver Stackdriver) printLog(severity logging.Severity, message interface{}) error {
	ctx := context.Background()
	println("Stackdriver : Trying to Connect to database ")
	// Sets your Google Cloud Platform project ID.
	projectID := stackdriver.ProjectId
	var err error
	// Creates a client.
	StackdriverCon, err = logging.NewClient(ctx, projectID)
	if err != nil {
		log.Println("Failed to create client: %v", err)
		return err
	}
	defer StackdriverCon.Close()

	// Sets the name of the log to write to.
	logName := stackdriver.LogName

	StackdriverCon.Logger(logName).Log(logging.Entry{Payload: message, Severity: severity})

	return nil
}

//Intiliaze influx db
func (stackdriver Stackdriver) Initialize() error {
	_ = stackdriver.printLog(logging.Info, "Initialization of the logging for statusok")
	return nil
}

//Add request information to database
func (stackdriver Stackdriver) AddRequestInfo(requestInfo RequestInfo) error {
	err := stackdriver.printLog(logging.Info, requestInfo)
	return err
}

//Add Error information to database
func (stackdriver Stackdriver) AddErrorInfo(errorInfo ErrorInfo) error {
	err := stackdriver.printLog(logging.Error, errorInfo)
	return err
}
