package phabricator

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"reflect"
	"sync"

	// https://github.com/Sirupsen/logrus
	log "github.com/sirupsen/logrus"

	// https://godoc.org/github.com/google/go-querystring/query
	query "github.com/google/go-querystring/query"
)

type searchEndpointCallback func(ctx context.Context, endpoint string, params endpointInfo, arguments EndpointArguments, typ reflect.Type) <-chan interface{}

type baseSearchResponse struct {
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

// Call ENDPOINT with ARGUMENTS, using the callback CB to
// pass results to the caller
func (p *Phabricator) CallSearch(ctx context.Context, endpoint string, arguments EndpointArguments, typ interface{}) <-chan interface{} {
	handler, defined := p.searchEndpoints[endpoint]
	if !defined {
		errMsg := "No callback defined for endpoint"

		logger.WithFields(log.Fields{
			"endpoint": endpoint,
		}).Error(errMsg)
		return nil
	}
	t := reflect.TypeOf(typ) // TODO pointer types
	return handler(ctx, endpoint, p.apiInfo[endpoint], arguments, t)
}

func (p *Phabricator) searchEndpointHandler(ctx context.Context, endpoint string, einfo endpointInfo, arguments EndpointArguments, typ reflect.Type) <-chan interface{} {
	queryArgs, err := query.Values(arguments)
	resultChan := make(chan interface{}, maxBufferedResponses)
	dataChan := make(chan json.RawMessage, maxBufferedResponses)

	if err != nil {
		logger.WithFields(log.Fields{
			"error":    err,
			"endpoint": endpoint,
		}).Error("Failed to encode endpoint query arguments")
		resultChan <- err
		return resultChan
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
				logger.Debug("Context cancellation")
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

				body, err := p.postRequest(ctx, fullEndpoint, postData)
				if err != nil {
					logger.WithFields(log.Fields{
						"error":     err,
						"post_data": queryArgs.Encode(),
						"endpoint":  endpoint,
						"after":     after,
					}).Error("Request to Phabricator failed")
					resultChan <- err
					return ""
				}
				norm, exists := normalization[endpoint]
				if exists {
					body = bytes.Replace(body, norm.from, norm.to, -1)
				}
				var baseResp baseSearchResponse
				err = json.Unmarshal(body, &baseResp)
				if err != nil {
					logger.WithFields(log.Fields{
						"error": err,
					}).Error("Failed to decode JSON")
					resultChan <- err
					return ""
				}
				if baseResp.ErrorCode != "" {
					logger.WithFields(log.Fields{
						"PhabricatorErrorCode": baseResp.ErrorCode,
						"PhabricatorErrorInfo": baseResp.ErrorInfo,
					}).Error("Invalid Phabricator Request")
					resultChan <- err
					return ""
				}
				wg.Add(1)
				go func() {
					defer wg.Done()
					for _, m := range baseResp.Result.Data {
						select {
						case <-ctx.Done():
							logger.Debug("Context cancellation")
							return
						case dataChan <- m:
							logger.Debug("Sending a Phabricator object for processing")
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
				logger.WithError(err).Error("Failed to convert JSON to user-supplied type")
				resultChan <- err
				continue
			}
			resultChan <- t
		}
	}()
	return resultChan
}
