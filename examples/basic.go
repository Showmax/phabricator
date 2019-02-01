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
	phab.Init("https://phabricator.showmax.cc/api/", "<API_TOKEN>", &phabricator.PhabOptions{
		LogLevel: "info", // Must be a level recognized by the logrus library
		Timeout:  10 * time.Second,
	})

	/* Create constraints for the search:
	 */
	ticketArgs := phabTypes.TicketSearchArgs{}

	// Include the PHIDs of people watching the tickets in the results
	ticketArgs.Attachments.Subscribers = true
	// Only search the tickets the current user has authored
	ticketArgs.QueryKey = "authored"
	// Only consider tickets created in the past week in the search
	now := time.Now()
	sevenDaysAgo := now.AddDate(0, 0, -7)
	ticketArgs.Constraints.CreatedStart = sevenDaysAgo.Unix()

	ctx, cancelCtx := context.WithCancel(context.Background())
	tickets, errors := phab.Call(ctx, "maniphest.search", ticketArgs, phabTypes.Ticket{})
	if tickets == nil || errors == nil {
		log.Fatal("Non-existent endpoint")
	}
	for e := range errors {
		log.Print(e)
		cancelCtx()
		return
	}
	for t := range tickets {
		ticket := t.(*phabTypes.Ticket)
		fmt.Printf("T%d: %s\n", ticket.Id, ticket.Fields.Name)
	}
	cancelCtx()
}
