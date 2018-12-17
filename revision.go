package main

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
