// Package phabricator provides Phabricator endpoint discovery
// and helps you consume *.search endpoints.
// It hides away all the ugly details of Phabricator API:
// Response pagination
// API errors
// Response parsing
// Creating POST requests
package phabricator

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"
	"time"

	// https://github.com/Sirupsen/logrus
	log "github.com/sirupsen/logrus"
)

type normRegex struct {
	from []byte
	to   []byte
}

var normalization map[string]normRegex

// Phabricator's JSON responses are absolute garbage
// so we normalize the result here - if a map-type field
// in the response is empty, phab responds with an array instead.
// I haven't found ANY way
// to make this more sane. The below rules are handcrafted,
// but it calls for a much more systematic solution.
//
// TODO: Something like reflecting on the response type
// and substituting for all empty map-types
func init() {
	normalization = map[string]normRegex{
		"maniphest.search": normRegex{
			from: []byte(`"boards":[]`),
			to:   []byte(`"boards":{}`),
		},
		"conduit.query": normRegex{
			from: []byte(`"params":[]`),
			to:   []byte(`"params":{}`),
		},
	}
}

var logger = log.New()

const (
	// Phabricator paginates responses in pages of 100 results.
	maxBufferedResponses = 100
)

type endpointInfo struct {
	Description string            `json:"description"`
	Params      map[string]string `json:"params"`
	Return      string            `json:"return"`
}

// PhabResultCallback specified the function type of the callback used by Phabricator.Call.
// It is given an IN channel to feed the callback JSON and out channel
// that is also returned to the caller of Phabricator.Call.
type PhabResultCallback func(out chan<- interface{}, in <-chan json.RawMessage) error

// EndpointArguments should be a struct
// that represents the postform data passed to
// phabricator endpoints
type EndpointArguments interface{}

func (ei endpointInfo) String() string {
	var builder strings.Builder
	// Gosec will complain about unchecked WriteString errors
	// but if you look at the code, WriteString always returns nil... (Golang 1.11)
	builder.WriteString(fmt.Sprintf("\tDescription: %s\n", ei.Description))
	builder.WriteString(fmt.Sprintf("\tParams:\n"))
	for param, desc := range ei.Params {
		builder.WriteString(fmt.Sprintf("\t\t%s: %s\n", param, desc))
	}
	builder.WriteString(fmt.Sprintf("\tReturn:\n\t\t%s\n", ei.Return))
	return builder.String()
}

type conduitQueryResponse struct {
	Result    map[string]endpointInfo `json:"result"`
	ErrorCode string                  `json:"error_code"`
	ErrorInfo string                  `json:"error_info"`
}

func (cr conduitQueryResponse) String() string {
	var builder strings.Builder
	for endpoint, details := range cr.Result {
		builder.WriteString(fmt.Sprintf("%s:\n%s", endpoint, details))
	}
	return builder.String()
}

// Phabricator wraps around the API calls
// bound to a single API root
type Phabricator struct {
	apiEndpoint     *url.URL
	apiToken        string
	apiInfo         map[string]endpointInfo
	searchEndpoints map[string]searchEndpointCallback
	editEndpoints   map[string]editEndpointCallback
	client          *http.Client
	timeout         time.Duration
}

func (p *Phabricator) postRequest(ctx context.Context, endpoint, postData string) ([]byte, error) {
	req, err := http.NewRequest("POST", endpoint, strings.NewReader(postData))
	req = req.WithContext(ctx)
	// We delay error reporting to the caller, which has
	// more human-readable data to report
	if err != nil {
		return nil, err
	}
	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")
	resp, err := p.client.Do(req)
	if err != nil {
		return nil, err
	}
	logger.WithFields(log.Fields{
		"status":   resp.Status,
		"method":   resp.Request.Method,
		"endpoint": endpoint,
	}).Info("HTTP Request")
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		logger.WithFields(log.Fields{
			"error": err,
		}).Error("Failed to read HTTP response")
		return nil, err
	}
	return body, nil
}

func (p *Phabricator) loadEndpoints(einfo map[string]endpointInfo) {
	p.searchEndpoints = make(map[string]searchEndpointCallback)
	p.editEndpoints = make(map[string]editEndpointCallback)
	for endpoint := range einfo {
		if strings.HasSuffix(endpoint, ".search") {
			logger.WithFields(log.Fields{
				"endpoint": endpoint,
			}).Debug("Defining callback for a search endpoint")
			p.searchEndpoints[endpoint] = p.searchEndpointHandler
		} else if strings.HasSuffix(endpoint, ".edit") {
			logger.WithFields(log.Fields{
				"endpoint": endpoint,
			}).Debug("Defining callback for an edit endpoint")
			p.editEndpoints[endpoint] = p.editEndpointHandler
		} else {
			logger.WithFields(log.Fields{
				"endpoint": endpoint,
			}).Warn("Endpoint not supported yet - skipping")
		}
	}
}

func (p *Phabricator) queryEndpoints() (map[string]endpointInfo, error) {
	endpoint := "conduit.query"
	path, _ := url.Parse(endpoint)
	phabConduitQuery := p.apiEndpoint.ResolveReference(path)
	data := url.Values{"api.token": {p.apiToken}}
	resp, err := p.client.PostForm(phabConduitQuery.String(), data)
	if err != nil {
		logger.WithFields(log.Fields{
			"error":    err,
			"endpoint": phabConduitQuery.String(),
		}).Error("HTTP Request failed")
		return nil, err
	}
	logger.WithFields(log.Fields{
		"status":   resp.Status,
		"method":   resp.Request.Method,
		"endpoint": phabConduitQuery.String(),
	}).Info("HTTP Request")
	defer resp.Body.Close()
	var conduitAPI conduitQueryResponse
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		logger.WithFields(log.Fields{
			"error": err,
		}).Error("Failed to read HTTP response")
		return nil, err
	}
	norm, exists := normalization[endpoint]
	if exists {
		body = bytes.Replace(body, norm.from, norm.to, -1)
	}

	err = json.Unmarshal(body, &conduitAPI)
	if err != nil {
		logger.WithFields(log.Fields{
			"error": err,
		}).Error("Failed to decode JSON")
		return nil, err
	}
	if conduitAPI.ErrorCode != "" {
		logger.WithFields(log.Fields{
			"PhabricatorErrorCode": conduitAPI.ErrorCode,
			"PhabricatorErrorInfo": conduitAPI.ErrorInfo,
		}).Error("Invalid Phabricator Request")
		errMsg := fmt.Sprintf("[%s] %s", conduitAPI.ErrorCode, conduitAPI.ErrorInfo)
		return nil, Error{errMsg}
	}
	return conduitAPI.Result, nil
}

// PhabOptions allows you to config the log level
// and request timeouts for Phabricator
type PhabOptions struct {
	LogLevel string
	Timeout  time.Duration
	Out      io.Writer
}

// Init discovers known API endpoints and defines
// appropriate callbacks
func (p *Phabricator) Init(endpoint, token string, opts *PhabOptions) error {
	loglevel := "info"
	p.timeout = 10 * time.Second
	if opts != nil {
		if opts.LogLevel != "" {
			loglevel = opts.LogLevel
		}
		if opts.Timeout > 0 {
			p.timeout = opts.Timeout
		} else {
			return errors.New("Negative timeout specified")
		}
		if opts.Out != nil {
			logger.SetOutput(opts.Out)
		}
	}
	p.client = &http.Client{Timeout: p.timeout}

	level, err := log.ParseLevel(loglevel)
	if err != nil {
		return err
	}
	logger.SetLevel(level)
	// Display file & line info - needs a relatively new version of logrus
	logger.SetReportCaller(true)

	api, err := url.Parse(endpoint)
	if err != nil {
		logger.WithFields(log.Fields{
			"url":   endpoint,
			"error": err,
		}).Error("Unable to parse the API URL")
		return Error{err.Error()}
	}

	logger.WithFields(log.Fields{
		"url":      endpoint,
		"loglevel": loglevel,
		"timeout":  p.timeout,
	}).Debug("Initializing a Phabricator instance")

	p.apiEndpoint = api
	p.apiToken = token
	ep, err := p.queryEndpoints()
	if err != nil {
		return err
	}

	p.apiInfo = ep
	p.loadEndpoints(ep)

	return nil
}
