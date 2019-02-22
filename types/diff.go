package phabricator

type DiffSearchArgs struct {
	QueryKey    string `url:"queryKey,omitempty"`
	Attachments struct {
		Commits bool `url:"commits,omitempty"`
	} `url:"attachments"`
	Constraints struct {
		IDs           []int    `url:"ids,omitempty,brackets"`
		PHIDs         []string `url:"phids,omitempty,brackets"`
		RevisionPHIDs []string `url:"revisionPHIDs,omitempty,brackets"`
	} `url:"constraints,omitempty"`
	Order string `url:"order,omitempty"`
}

type DiffRef struct {
	Type string `json:"type"`
	Name string `json:"name"`
}

type DiffCommit struct {
	Identifier string   `json:"identifier"`
	Tree       string   `json:"tree"`
	Parents    []string `json:"parents"`
	Author     struct {
		Name  string `json:"name"`
		Email string `json:"email"`
		Raw   string `json:"raw"`
		Epoch int64  `json:"epoch"`
	} `json:"author"`
	Message string `json:"message"`
}

type Diff struct {
	Id     int    `json:"id"`
	Type   string `json:"type"`
	Phid   string `json:"phid"`
	Fields struct {
		RevisionPHID   string    `json:"revisionPHID"`
		AuthorPHID     string    `json:"authorPHID"`
		RepositoryPHID string    `json:"repositoryPHID"`
		Refs           []DiffRef `json:"refs"`
		DateCreated    int64     `json:"dateCreated"`
		DateModified   int64     `json:"dateModified"`
		Policy         struct {
			View string `json:"view"`
			Edit string `json:"edit"`
		} `json:"policy"`
	} `json:"fields"`
	Attachments struct {
		Commits struct {
			Commits []DiffCommit `json:"commits"`
		} `json:"commits"`
	} `json:"attachments"`
}
