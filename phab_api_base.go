package phabricator

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"strings"

	// https://godoc.org/github.com/google/go-querystring/query
	query "github.com/google/go-querystring/query"
)

type PhabricatorError struct {
	err string
}

func (pe PhabricatorError) Error() string {
	return pe.err
}

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

// all data must be marshaled into a JSON-string
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
		err_msg := fmt.Sprintf("No callback defined for endpoint '%s'", endpoint)
		log.Print(err_msg)
		return nil, PhabricatorError{err_msg}
	}
	resp, err := callback(endpoint, p.ApiInfo[endpoint], arguments, cb)
	return resp, err
}

func (p *Phabricator) loadEndpoints(einfo map[string]EndpointInfo) error {
	p.endpoints = make(map[string]EndpointCallback)
	for endpoint := range einfo {
		eh := func(endpoint string, einfo EndpointInfo, arguments EndpointArguments, cb PhabResultCallback) (<-chan interface{}, error) {
			log.Print(endpoint)
			query_args, _ := query.Values(arguments)
			data := query_args.Encode()
			data = fmt.Sprintf("%s=%s&%s", "api.token", p.ApiToken, data)
			data_chan := make(chan json.RawMessage, MAX_BUFFERED_RESPONSES)
			path, _ := url.Parse(endpoint)
			ep := p.ApiEndpoint.ResolveReference(path)
			go func() {
				after := ""
				for {
					post_data := data
					if after != "" {
						post_data = fmt.Sprintf("%s&after=%s", post_data, after)
					}
					log.Print(post_data)

					req, err := http.NewRequest("POST", ep.String(), strings.NewReader(post_data))
					req.Header.Add("Content-Type", "application/x-www-form-urlencoded")
					client := http.Client{}
					resp, err := client.Do(req)
					log.Printf("[%s] %s %s", resp.Status, resp.Request.Method, ep.String())
					if err != nil {
						log.Fatal(err)
					}
					defer resp.Body.Close()
					dec := json.NewDecoder(resp.Body)
					dec.DisallowUnknownFields()
					var base_resp BaseResponse
					err = dec.Decode(&base_resp)
					if err != nil {
						log.Fatal(err)
					}
					if base_resp.ErrorCode != "" {
						log.Fatal(base_resp.ErrorCode, base_resp.ErrorInfo)
					}
					for _, m := range base_resp.Result.Data {
						data_chan <- m
					}

					after = base_resp.Result.Cursor.After
					if after == "" {
						break
					}
				}
				close(data_chan)
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
	log.Printf("[%s] %s %s", resp.Status, resp.Request.Method, phab_conduit_query.String())
	if err != nil {
		log.Fatal(err)
	}
	var conduit_api ConduitQueryResponse
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Fatal(err)
	}
	// Phabricator's JSON responses are absolute garbage
	// so we normalize the result here. I haven't found ANY way
	// to make this more sane
	body = bytes.Replace(body, []byte(`"params":[]`), []byte(`"params":{}`), -1)

	err = json.Unmarshal(body, &conduit_api)
	//TODO return err
	if err != nil {
		log.Fatal(err)
	}
	if conduit_api.ErrorCode != "" {
		err_msg := fmt.Sprintf("[%s] %s", conduit_api.ErrorCode, conduit_api.ErrorInfo)
		log.Printf(err_msg)
		return nil, PhabricatorError{err_msg}
	}
	return conduit_api.Result, nil
}

func (p *Phabricator) Init(endpoint, token string) {
	api, err := url.Parse(endpoint)
	if err != nil {
		log.Fatal(err)
	}
	p.ApiEndpoint = api
	p.ApiToken = token
	ep, err := p.queryEndpoints()
	if err != nil {
		log.Fatal(err)
	}
	p.ApiInfo = ep
	err = p.loadEndpoints(ep)
	if err != nil {
		log.Fatal(err)
	}
}
