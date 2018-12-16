package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"
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
		} `json:"priority"`
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
		} `json:"policy"`
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
			} `json:"projects"`
		} `json:"attachments"`
	} `json:"fields"`
}

type ProjectMember struct {
	Phid string `json:"phid"`
}

type ProjectAncestor struct {
	Id   int    `json:"id"`
	Phid string `json:"phid"`
	Name string `json:"name"`
}

type ProjectWatcher struct {
	Phid string `json:"phid"`
}

type Project struct {
	Id     int    `json:"id"`
	Type   string `json:"type"`
	Phid   string `json:"phid"`
	Fields struct {
		Name      string `json:"name"`
		Slug      string `json:"slug"`
		Milestone int    `json:"milestone"`
		Depth     int    `json:"depth"`
		Parent    struct {
			Id   int    `json:"id"`
			Phid string `json:"phid"`
			Name string `json:"name"`
		} `json:"parent"`
		Icon struct {
			Key  string `json:"key"`
			Name string `json:"name"`
			Icon string `json:"icon"`
		} `json:"icon"`
		Color struct {
			Key  string `json:"key"`
			Name string `json:"name"`
		} `json:"color"`
		SpacePHID    string `json:"spacePHID"`
		DateCreated  int    `json:"dateCreated"`
		DateModified int    `json:"dateModified"`
		Policy       struct {
			View string `json:"view"`
			Edit string `json:"edit"`
			Join string `json:"join"`
		} `json:"policy"`
		Description string `json:"description"`
		Attachments struct {
			Members struct {
				Members []ProjectMember `json:"members"`
			} `json:"members"`
			Watchers struct {
				Watchers []ProjectWatcher `json:"watchers"`
			} `json:"watchers"`
			Ancestors struct {
				Ancestors []ProjectAncestor `json:"ancestors"`
			} `json:"ancestors"`
		} `json:"attachments"`
	} `json:"fields"`
}

/*

 */
type RepositoryUri struct {
	Id     int    `json:"id"`
	Type   string `json:"type"`
	Phid   string `json:"phid"`
	Fields struct {
		RepositoryPHID string `json:"repositoryPHID"`
		Uri            struct {
			Raw        string `json:"raw"`
			Display    string `json:"display"`
			Effective  string `json:"effective"`
			Normalized string `json:"normalized"`
		} `json:"uri"`
		Io             json.RawMessage // unused
		Display        json.RawMessage
		CredentialPHID string `json:"credentialPHID"`
		Disabled       bool   `json:"disabled"`
		Builtin        struct {
			Protocol   string `json:"protocol"`
			Identifier string `json:"identifier"`
		} `json:"builtin"`
		DateCreated  int `json:"dateCreated"`
		DateModified int `json:"dateModified"`
	} `json:"fields"`
}

type RevisionReviewer struct {
	ReviewerPHID string `json:"reviewerPHID"`
	Status       string `json:"status"`
	IsBlocking   bool   `json:"isBlocking"`
	ActorPHID    string `json:"actorPHID"`
}

type Revision struct {
	Id     int    `json:"id"`
	Type   string `json:"type"`
	Phid   string `json:"phid"`
	Fields struct {
		Title      string `json:"title"`
		AuthorPHID string `json:"authorPHID"`
		Status     struct {
			Value     string `json:"value"`
			Name      string `json:"name"`
			Closed    string `json:"closed"`
			ColorAnsi string `json:"color.ansi"`
		} `json:"status"`
		RepositoryPHID string `json:"repositoryPHID"`
		DiffPHID       string `json:"diffPHID"`
		Summary        string `json:"summary"`
		TestPlan       string `json:"testPlan"`
		IsDraft        bool   `json:"isDraft"`
		HoldAsDraft    bool   `json:"holdAsDraft"`
		DateCreated    int    `json:"dateCreated"`
		DateModified   int    `json:"dateModified"`
		Policy         struct {
			View string `json:"view"`
			Edit string `json:"edit"`
		} `json:"policy"`
	} `json:"fields"`
	Attachments struct {
		Reviewers struct {
			Reviewers []RevisionReviewer `json:"reviewers"`
		} `json:"reviewers"`
		Subscribers struct {
			SubscriberPHIDs    []string `json:"subscriberPHIDs"`
			SubscriberCount    int      `json:"subscriberCount"`
			ViewerIsSubscribed bool     `json:"viewerIsSubscribed"`
		} `json:"subscribers"`
		Projects struct {
			ProjectPHIDs []string `json:"projectPHIDs"`
		} `json:"projects"`
	} `json:"attachments"`
}

type Repository struct {
	Id     int    `json:"id"`
	Type   string `json:"type"`
	Phid   string `json:"phid"`
	Fields struct {
		Name               string `json:"name"`
		Vcs                string `json:"vcs"`
		Callsign           string `json:"callsign"`
		ShortName          string `json:"shortName"`
		Status             string `json:"status"`
		IsImporting        bool   `json:"isImporting"`
		AlmanacServicePHID string `json:"almanacServicePHID"`
		SpacePHID          string `json:"spacePHID"`
		DateCreated        int    `json:"dateCreated"`
		DateModified       int    `json:"dateModified"`
		Policy             struct {
			View          string `json:"view"`
			Edit          string `json:"edit"`
			DiffusionPush string `json:"diffusion.push"`
		} `json:"policy"`
	} `json:"fields"`
	Attachments struct {
		Uris struct {
			Uris []RepositoryUri `json:"uris"`
		} `json:"uris"`
		Projects struct {
			ProjectPHIDs []string `json:"projectPHIDs"`
		} `json:"projects"`
	} `json:"attachments"`
}

type User struct {
	Id     int    `json:"id"`
	Type   string `json:"type"`
	Phid   string `json:"phid"`
	Fields struct {
		Username     string   `json:"username"`
		RealName     string   `json:"realName"`
		Roles        []string `json:"roles"`
		DateCreated  int      `json:"dateCreated"`
		DateModified int      `json:"dateModified"`
		Policy       struct {
			View string `json:"view"`
			Edit string `json:"edit"`
		} `json:"policy"`
	} `json:"fields"`
	Attachments struct {
		Availability struct {
			Value string `json:"value"`
			Until int    `json:"until"`
			Name  string `json:"name"`
			Color string `json:"color"`
		} `json:"availability"`
	} `json:"attachments"`
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
			//TODO pagination
			//TODO actually pass the arguments
			data := url.Values{}
			data.Add("api.token", p.ApiToken)
			for key, val := range arguments {
				/*
					_, ok := einfo.Params[key]
					if !ok {
						log.Fatal("%s: Not a valid param for %s", key, endpoint)
					}
				*/
				data.Add(key, val)
			}
			data_chan := make(chan json.RawMessage, MAX_BUFFERED_RESPONSES)
			path, _ := url.Parse(endpoint)
			ep := p.ApiEndpoint.ResolveReference(path)
			go func() {
				for {
					log.Print(data.Encode())
					resp, err := http.PostForm(ep.String(), data)
					log.Printf("[%s] %s %s", resp.Status, resp.Request.Method, ep.String())
					if err != nil {
						log.Fatal(err)
					}
					defer resp.Body.Close()
					body, err := ioutil.ReadAll(resp.Body)
					if err != nil {
						log.Fatal(err)
					}
					var base_resp BaseResponse
					err = json.Unmarshal(body, &base_resp)
					if err != nil {
						log.Fatal(err)
					}
					if base_resp.ErrorCode != "" {
						log.Fatal(base_resp.ErrorCode, base_resp.ErrorInfo)
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

func (t *Ticket) String() string {
	return fmt.Sprintf("T%d: %s", t.Id, t.Fields.Name)
}

func (t *Project) String() string {
	return fmt.Sprintf("[%s|%d]: %s", t.Type, t.Id, t.Fields.Name)
}

func ProjectResponseCallback(projects chan<- interface{}, data <-chan json.RawMessage) error {
	for json_data := range data {
		var p Project
		err := json.Unmarshal(json_data, &p)
		if err != nil {
			log.Fatal(err)
		}
		projects <- p
	}
	close(projects)
	return nil // TODO dont ignore
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
	token, ok := os.LookupEnv("PHAB_TOKEN")
	if !ok {
		log.Fatal("No PHAB_TOKEN in the environment")
	}
	var phab Phabricator
	phab.Init(phab_conduit_api, token)

	ticket_args := make(EndpointArguments)
	ticket_args["queryKey"] = "authored"
	/*
		ticket_attachments := map[string]bool{
			"columns":     true,
			"subscribers": true,
			"projects":    true,
		}
		ta, err := json.Marshal(ticket_attachments)
		ticket_args["attachments"] = string(ta)
	*/
	// ticket_args["constraints[statuses][0]"] = "open"
	ticket_args["constraints"] = `{ "statuses": ["open"] }`
	ticket_chan, err := phab.Call("maniphest.search", ticket_args, TicketResponseCallback)
	if err != nil {
	}
	for ticket := range ticket_chan {
		//fmt.Sprintf("T%d\n", ticket.(Ticket).Id)
		t, _ := ticket.(Ticket)
		fmt.Println(&t)
	}

	// ----------------------------------------
	project_args := make(EndpointArguments)
	/*
		project_attachments := map[string]bool{
			"watchers":  true,
			"members":   true,
			"ancestors": true,
		}
		pa, err := json.Marshal(project_attachments)
		project_args["attachments"] = string(pa)
	*/
	project_chan, err := phab.Call("project.search", project_args, ProjectResponseCallback)
	for project := range project_chan {
		//fmt.Sprintf("T%d\n", ticket.(Ticket).Id)
		p, _ := project.(Project)
		fmt.Println(&p)
	}
	if err != nil {
		log.Fatal(err)
	}
}
