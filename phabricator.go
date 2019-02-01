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
	"reflect"
	"strings"
	"sync"
	"time"

	// https://godoc.org/github.com/google/go-querystring/query
	query "github.com/google/go-querystring/query"

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

type baseResponse struct {
	Result struct {
		Data   []json.RawMessage `json:"data"`
		Maps   json.RawMessage   `json:"maps"`  // unused
		Query  json.RawMessage   `json:"query"` // unused
		Cursor struct {
			Limit  int             `json:"limit"`
			After  string          `json:"after"`
			Before string          `json:"before"`
			Order  json.RawMessage `json:"order"` //unused
		} `json:"cursor"`
	} `json:"result"`
	ErrorCode string `json:"error_code"`
	ErrorInfo string `json:"error_info"`
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

type endpointCallback func(ctx context.Context, endpoint string, params endpointInfo, arguments EndpointArguments, typ reflect.Type) (<-chan interface{}, <-chan error)

// Phabricator wraps around the API calls
// bound to a single API root
type Phabricator struct {
	apiEndpoint *url.URL
	apiToken    string
	apiInfo     map[string]endpointInfo
	endpoints   map[string]endpointCallback
	client      *http.Client
	timeout     time.Duration
}

// Call ENDPOINT with ARGUMENTS, using the callback CB to
// pass results to the caller
func (p *Phabricator) Call(ctx context.Context, endpoint string, arguments EndpointArguments, typ interface{}) (<-chan interface{}, <-chan error) {
	handler, defined := p.endpoints[endpoint]
	if !defined {
		errMsg := "No callback defined for endpoint"

		logger.WithFields(log.Fields{
			"endpoint": endpoint,
		}).Error(errMsg)
		return nil, nil
	}
	t := reflect.TypeOf(typ) // TODO pointer types
	return handler(ctx, endpoint, p.apiInfo[endpoint], arguments, t)
}

func (p *Phabricator) postRequest(endpoint, postData string) ([]byte, error) {
	req, err := http.NewRequest("POST", endpoint, strings.NewReader(postData))
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

func (p *Phabricator) endpointHandler(ctx context.Context, endpoint string, einfo endpointInfo, arguments EndpointArguments, typ reflect.Type) (<-chan interface{}, <-chan error) {
	queryArgs, err := query.Values(arguments)
	resultChan := make(chan interface{}, maxBufferedResponses)
	errorChan := make(chan error)
	dataChan := make(chan json.RawMessage, maxBufferedResponses)

	if err != nil {
		logger.WithFields(log.Fields{
			"error":    err,
			"endpoint": endpoint,
		}).Error("Failed to encode endpoint query arguments")
		errorChan <- err
		return resultChan, errorChan
	}
	data := queryArgs.Encode()
	data = fmt.Sprintf("%s=%s&%s", "api.token", p.apiToken, data)
	path, _ := url.Parse(endpoint)
	fullEndpoint := p.apiEndpoint.ResolveReference(path).String()
	go func() {
		var wg sync.WaitGroup
		defer close(dataChan)
		after := ""
		for {
			select {
			case <-ctx.Done():
				return
			default:
			}
			// Fire off the next response as soon as we know the value of "after"
			// from the previous one
			after = func(after string) string {
				postData := data
				if after != "" {
					postData = fmt.Sprintf("%s&after=%s", postData, after)
				}

				body, err := p.postRequest(fullEndpoint, postData)
				if err != nil {
					logger.WithFields(log.Fields{
						"error":     err,
						"post_data": queryArgs.Encode(),
						"endpoint":  endpoint,
						"after":     after,
					}).Error("Request to Phabricator failed")
					errorChan <- err
					return ""
				}
				norm, exists := normalization[endpoint]
				if exists {
					body = bytes.Replace(body, norm.from, norm.to, -1)
				}
				var baseResp baseResponse
				err = json.Unmarshal(body, &baseResp)
				if err != nil {
					logger.WithFields(log.Fields{
						"error": err,
					}).Error("Failed to decode JSON")
					errorChan <- err
					return ""
				}
				if baseResp.ErrorCode != "" {
					logger.WithFields(log.Fields{
						"PhabricatorErrorCode": baseResp.ErrorCode,
						"PhabricatorErrorInfo": baseResp.ErrorInfo,
					}).Error("Invalid Phabricator Request")
					errorChan <- err
					return ""
				}
				wg.Add(1)
				go func() {
					defer wg.Done()
					for _, m := range baseResp.Result.Data {
						select {
						case <-ctx.Done():
							return
						default:
							dataChan <- m
						}
					}
				}()

				return baseResp.Result.Cursor.After
			}(after)
			if after == "" {
				break
			}
		}
		wg.Wait()
	}()
	go func() {
		defer close(resultChan)
		for jsonData := range dataChan {
			t := reflect.New(typ).Interface()
			err := json.Unmarshal(jsonData, t)
			if err != nil {
				errorChan <- err
				continue
			}
			resultChan <- t
		}
	}()
	return resultChan, errorChan
}

func (p *Phabricator) loadEndpoints(einfo map[string]endpointInfo) {
	p.endpoints = make(map[string]endpointCallback)
	for endpoint := range einfo {
		if !strings.HasSuffix(endpoint, ".search") {
			logger.WithFields(log.Fields{
				"endpoint": endpoint,
			}).Warn("Endpoint not supported yet - skipping")
			continue
		}
		logger.WithFields(log.Fields{
			"endpoint": endpoint,
		}).Debug("Defining callback for endpoint")
		p.endpoints[endpoint] = p.endpointHandler
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
