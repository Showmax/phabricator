package phabricator

import (
	"encoding/json"
	"fmt"

	log "github.com/sirupsen/logrus"
)

type PhabTransaction struct {
	Type  string      `url:"type"`
	Value interface{} `url:"value"`
}

func (pt *PhabTransaction) MarshalJSON() ([]byte, error) {

	logger := logger.WithFields(log.Fields{
		"tx_type":    pt.Type,
		"value_type": fmt.Sprintf("%T", pt.Value),
	})

	val, err := json.Marshal(pt.Value)
	if err != nil {
		logger.Error("Failed to marshal a transaction value to JSON")
		return []byte{}, err
	}
	j := fmt.Sprintf(`{ "type": %s, "value": %s }`, pt.Type, string(val))
	logger.WithField("marshal", j).Debug("Marshaled a transaction to JSON")
	return []byte(j), nil
}

func NewTransaction(tx string, val interface{}) PhabTransaction {
	return PhabTransaction{Type: tx, Value: val}
}
