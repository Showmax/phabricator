package phabricator

import (
	"testing"
	"time"
)

func TestPhabUnknownLogLevel(t *testing.T) {
	var phab Phabricator
	err := phab.Init(&PhabOptions{
		API:      "localhost",
		LogLevel: "apocalypse",
	})
	if err == nil {
		t.Error("Initialization didn't catch invalid log level")
	}
}

func TestPhabNegativeTimeout(t *testing.T) {
	var phab Phabricator
	err := phab.Init(&PhabOptions{
		API:     "localhost",
		Timeout: -1 * time.Second,
	})
	if err == nil {
		t.Error("Initialization didn't catch negative timeout")
	}
}
