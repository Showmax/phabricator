package main

import (
	"encoding/json"
	"fmt"
	"log"
	"time"

	"go.showmax.cc/phabricator"
	phab_types "go.showmax.cc/phabricator/types"
)

/*
This callback gets fed the results of the API calls as unparsed JSON messages.
It's up to the callback to parse the objects.
*/
func TicketResponseCallback(tickets chan<- interface{}, data <-chan json.RawMessage) error {
	defer close(tickets)
	for json_data := range data {
		var t phab_types.Ticket
		err := json.Unmarshal(json_data, &t)
		if err != nil {
			return err
		}
		tickets <- t
	}
	return nil
}

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
	ticket_args := phab_types.TicketSearchArgs{}

	// Include the PHIDs of people watching the tickets in the results
	ticket_args.Attachments.Subscribers = true
	// Only search the tickets the current user has authored
	ticket_args.QueryKey = "authored"
	// Only consider tickets created in the past week in the search
	now := time.Now()
	seven_days_ago := now.AddDate(0, 0, -7)
	ticket_args.Constraints.CreatedStart = seven_days_ago.Unix()

	tickets, err := phab.Call("maniphest.search", ticket_args, TicketResponseCallback, false)
	if err != nil {
		log.Fatal(err)
	}
	for t := range tickets {
		ticket := t.(phab_types.Ticket)
		fmt.Printf("T%d: %s\n", ticket.Id, ticket.Fields.Name)
	}
}
