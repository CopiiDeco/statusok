package requests

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"
	"github.com/tcnksm/go-httpstat"

	"github.com/CopiiDeco/statusok/database"
)

var Client *http.Client

func Init() {
	Client = &http.Client{Transport: &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}}
}

var (
	RequestsList   []RequestConfig
	requestChannel chan RequestConfig
	throttle       chan int
)

const (
	ContentType     = "Content-Type"
	ContentLength   = "Content-Length"
	FormContentType = "application/x-www-form-urlencoded"
	JsonContentType = "application/json"

	DefaultTime         = "300s"
	DefaultResponseCode = http.StatusOK
	DefaultConcurrency  = 1
)

type OauthCredentials struct {
	ClientID     string `json:"clientId"`
	ClientSecret string `json:"clientSecret"`
	OauthServer  string `json:"oAuthServer"`
}

type OauthResponse struct {
	Error        string `json:"error"`
	ErrorMessage string `json:"error_description"`
	AccessToken  string `json:"access_token"`
	TokenType    string `json:"token_type"`
	ExpiresIn    int    `json:"expires_in"`
	Scope        string `json:"scope"`
}

type RequestConfig struct {
	Id           int
	OauthCreds   *OauthCredentials `json:"oAuthCreds"`
	Url          string            `json:"url"`
	RequestType  string            `json:"requestType"`
	Headers      map[string]string `json:"headers"`
	FormParams   map[string]string `json:"formParams"`
	UrlParams    map[string]string `json:"urlParams"`
	ResponseCode int               `json:"responseCode"`
	ResponseTime int64             `json:"responseTime"`
	CheckEvery   time.Duration     `json:"checkEvery"`
}

//Set Id for request
func (requestConfig *RequestConfig) SetId(id int) {
	requestConfig.Id = id
}

//check whether all requestConfig fields are valid
func (requestConfig *RequestConfig) Validate() error {

	if len(requestConfig.Url) == 0 {
		return errors.New("Invalid Url")
	}

	if _, err := url.Parse(requestConfig.Url); err != nil {
		return errors.New("Invalid Url")
	}

	if len(requestConfig.RequestType) == 0 {
		return errors.New("RequestType cannot be empty")
	}

	if requestConfig.ResponseTime == 0 {
		return errors.New("ResponseTime cannot be empty")
	}

	if requestConfig.ResponseCode == 0 {
		requestConfig.ResponseCode = DefaultResponseCode
	}

	if requestConfig.CheckEvery == 0 {
		defTime, _ := time.ParseDuration(DefaultTime)
		requestConfig.CheckEvery = defTime
	}

	return nil
}

//Initialize data from config file and check all requests
func RequestsInit(data []RequestConfig, concurrency int) {
	RequestsList = data

	//throttle channel is used to limit number of requests performed at a time
	if concurrency == 0 {
		throttle = make(chan int, DefaultConcurrency)
	} else {
		throttle = make(chan int, concurrency)
	}

	requestChannel = make(chan RequestConfig, len(data))

	if len(data) == 0 {
		println("\nNo requests to monitor.Please add requests to you config file")
		os.Exit(3)
	}
	//send requests to make sure every every request is valid
	println("\nSending requests to apis.....making sure everything is right before we start monitoring")
	println("Api Count: ", len(data))

	for i, requestConfig := range data {
		println("Request #", i, " : ", requestConfig.RequestType, " ", requestConfig.Url)

		//Perform request
		reqErr := PerformRequest(requestConfig, nil)

		if reqErr != nil {
			//Request Failed
			println("\nFailed !!!! Not able to perfom below request")
			println("\n----Request Deatails---")
			println("Url :", requestConfig.Url)
			println("Type :", requestConfig.RequestType)
			println("Error Reason :", reqErr.Error())
			println("\nPlease check the config file and try again")
			//Disable exit because when we monitor multiple systems one can be down when starting up
			//os.Exit(3)
		}
	}

	println("All requests tested")
}

//Start monitoring by calling createTicker method for each request
func StartMonitoring() {
	fmt.Println("\nStarted Monitoring all ", len(RequestsList), " apis .....")

	go listenToRequestChannel()

	for _, requestConfig := range RequestsList {
		go createTicker(requestConfig)
	}
}

//A time ticker writes data to request channel for every request.CheckEvery seconds
func createTicker(requestConfig RequestConfig) {

	var ticker *time.Ticker = time.NewTicker(requestConfig.CheckEvery * time.Second)
	quit := make(chan struct{})
	for {
		select {
		case <-ticker.C:
			requestChannel <- requestConfig
		case <-quit:
			ticker.Stop()
			return
		}
	}
}

//all tickers write to request channel
//here we listen to request channel and perfom each request
func listenToRequestChannel() {

	//throttle is used to limit number of requests executed at a time
	for {
		select {
		case requect := <-requestChannel:
			throttle <- 1
			go PerformRequest(requect, throttle)
		}
	}

}

func GetOauthToken(requestConfig RequestConfig,retry bool) (string, error) {
	if Client == nil {
		Init()
	}
	//Get a new Oauth token since Token credentials have been defined

	payload := strings.NewReader("client_id=" + requestConfig.OauthCreds.ClientID + "&client_secret=" + requestConfig.OauthCreds.ClientSecret + "&grant_type=client_credentials")

	req, _ := http.NewRequest("POST", requestConfig.OauthCreds.OauthServer, payload)

	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Add("cache-control", "no-cache")

	res, respErr := Client.Do(req)

	defer res.Body.Close()

	if respErr != nil {
		//Request failed . Add error info to database
		var statusCode int
		if res == nil {
			statusCode = 0
		} else {
			statusCode = res.StatusCode
		}
		go database.AddErrorInfo(database.ErrorInfo{
			Id:           requestConfig.Id,
			Url:          requestConfig.Url,
			RequestType:  requestConfig.RequestType,
			ResponseCode: statusCode,
			ResponseBody: convertResponseToString(res),
			Reason:       database.ErrDoRequest,
			OtherInfo:    respErr.Error(),
		})
		return "", respErr
	}

	data, _ := ioutil.ReadAll(res.Body)
	var oauthResponse OauthResponse
	respErr = json.Unmarshal(data, &oauthResponse)
	if respErr != nil {
		return "", respErr
	}

	if retry && oauthResponse.ExpiresIn < 2 {
		println("Oauth token expiring soon, waiting and asking a new one")
		time.Sleep(2*time.Second)
		println("Calling the new token")
		return GetOauthToken(requestConfig,false)
	}

	return "Bearer " + oauthResponse.AccessToken, respErr

}

//takes the date from requestConfig and creates http request and executes it
func PerformRequest(requestConfig RequestConfig, throttle chan int) error {
	if Client == nil {
		Init()
	}
	//Remove value from throttel channel when request is completed
	defer func() {
		if throttle != nil {
			<-throttle
		}
	}()

	var request *http.Request
	var reqErr error
	if len(requestConfig.FormParams) == 0 {
		//formParams create a request
		request, reqErr = http.NewRequest(requestConfig.RequestType,
			requestConfig.Url,
			nil)
	} else {
		if requestConfig.Headers[ContentType] == JsonContentType {
			//create a request using using formParams

			jsonBody, jsonErr := GetJsonParamsBody(requestConfig.FormParams)
			if jsonErr != nil {
				//Not able to create Request object.Add Error to Database
				go database.AddErrorInfo(database.ErrorInfo{
					Id:           requestConfig.Id,
					Url:          requestConfig.Url,
					RequestType:  requestConfig.RequestType,
					ResponseCode: 0,
					ResponseBody: "",
					Reason:       database.ErrCreateRequest,
					OtherInfo:    jsonErr.Error(),
				})

				return jsonErr
			}
			request, reqErr = http.NewRequest(requestConfig.RequestType,
				requestConfig.Url,
				jsonBody)

		} else {
			//create a request using formParams
			formParams := GetUrlValues(requestConfig.FormParams)

			request, reqErr = http.NewRequest(requestConfig.RequestType,
				requestConfig.Url,
				bytes.NewBufferString(formParams.Encode()))

			request.Header.Add(ContentLength, strconv.Itoa(len(formParams.Encode())))

			if requestConfig.Headers[ContentType] != "" {
				//Add content type to header if user doesnt mention it config file
				//Default content type application/x-www-form-urlencoded
				request.Header.Add(ContentType, FormContentType)
			}

		}
	}

	if reqErr != nil {
		//Not able to create Request object.Add Error to Database
		go database.AddErrorInfo(database.ErrorInfo{
			Id:           requestConfig.Id,
			Url:          requestConfig.Url,
			RequestType:  requestConfig.RequestType,
			ResponseCode: 0,
			ResponseBody: "",
			Reason:       database.ErrCreateRequest,
			OtherInfo:    reqErr.Error(),
		})

		return reqErr
	}

	//add url parameters to query if present
	if len(requestConfig.UrlParams) != 0 {
		println("Retrieving Oauth token")
		urlParams := GetUrlValues(requestConfig.UrlParams)
		request.URL.RawQuery = urlParams.Encode()
	}

	if requestConfig.OauthCreds != nil {

		oauthToken, reqErr := GetOauthToken(requestConfig,true)
		if reqErr != nil || oauthToken == "" {
			return errors.New(fmt.Sprintf("%v", reqErr))
		} else if oauthToken == "" {
			return errors.New(fmt.Sprintf("Couldn't retrieve the Oauth token, it was empty"))
		}
		requestConfig.Headers["Authorization"] = oauthToken
	}

	//Add headers to the request
	AddHeaders(request, requestConfig.Headers)

    // Create a httpstat powered context
    var result httpstat.Result
    ctx := httpstat.WithHTTPStat(request.Context(), &result)
    request = request.WithContext(ctx)
    start := time.Now()
	getResponse, respErr := Client.Do(request)

	elapsed := time.Since(start)

	if respErr != nil {
		//Request failed . Add error info to database
		var statusCode int
		if getResponse == nil {
			statusCode = 0
		} else {
			statusCode = getResponse.StatusCode
		}
		go database.AddErrorInfo(database.ErrorInfo{
			Id:           requestConfig.Id,
			Url:          requestConfig.Url,
			RequestType:  requestConfig.RequestType,
			ResponseCode: statusCode,
			ResponseBody: convertResponseToString(getResponse),
			Reason:       database.ErrDoRequest,
			OtherInfo:    respErr.Error(),
		})
		return respErr
	}

	defer getResponse.Body.Close()

	if getResponse.StatusCode != requestConfig.ResponseCode {
		//Response code is not the expected one .Add Error to database
		go database.AddErrorInfo(database.ErrorInfo{
			Id:           requestConfig.Id,
			Url:          requestConfig.Url,
			RequestType:  requestConfig.RequestType,
			ResponseCode: getResponse.StatusCode,
			ResponseBody: convertResponseToString(getResponse),
			Reason:       errResposeCode(getResponse.StatusCode, requestConfig.ResponseCode),
			OtherInfo:    "",
		})
		return errResposeCode(getResponse.StatusCode, requestConfig.ResponseCode)
	}
	//Request succesfull . Add infomartion to Database
	go database.AddRequestInfo(database.RequestInfo{
		Id:                   requestConfig.Id,
		Url:                  requestConfig.Url,
		RequestType:          requestConfig.RequestType,
		ResponseCode:         getResponse.StatusCode,
		ResponseTime:         elapsed.Nanoseconds() / 1000000,
		ExpectedResponseTime: requestConfig.ResponseTime,
		DnsLookupTime:        int(result.DNSLookup/time.Millisecond),
        ConnectTime:          int(result.TCPConnection/time.Millisecond),
        TlsHandshakeTime:     int(result.TLSHandshake/time.Millisecond),
        ServerProcessingTime: int(result.ServerProcessing/time.Millisecond),
	})

	return nil
}

//convert response body to string
func convertResponseToString(resp *http.Response) string {
	if resp == nil {
		return " "
	}
	buf := new(bytes.Buffer)
	_, bufErr := buf.ReadFrom(resp.Body)

	if bufErr != nil {
		return " "
	}

	return buf.String()
}

//Add header values from map to request
func AddHeaders(req *http.Request, headers map[string]string) {
	for key, value := range headers {
		req.Header.Add(key, value)
	}
}

//convert params in map to url.Values
func GetUrlValues(params map[string]string) url.Values {
	urlParams := url.Values{}
	i := 0
	for key, value := range params {
		if i == 0 {
			urlParams.Set(key, value)
		} else {
			urlParams.Add(key, value)
		}
	}

	return urlParams
}

//Creates body for request of type application/json from map
func GetJsonParamsBody(params map[string]string) (io.Reader, error) {
	data, jsonErr := json.Marshal(params)

	if jsonErr != nil {

		jsonErr = errors.New("Invalid Parameters for Content-Type application/json : " + jsonErr.Error())

		return nil, jsonErr
	}

	return bytes.NewBuffer(data), nil
}

//creates an error when response code from server is not equal to response code mentioned in config file
func errResposeCode(status int, expectedStatus int) error {
	return errors.New(fmt.Sprintf("Got Response code %v. Expected Response Code %v ", status, expectedStatus))
}
