package phabricator

import "fmt"

type TicketSearchArgs struct {
	QueryKey    string `url:"queryKey,omitempty"`
	Attachments struct {
		Columns     bool `url:"columns,omitempty"`
		Subscribers bool `url:"subscribers,omitempty"`
		Projects    bool `url:"projects,omitempty"`
	} `url:"attachments"`
	Constraints struct {
		Ids           []int    `url:"ids,omitempty,brackets"`
		Phids         []string `url:"phids,omitempty,brackets"`
		Assigned      []string `url:"assigned,omitempty,brackets"`
		AuthorPHIDs   []string `url:"authorPHIDs,omitempty,brackets"`
		Statuses      []string `url:"statuses,omitempty,brackets"`
		Priorities    []int    `url:"priorities,omitempty,brackets"`
		Aubtypes      []string `url:"subtypes,omitempty,brackets"`
		ColumnPHIDs   []string `url:"columnPHIDs,omitempty,brackets"`
		HasParents    bool     `url:"hasParents,omitempty"`
		HasSubtasks   bool     `url:"hasSubtasks,omitempty"`
		ParentIDs     []string `url:"parentIDs,omitempty,brackets"`
		SubtaskIDs    []string `url:"subtaskIDs,omitempty,brackets"`
		CreatedStart  int64    `url:"createdStart,omitempty"`
		ModifiedStart int64    `url:"modifiedStart,omitempty"`
		CreatedEnd    int64    `url:"createdEnd,omitempty"`
		ModifiedEnd   int64    `url:"modifiedEnd,omitempty"`
		ClosedStart   int64    `url:"closedStart,omitempty"`
		ClosedEnd     int64    `url:"closedEnd,omitempty"`
		CloserPHIDs   []string `url:"closerPHIDs,omitempty"`
		Query         string   `url:"query,omitempty"`
		Subscribers   []string `url:"subscribers,omitempty,brackets"`
		Projects      []string `url:"projects,omitempty,brackets"`
		Spaces        []string `url:"spaces,omitempty,brackets"`
	} `url:"constraints,omitempty"`
	Order string `url:"order,omitempty"`
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
		DateClosed   int64  `json:"dateClosed"`
		SpacePHID    string `json:"spacePHID"`
		DateCreated  int64  `json:"dateCreated"`
		DateModified int64  `json:"dateModified"`
		Policy       struct {
			View     string `json:"view"`
			Interact string `json:"interact"`
			Edit     string `json:"edit"`
		} `json:"policy"`
		ShowmaxAppVersion string `json:"custom.showmax:app-version"`
	} `json:"fields"`
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
}

func (t *Ticket) String() string {
	return fmt.Sprintf("T%d: %s", t.Id, t.Fields.Name)
}
