package database

import (
	"context"
	"log"

	"cloud.google.com/go/logging"
	"google.golang.org/api/option"
)

type Stackdriver struct {
	ProjectId              string `json:"projectId"`
	LogName                string `json:"logName"`
	ServiceAccountFilePath string `json:"serviceAccountFilePath"`
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
	println("Stackdriver logging ")
	// Sets your Google Cloud Platform project ID.
	projectID := stackdriver.ProjectId
	serviceAccountFilePath := stackdriver.ServiceAccountFilePath
	var err error
	// Creates a client.
	if serviceAccountFilePath != "" {
		StackdriverCon, err = logging.NewClient(ctx, projectID, option.WithCredentialsFile(serviceAccountFilePath))
		if err != nil {
			log.Println("Failed to create client: ", err)
			return err
		}
	} else {
		StackdriverCon, err = logging.NewClient(ctx, projectID)
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
