package phabricator

import "encoding/json"

type RepositorySearchArgs struct {
	QueryKey    string `url:"queryKey"`
	Attachments struct {
		Uris     bool `url:"uris,omitempty"`
		Projects bool `url:"projects,omitempty"`
	} `url:"attachments"`
	Constraints struct {
		Ids        []int    `url:"ids,omitempty,brackets"`
		Phids      []string `url:"phids,omitempty,brackets"`
		Callsigns  []string `url:"callsigns,omitempty,brackets"`
		ShortNames []string `url:"shortNames,omitempty,brackets"`
		Types      []string `url:"types,omitempty,brackets"`
		Uris       []string `url:"uris,omitempty,brackets"`
		Query      string   `url:"query,omitempty"`
		Projects   []string `url:"projects,omitempty,brackets"`
		Spaces     []string `url:"spaces,omitempty,brackets"`
	} `url:"constraints,omitempty"`
}
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
		DateCreated  int64 `json:"dateCreated"`
		DateModified int64 `json:"dateModified"`
	} `json:"fields"`
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
		DateCreated        int64  `json:"dateCreated"`
		DateModified       int64  `json:"dateModified"`
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
