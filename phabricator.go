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
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"
	"time"

	// https://godoc.org/github.com/google/go-querystring/query
	query "github.com/google/go-querystring/query"

	// https://github.com/Sirupsen/logrus
	log "github.com/sirupsen/logrus"
)

var logger = log.New()

const (
	// Phabricator paginates responses in pages of 100 results.
	MAX_BUFFERED_RESPONSES = 100
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

type PhabResultCallback func(chan<- interface{}, <-chan json.RawMessage) error

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

type endpointCallback func(endpoint string, params endpointInfo, arguments EndpointArguments, cb PhabResultCallback) (<-chan interface{}, error)

type Phabricator struct {
	apiEndpoint *url.URL
	apiToken    string
	apiInfo     map[string]endpointInfo
	endpoints   map[string]endpointCallback
	client      *http.Client
	timeout     time.Duration
}

func (p *Phabricator) Call(endpoint string, arguments EndpointArguments, cb PhabResultCallback) (<-chan interface{}, error) {
	callback, defined := p.endpoints[endpoint]
	if !defined {
		err_msg := "No callback defined for endpoint"

		logger.WithFields(log.Fields{
			"endpoint": endpoint,
		}).Error(err_msg)
		return nil, PhabricatorError{err_msg}
	}
	resp, err := callback(endpoint, p.apiInfo[endpoint], arguments, cb)
	return resp, err
}

func (p *Phabricator) loadEndpoints(einfo map[string]endpointInfo) error {
	p.endpoints = make(map[string]endpointCallback)
	for endpoint := range einfo {
		logger.WithFields(log.Fields{
			"endpoint": endpoint,
		}).Debug("Defining callback for endpoint")
		eh := func(endpoint string, einfo endpointInfo, arguments EndpointArguments, cb PhabResultCallback) (<-chan interface{}, error) {
			query_args, err := query.Values(arguments)
			if err != nil {
				logger.WithFields(log.Fields{
					"error":    err,
					"endpoint": endpoint,
				}).Error("Failed to encode endpoint query arguments")
				return nil, err
			}
			data := query_args.Encode()
			data = fmt.Sprintf("%s=%s&%s", "api.token", p.apiToken, data)
			data_chan := make(chan json.RawMessage, MAX_BUFFERED_RESPONSES)
			path, _ := url.Parse(endpoint)
			ep := p.apiEndpoint.ResolveReference(path)
			go func() {
				defer close(data_chan)
				after := ""
				for {
					post_data := data
					if after != "" {
						post_data = fmt.Sprintf("%s&after=%s", post_data, after)
					}

					req, err := http.NewRequest("POST", ep.String(), strings.NewReader(post_data))
					if err != nil {
						logger.WithFields(log.Fields{
							"method": endpoint,
							"data":   post_data, // TODO: This logs the API token as well :(
						}).Error("Failed to construct a HTTP request")
						return
					}
					req.Header.Add("Content-Type", "application/x-www-form-urlencoded")
					resp, err := p.client.Do(req)
					if err != nil {
						logger.WithFields(log.Fields{
							"error":    err,
							"endpoint": ep.String(),
						}).Error("HTTP Request failed")
						return
					}
					logger.WithFields(log.Fields{
						"status":   resp.Status,
						"method":   resp.Request.Method,
						"data":     post_data, // TODO: This logs the API token as well :(
						"endpoint": ep.String(),
					}).Info("HTTP Request")
					defer resp.Body.Close() // TODO does this really work?
					dec := json.NewDecoder(resp.Body)
					dec.DisallowUnknownFields()
					var base_resp baseResponse
					err = dec.Decode(&base_resp)
					if err != nil {
						logger.WithFields(log.Fields{
							"error": err,
						}).Error("Failed to decode JSON")
						return
					}
					if base_resp.ErrorCode != "" {
						logger.WithFields(log.Fields{
							"PhabricatorErrorCode": base_resp.ErrorCode,
							"PhabricatorErrorInfo": base_resp.ErrorInfo,
						}).Error("Invalid Phabricator Request")
						return
					}
					for _, m := range base_resp.Result.Data {
						data_chan <- m
					}

					after = base_resp.Result.Cursor.After
					if after == "" {
						return
					}
				}
			}()
			result_chan := make(chan interface{})
			go cb(result_chan, data_chan)
			return result_chan, nil // TODO
		}
		p.endpoints[endpoint] = eh
	}
	return nil
}

func (p *Phabricator) queryEndpoints() (map[string]endpointInfo, error) {
	path, err := url.Parse("conduit.query")
	phab_conduit_query := p.apiEndpoint.ResolveReference(path)
	data := url.Values{"api.token": {p.apiToken}}
	// TODO timeout
	resp, err := p.client.PostForm(phab_conduit_query.String(), data)
	if err != nil {
		logger.WithFields(log.Fields{
			"error":    err,
			"endpoint": phab_conduit_query.String(),
		}).Error("HTTP Request failed")
		return nil, err
	}
	logger.WithFields(log.Fields{
		"status":   resp.Status,
		"method":   resp.Request.Method,
		"endpoint": phab_conduit_query.String(),
	}).Info("HTTP Request")
	defer resp.Body.Close()
	var conduit_api conduitQueryResponse
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		logger.WithFields(log.Fields{
			"error": err,
		}).Error("Failed to read HTTP response")
		return nil, err
	}
	// Phabricator's JSON responses are absolute garbage
	// so we normalize the result here. I haven't found ANY way
	// to make this more sane
	body = bytes.Replace(body, []byte(`"params":[]`), []byte(`"params":{}`), -1)

	err = json.Unmarshal(body, &conduit_api)
	if err != nil {
		logger.WithFields(log.Fields{
			"error": err,
		}).Error("Failed to decode JSON")
		return nil, err
	}
	if conduit_api.ErrorCode != "" {
		logger.WithFields(log.Fields{
			"PhabricatorErrorCode": conduit_api.ErrorCode,
			"PhabricatorErrorInfo": conduit_api.ErrorInfo,
		}).Error("Invalid Phabricator Request")
		err_msg := fmt.Sprintf("[%s] %s", conduit_api.ErrorCode, conduit_api.ErrorInfo)
		return nil, PhabricatorError{err_msg}
	}
	return conduit_api.Result, nil
}

type PhabOptions struct {
	LogLevel string
	Timeout  time.Duration
}

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
	}
	p.client = &http.Client{Timeout: p.timeout}

	level, err := log.ParseLevel(loglevel)
	if err != nil {
		return err
	}
	logger.SetLevel(level)

	api, err := url.Parse(endpoint)
	if err != nil {
		logger.WithFields(log.Fields{
			"url":   endpoint,
			"error": err,
		}).Error("Unable to parse the API URL")
		return PhabricatorError{err.Error()}
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

	err = p.loadEndpoints(ep)
	if err != nil {
		return err
	}

	return nil
}
