package phabricator

import "fmt"

type ProjectSearchArgs struct {
	QueryKey    string `url:"queryKey"`
	Attachments struct {
		Members   bool `url:"members,omitempty"`
		Watchers  bool `url:"watchers,omitempty"`
		Ancestors bool `url:"ancestors,omitempty"`
	} `url:"attachments"`
	Constraints struct {
		Ids         []int    `url:"ids,omitempty,brackets"`
		Phids       []string `url:"phids,omitempty,brackets"`
		Slugs       []string `url:"slugs,omitempty,brackets"`
		Members     []string `url:"members,omitempty,brackets"`
		Watchers    []string `url:"watchers,omitempty,brackets"`
		IsMilestone []bool   `url:"isMilestone,omitempty`
		Icons       []string `url:"icons,omitempty,brackets"`
		Colors      []string `url:"colors,omitempty,brackets"`
		Parents     []string `url:"parents,omitempty,brackets"`
		Ancestors   []string `url:"ancestors,omitempty,brackets"`
		Query       string   `url:"query,omitempty"`
		Spaces      []string `url:"spaces,omitempty,brackets"`
	} `url:"constraints"`
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
		DateCreated  int64  `json:"dateCreated"`
		DateModified int64  `json:"dateModified"`
		Policy       struct {
			View string `json:"view"`
			Edit string `json:"edit"`
			Join string `json:"join"`
		} `json:"policy"`
		Description string `json:"description"`
	} `json:"fields"`
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
}

func (t *Project) String() string {
	return fmt.Sprintf("[%s|%d]: %s", t.Type, t.Id, t.Fields.Name)
}
