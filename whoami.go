package phabricator

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"

	// https://github.com/Sirupsen/logrus
	log "github.com/sirupsen/logrus"
)

type WhoAmI struct {
	PHID         string   `json:"phid"`
	Username     string   `json:"userName"`
	RealName     string   `json:"realName"`
	Image        string   `json:"image"`
	URI          string   `json:"uri"`
	Roles        []string `json:"roles"`
	PrimaryEmail string   `json:"primaryEmail"`
}
type WhoamiResponse struct {
	// Unfortunately, this User response is different from what you'd
	// get if you did a user.search for yourself...
	User      WhoAmI `json:"result"`
	ErrorCode string `json:"error_code"`
	ErrorInfo string `json:"error_info"`
}

func (p *Phabricator) WhoAmI(ctx context.Context) *WhoAmI {
	endpoint := "user.whoami"
	data := fmt.Sprintf("api.token=%s", p.apiToken)
	path, _ := url.Parse(endpoint)
	fullEndpoint := p.apiEndpoint.ResolveReference(path).String()

	body, err := p.postRequest(ctx, fullEndpoint, data)
	if err != nil {
		logger.WithFields(log.Fields{
			"error":    err,
			"endpoint": endpoint,
		}).Error("Request to Phabricator failed")
		return nil
	}
	var who WhoamiResponse
	fmt.Println(string(body))
	err = json.Unmarshal(body, &who)
	if err != nil {
		logger.WithFields(log.Fields{
			"error": err,
		}).Error("Failed to decode JSON")
		return nil
	}
	return &who.User
}
