package phabricator

import (
	"encoding/json"
	"fmt"
)

type PhabTransaction struct {
	Type  string      `url:"type"`
	Value interface{} `url:"value"`
}

func (pt *PhabTransaction) MarshalJSON() ([]byte, error) {
	val, err := json.Marshal(pt.Value)
	if err != nil {
		return []byte{}, err
	}
	return []byte(fmt.Sprintf(`{ "type": %s, "value": %s }`, pt.Type, string(val))), nil
}

func NewTransaction(tx string, val interface{}) PhabTransaction {
	return PhabTransaction{Type: tx, Value: val}
}
