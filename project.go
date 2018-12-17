package main

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
