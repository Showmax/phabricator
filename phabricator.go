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

type EndpointInfo struct {
	Description string            `json:"description"`
	Params      map[string]string `json:"params"`
	Return      string            `json:"return"`
}

type BaseResponse struct {
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

func (ei EndpointInfo) String() string {
	var builder strings.Builder
	builder.WriteString(fmt.Sprintf("\tDescription: %s\n", ei.Description))
	builder.WriteString(fmt.Sprintf("\tParams:\n"))
	for param, desc := range ei.Params {
		builder.WriteString(fmt.Sprintf("\t\t%s: %s\n", param, desc))
	}
	builder.WriteString(fmt.Sprintf("\tReturn:\n\t\t%s\n", ei.Return))
	return builder.String()
}

type ConduitQueryResponse struct {
	Result    map[string]EndpointInfo `json:"result"`
	ErrorCode string                  `json:"error_code"`
	ErrorInfo string                  `json:"error_info"`
}

func (cr ConduitQueryResponse) String() string {
	var builder strings.Builder
	for endpoint, details := range cr.Result {
		builder.WriteString(fmt.Sprintf("%s:\n%s", endpoint, details))
	}
	return builder.String()
}

type EndpointCallback func(endpoint string, params EndpointInfo, arguments EndpointArguments, cb PhabResultCallback) (<-chan interface{}, error)

type Phabricator struct {
	ApiEndpoint *url.URL
	ApiToken    string
	ApiInfo     map[string]EndpointInfo
	endpoints   map[string]EndpointCallback
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
	resp, err := callback(endpoint, p.ApiInfo[endpoint], arguments, cb)
	return resp, err
}

func (p *Phabricator) loadEndpoints(einfo map[string]EndpointInfo) error {
	p.endpoints = make(map[string]EndpointCallback)
	timeout_duration := time.Duration(5) * time.Second
	for endpoint := range einfo {
		eh := func(endpoint string, einfo EndpointInfo, arguments EndpointArguments, cb PhabResultCallback) (<-chan interface{}, error) {
			query_args, err := query.Values(arguments)
			if err != nil {
				logger.WithFields(log.Fields{
					"error":    err,
					"endpoint": endpoint,
				}).Error("Failed to encode endpoint query arguments")
				return nil, err
			}
			data := query_args.Encode()
			data = fmt.Sprintf("%s=%s&%s", "api.token", p.ApiToken, data)
			data_chan := make(chan json.RawMessage, MAX_BUFFERED_RESPONSES)
			path, _ := url.Parse(endpoint)
			ep := p.ApiEndpoint.ResolveReference(path)
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
					client := http.Client{}
					client.Timeout = timeout_duration
					resp, err := client.Do(req)
					logger.WithFields(log.Fields{
						"status":   resp.Status,
						"method":   resp.Request.Method,
						"endpoint": ep.String(),
					}).Info("HTTP Request")
					if err != nil {
						logger.WithFields(log.Fields{
							"status":   resp.Status,
							"method":   resp.Request.Method,
							"endpoint": ep.String(),
						}).Error("HTTP Request failed")
						return
					}
					defer resp.Body.Close() // TODO does this really work?
					dec := json.NewDecoder(resp.Body)
					dec.DisallowUnknownFields()
					var base_resp BaseResponse
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

func (p *Phabricator) queryEndpoints() (map[string]EndpointInfo, error) {
	path, err := url.Parse("conduit.query")
	phab_conduit_query := p.ApiEndpoint.ResolveReference(path)
	data := url.Values{"api.token": {p.ApiToken}}
	resp, err := http.PostForm(phab_conduit_query.String(), data)
	logger.WithFields(log.Fields{
		"status":   resp.Status,
		"method":   resp.Request.Method,
		"endpoint": phab_conduit_query.String(),
	}).Info("HTTP Request")
	if err != nil {
		logger.WithFields(log.Fields{
			"error":    err,
			"status":   resp.Status,
			"method":   resp.Request.Method,
			"endpoint": phab_conduit_query.String(),
		}).Error("HTTP Request failed")
		return nil, err
	}
	defer resp.Body.Close()
	var conduit_api ConduitQueryResponse
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

func (p *Phabricator) Init(endpoint, token string) error {
	logger.SetLevel(log.InfoLevel)

	logger.WithFields(log.Fields{
		"url": endpoint,
	}).Debug("Initializing a Phabricator instance")

	api, err := url.Parse(endpoint)
	if err != nil {
		logger.WithFields(log.Fields{
			"url":   endpoint,
			"error": err,
		}).Error("Unable to parse the API URL")
		return PhabricatorError{err.Error()}
	}
	p.ApiEndpoint = api
	p.ApiToken = token
	ep, err := p.queryEndpoints()
	if err != nil {
		return err
	}
	p.ApiInfo = ep

	err = p.loadEndpoints(ep)
	if err != nil {
		return err
	}

	return nil
}
