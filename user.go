package phabricator

type UserSearchArgs struct {
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
