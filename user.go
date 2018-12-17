package phabricator

type UserSearchArgs struct {
	QueryKey    string `url:"queryKey"`
	Attachments struct {
		Availability bool `url:"availability,omitempty"`
	} `url:"attachments"`
	Constraints struct {
		Ids           []int    `url:"ids,omitempty,brackets"`
		Phids         []string `url:"phids,omitempty,brackets"`
		Usernames     []string `url:"usernames,omitempty,brackets"`
		NameLike      string   `url:"nameLike,omitempty"`
		IsAdmin       bool     `url:"isAdmin,omitempty`
		IsDisabled    bool     `url:"isDisabled,omitempty`
		IsBot         bool     `url:"isBot,omitempty`
		IsMailingList bool     `url:"isMailingList,omitempty`
		NeedsApproval bool     `url:"needsApproval,omitempty`
		CreatedStart  int64    `url:"createdStart,omitempty"`
		ModifiedStart int64    `url:"modifiedStart,omitempty"`
		Query         string   `url:"query,omitempty"`
	} `url:"constraints"`
}
type User struct {
	Id     int    `json:"id"`
	Type   string `json:"type"`
	Phid   string `json:"phid"`
	Fields struct {
		Username     string   `json:"username"`
		RealName     string   `json:"realName"`
		Roles        []string `json:"roles"`
		DateCreated  int64    `json:"dateCreated"`
		DateModified int64    `json:"dateModified"`
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
