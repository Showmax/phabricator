package phabricator

type RevisionSearchArgs struct {
	QueryKey    string `url:"queryKey,omitempty"`
	Attachments struct {
		Reviewers   bool `url:"reviewers,omitempty"`
		Subscribers bool `url:"subscribers,omitempty"`
		Projects    bool `url:"projects,omitempty"`
	} `url:"attachments"`
	Constraints struct {
		Ids              []int    `url:"ids,omitempty,brackets"`
		Phids            []string `url:"phids,omitempty,brackets"`
		ResponsiblePHIDs []string `url:"responsiblePHIDs,omitempty,brackets"`
		AuthorPHIDs      []string `url:"authorPHIDs,omitempty,brackets"`
		ReviewerPHIDs    []string `url:"reviewerPHIDs,omitempty,brackets"`
		RepositoryPHIDs  []string `url:"repositoryPHIDs,omitempty,brackets"`
		Statuses         []string `url:"statuses,omitempty,brackets"`
		CreatedStart     int64    `url:"createdStart,omitempty"`
		CreatedEnd       int64    `url:"createdEnd,omitempty"`
		Query            string   `url:"query,omitempty"`
		Subscribers      []string `url:"subscribers,omitempty,brackets"`
		Projects         []string `url:"projects,omitempty,brackets"`
	} `url:"constraints,omitempty"`
	Order string `url:"order,omitempty"`
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
			Closed    bool   `json:"closed"`
			ColorAnsi string `json:"color.ansi"`
		} `json:"status"`
		RepositoryPHID string `json:"repositoryPHID"`
		DiffPHID       string `json:"diffPHID"`
		Summary        string `json:"summary"`
		TestPlan       string `json:"testPlan"`
		IsDraft        bool   `json:"isDraft"`
		HoldAsDraft    bool   `json:"holdAsDraft"`
		DateCreated    int64  `json:"dateCreated"`
		DateModified   int64  `json:"dateModified"`
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
