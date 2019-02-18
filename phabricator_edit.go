package phabricator

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"strings"

	structs "github.com/fatih/structs"

	// https://github.com/Sirupsen/logrus
	log "github.com/sirupsen/logrus"
)

type editEndpointCallback func(ctx context.Context, endpoint string, einfo endpointInfo, arguments *EditArguments) error

type TxnResp struct {
	PHID string
}
type baseEditResponse struct {
	Result struct {
		Object struct {
			ID   int    `json:"id"`
			PHID string `json:"phid"`
		} `json:"object"`
		Transactions []TxnResp `json:"transactions"`
	} `json:"result"`
	ErrorCode string `json:"error_code"`
	ErrorInfo string `json:"error_info"`
}

type EditArguments struct {
	ObjectIdentifier interface{}       `url:"objectIdentifier,omitempty"`
	Transactions     []PhabTransaction `url:"transactions,numbered,brackets"`
}

func (p *Phabricator) CallEdit(ctx context.Context, endpoint string, arguments *EditArguments) error {
	handler, defined := p.editEndpoints[endpoint]
	if !defined {
		errMsg := "No callback defined for endpoint"

		logger.WithFields(log.Fields{
			"endpoint": endpoint,
		}).Error(errMsg)
		return nil
	}
	return handler(ctx, endpoint, p.apiInfo[endpoint], arguments)
}

func editArgsToPost(arguments *EditArguments) (string, error) {
	var builder strings.Builder
	switch arguments.ObjectIdentifier.(type) {
	case int:
		builder.WriteString(fmt.Sprintf("objectIdentifier=%d", arguments.ObjectIdentifier))
	case string:
		builder.WriteString(fmt.Sprintf("objectIdentifier=%s", arguments.ObjectIdentifier))
	case nil:
		// No objectIdentifier
	default:
		return "", errors.New("objectIdentifier has unsupported type")
	}

	for index, tx := range arguments.Transactions {
		builder.WriteString(fmt.Sprintf("&transactions[%d][type]=%s", index, tx.Type))
		builder.WriteString(fmt.Sprintf("&transactions[%d][value]=%v", index, tx.Value))
	}
	return builder.String(), nil
}

func (p *Phabricator) editEndpointHandler(ctx context.Context, endpoint string, einfo endpointInfo, arguments *EditArguments) error {
	queryArgs, err := editArgsToPost(arguments)
	if err != nil {
		return err
	}
	data := fmt.Sprintf("api.token=%s&%s", p.apiToken, queryArgs)
	path, _ := url.Parse(endpoint)
	fullEndpoint := p.apiEndpoint.ResolveReference(path).String()

	logger.WithFields(log.Fields{
		"endpoint":   fullEndpoint,
		"query_args": queryArgs,
	}).Debug("Sending request")
	body, err := p.postRequest(ctx, fullEndpoint, data)
	if err != nil {
		logger.WithFields(log.Fields{
			"error":     err,
			"post_data": queryArgs,
			"endpoint":  endpoint,
		}).Error("Request to Phabricator failed")
		return err
	}
	var baseResp baseEditResponse
	err = json.Unmarshal(body, &baseResp)
	if err != nil {
		logger.WithError(err).Error("Failed to decode JSON")
		return err
	}
	if baseResp.ErrorCode != "" {
		logger.WithFields(log.Fields{
			"PhabricatorErrorCode": baseResp.ErrorCode,
			"PhabricatorErrorInfo": baseResp.ErrorInfo,
		}).Error("Invalid Phabricator Request")
		return fmt.Errorf("[%s] %s", baseResp.ErrorCode, baseResp.ErrorInfo)
	}
	logger.WithFields(structs.Map(baseResp)).Debug("Response")
	return nil
}
