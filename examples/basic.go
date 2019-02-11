package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"go.showmax.cc/phabricator"
	phabTypes "go.showmax.cc/phabricator/types"
)

/*
This short program prints the names and ids of all the tickets created by the
current use in the past week.
*/
func main() {
	var phab phabricator.Phabricator
	err := phab.Init(&phabricator.PhabOptions{
		API:      "https://phabricator.showmax.cc/api/",
		LogLevel: "error", // Must be a level recognized by the logrus library
		Timeout:  10 * time.Second,
	})
	if err != nil {
		log.Fatal(err)
	}

	// Create constraints for the search:
	ticketArgs := phabTypes.TicketSearchArgs{QueryKey: "authored"}
	// Include the PHIDs of people watching the tickets in the results
	ticketArgs.Attachments.Subscribers = true
	// Only consider tickets created in the past 30 days in the search
	thirtyDaysAgo := time.Now().UTC().AddDate(0, 0, -30).Unix()
	ticketArgs.Constraints.CreatedStart = thirtyDaysAgo

	// The context allows you to cancel the current call prematurely
	ctx, cancelCtx := context.WithCancel(context.Background())
	defer cancelCtx()
	results := phab.CallSearch(ctx, "maniphest.search", ticketArgs, phabTypes.Ticket{})
	if results == nil {
		log.Fatal("Non-existent endpoint")
	}
	for t := range results {
		switch t.(type) {
		case error:
			log.Print(t)
			return
		case *phabTypes.Ticket:
			ticket := t.(*phabTypes.Ticket)
			fmt.Printf("T%d: %s\n", ticket.Id, ticket.Fields.Name)
		default:
			log.Panic("Unexpected type received from the Phabricator library")
		}
	}
}
