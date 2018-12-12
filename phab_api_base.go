package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"strings"
)

type EndpointInfo struct {
	Description string            `json:"description"`
	Params      map[string]string `json:"params"`
	Return      string            `json:"return"`
}

type BaseResponseData struct {
	Data   []json.RawMessage `json:"data"`
	Maps   json.RawMessage   `json:"maps"`  // unused
	Query  json.RawMessage   `json:"query"` // unused
	Cursor struct {
		Limit  int             `json:"limit"`
		After  string          `json:"after"`
		Before string          `json:"before"`
		Order  json.RawMessage `json:"order"` //unused
	} `json:"cursor"`
}

type BaseResponse struct {
	Result    BaseResponseData `json:"result"`
	ErrorCode string           `json:"error_code"`
	ErrorInfo string           `json:"error_info"`
}

type PhabResultCallback func(chan<- interface{}, <-chan json.RawMessage) error

// all data must be marshaled into a JSON-string
type EndpointArguments map[string]string

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

type TicketAttachmentColumn struct {
	Id   int    `json:"id"`
	Phid string `json:"phid"`
	Name string `json:"name"`
}

type TicketAttachmentBoard struct {
	Columns []TicketAttachmentColumn `json:"columns"`
}

type Ticket struct {
	Id     int    `json:"id"`
	Type   string `json:"type"`
	Phid   string `json:"phid"`
	Fields struct {
		Name        string `json:"name"`
		Description struct {
			Raw string `json:"raw"`
		} `json:"description"`
		AuthorPHID string `json:"authorPHID"`
		OwnerPHID  string `json:"ownerPHID"`
		Status     struct {
			Value string `json:"value"`
			Name  string `json:"name"`
			Color string `json:"color"`
		} `json:"status"`
		Priority struct {
			Value       int     `json:"value"`
			Subpriority float64 `json:"subpriority"`
			Name        string  `json:"name"`
			Color       string  `json:"color"`
		}
		Points       string `json:"points"`
		Subtype      string `json:"subtype"`
		CloserPHID   string `json:"closerPHID"`
		DateClosed   int    `json:"dateClosed"`
		SpacePHID    string `json:"spacePHID"`
		DateCreated  int    `json:"dateCreated"`
		DateModified int    `json:"dateModified"`
		Policy       struct {
			View     string `json:"view"`
			Interact string `json:"interact"`
			Edit     string `json:"edit"`
		}
		Attachments struct {
			Columns struct {
				Boards map[string]TicketAttachmentBoard `json:"boards"`
			} `json:"columns"`
			Subscribers struct {
				SubscriberPHIDs    []string `json:"subscriberPHIDs"`
				SubscriberCount    int      `json:"subscriberCount"`
				ViewerIsSubscribed bool     `json:"viewerIsSubscribed"`
			} `json:"subscribers"`
			Projects struct {
				ProjectPHIDs []string `json:"projectPHIDs"`
			}
		} `json:"attachments"`
	}
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
	resp, err := p.endpoints[endpoint](endpoint, p.ApiInfo[endpoint], arguments, cb)
	return resp, err
}

func (p *Phabricator) loadEndpoints(einfo map[string]EndpointInfo) error {
	p.endpoints = make(map[string]EndpointCallback)
	for endpoint := range einfo {
		eh := func(endpoint string, einfo EndpointInfo, arguments EndpointArguments, cb PhabResultCallback) (<-chan interface{}, error) {
			//TODO pagination
			//TODO actually pass the arguments
			data := url.Values{"api.token": {p.ApiToken}}
			for key, val := range arguments {
				_, ok := einfo.Params[key]
				if !ok {
					log.Fatal("%s: Not a valid param for %s", key, endpoint)
				}
				data.Add(key, val) // naive, do types
			}
			data_chan := make(chan json.RawMessage) // TODO make buffered 100?
			path, _ := url.Parse(endpoint)
			ep := p.ApiEndpoint.ResolveReference(path)
			go func() {
				for {
					log.Print("POST ", ep.String())
					resp, err := http.PostForm(ep.String(), data)
					if err != nil {
						log.Fatal(err)
					}
					body, err := ioutil.ReadAll(resp.Body)
					if err != nil {
						log.Fatal(err)
					}
					var base_resp BaseResponse
					// TODO use NewDecoder?
					err = json.Unmarshal(body, &base_resp)
					if err != nil {
						log.Fatal(err)
					}
					if base_resp.ErrorCode != "" {
						log.Fatal(base_resp.ErrorInfo)
					}
					for _, m := range base_resp.Result.Data {
						data_chan <- m
					}

					after := base_resp.Result.Cursor.After
					if len(after) == 0 {
						break
					}
					data.Set("after", after)
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
	log.Print("POST ", phab_conduit_query)
	resp, err := http.PostForm(phab_conduit_query.String(), data)
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

func (t *Ticket) String() string {
	return fmt.Sprintf("T%d: %s", t.Id, t.Fields.Name)
}

func TicketResponseCallback(tickets chan<- interface{}, data <-chan json.RawMessage) error {
	for json_data := range data {
		var t Ticket
		err := json.Unmarshal(json_data, &t)
		if err != nil {
			log.Fatal(err)
		}
		tickets <- t
	}
	close(tickets)
	return nil // TODO dont ignore
}

func main() {
	log.SetFlags(log.Lshortfile)
	phab_conduit_api := "https://phabricator.showmax.cc/api/"
	token := ""
	var phab Phabricator
	phab.Init(phab_conduit_api, token)
	// TODO should accept a user-specified callback for parsing
	// and return a channel
	out_chan, err := phab.Call("maniphest.search", EndpointArguments{}, TicketResponseCallback)
	for ticket := range out_chan {
		//fmt.Sprintf("T%d\n", ticket.(Ticket).Id)
		t, _ := ticket.(Ticket)
		fmt.Println(&t)
	}
	if err != nil {
		log.Fatal(err)
	}
}
