package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"math/rand"
	"net/http"
	"os"
	"time"

	"cloud.google.com/go/storage"
	"github.com/CopiiDeco/statusok/database"
	"github.com/CopiiDeco/statusok/notify"
	"github.com/CopiiDeco/statusok/requests"
	"github.com/codegangsta/cli"
)

type configParser struct {
	NotifyWhen    NotifyWhen               `json:"notifyWhen"`
	Requests      []requests.RequestConfig `json:"requests"`
	Notifications notify.NotificationTypes `json:"notifications"`
	Database      database.DatabaseTypes   `json:"database"`
	Concurrency   int                      `json:"concurrency"`
	Port          int                      `json:"port"`
}

type NotifyWhen struct {
	MeanResponseCount int `json:"meanResponseCount"`
	ErrorCount        int `json:"errorCount"`
}

func handler(w http.ResponseWriter, r *http.Request) {
	log.Print("Hello world received a request.")
	target := os.Getenv("TARGET")
	if target == "" {
		target = "World"
	}
	fmt.Fprintf(w, "Hello %s!\n", target)
}

func read(client *storage.Client, bucket, object string) ([]byte, error) {
	ctx := context.Background()
	// [START download_file]
	rc, err := client.Bucket(bucket).Object(object).NewReader(ctx)
	if err != nil {
		return nil, err
	}
	defer rc.Close()

	data, err := ioutil.ReadAll(rc)
	if err != nil {
		return nil, err
	}
	return data, nil
	// [END download_file]
}

func bucketReader(bucket string, filename string) (data []byte, err error) {
	ctx := context.Background()
	// Creates a client.
	client, err := storage.NewClient(ctx)
	log.Print("Reading the bucket")
	if err != nil {
		log.Fatalf("Failed to create client: %v", err)
	}
	data, err = read(client, bucket, filename)
	if err != nil {
		log.Fatalf("Cannot read object: %v", err)
	}

	log.Print("Config obtained from the bucket")
	return data, nil
}

func main() {

	log.Print("Starting the app")
	//Cli tool setup to get config file path from parameters
	app := cli.NewApp()
	app.Name = "StatusOk"
	app.Usage = "Monitor your website.Get notifications when its down"

	app.Flags = []cli.Flag{
		cli.StringFlag{
			Name:  "config",
			Value: "config.json",
			Usage: "location of config file",
		},
		cli.StringFlag{
			Name:  "log",
			Value: "",
			Usage: "file to save logs",
		},
	}

	app.Action = func(c *cli.Context) {
		bucket := os.Getenv("BUCKET")
		configFileName := os.Getenv("CONFIG_FILE_NAME")

		config, err := bucketReader(bucket, configFileName)
		if err == nil {
			log.Print("Entering the start sequence")
			configString := string(config)
			if configString != "" {
				log.Print("Checks passed starting monitoring")
				err := startMonitoring(configString, "")
				if err != nil {
					log.Fatal(err)
				}
			} else {
				log.Fatal("Config not present\nPlease give correct configuration using JSON_CONFIG env variable")
			}
		} else {
			fmt.Errorf("Error while fetching the file from the bucket : %s", err)
		}
	}

	//Run as cli app
	app.Run(os.Args)

}

func startMonitoring(configFile string, logFileName string) error {

	var config configParser
	log.Print("Parsing the config")
	if err := json.Unmarshal([]byte(configFile), &config); err != nil {
		return fmt.Errorf("Error parsing config file .Please check format of the file, parse error: %s", err.Error())
	}
	fmt.Println("parsed ", config.Notifications.Dingding)
	//setup different notification clients
	notify.AddNew(config.Notifications)
	//Send test notifications to all the notification clients
	notify.SendTestNotification()

	//Create unique ids for each request date given in config file
	reqs, ids := validateAndCreateIdsForRequests(config.Requests)

	//Set up and initialize databases
	database.AddNew(config.Database)
	database.Initialize(ids, config.NotifyWhen.MeanResponseCount, config.NotifyWhen.ErrorCount)

	//Initialize and start monitoring all the apis
	requests.RequestsInit(reqs, config.Concurrency)
	requests.StartMonitoring()

	database.EnableLogging(logFileName)

	//Just to check StatusOk is running or not
	http.HandleFunc("/", statusHandler)

	port := os.Getenv("PORT")
	if port == "" {
		//Default port
		http.ListenAndServe(":8080", nil)
	} else {
		//if port is mentioned in config file
		http.ListenAndServe(":"+port, nil)
	}
	return nil
}

//Currently just tells status ok is running
//Planning to display useful information in future
func statusHandler(w http.ResponseWriter, r *http.Request) {
	io.WriteString(w, "StatusOk is running \n Planning to display useful information in further releases")
}

//Tells whether a file exits or not
func fileExists(name string) bool {
	if _, err := os.Stat(name); err != nil {
		if os.IsNotExist(err) {
			return false
		}
	}
	return true
}

func logFilePathValid(name string) bool {
	f, err := os.OpenFile(name, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
	defer f.Close()
	if err != nil {
		return false
	}

	return true
}

//checks whether each request in config file has valid data
//Creates unique ids for each request using math/rand
func validateAndCreateIdsForRequests(reqs []requests.RequestConfig) ([]requests.RequestConfig, map[int]int64) {
	source := rand.NewSource(time.Now().UnixNano())
	random := rand.New(source)

	//an array of ids used by database pacakge to calculate mean response time and send notifications
	ids := make(map[int]int64, 0)

	//an array of new requests data after updating the ids
	newreqs := make([]requests.RequestConfig, 0)

	for i, requestConfig := range reqs {
		validateErr := requestConfig.Validate()
		if validateErr != nil {
			println("\nInvalid Request data in config file for Request #", i, " ", requestConfig.Url)
			println("Error:", validateErr.Error())
			os.Exit(3)
		}

		//Set a random value as id
		randInt := random.Intn(1000000)
		ids[randInt] = requestConfig.ResponseTime
		requestConfig.SetId(randInt)
		newreqs = append(newreqs, requestConfig)
	}

	return newreqs, ids
}
