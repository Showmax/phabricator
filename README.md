# Go-Phabricator

## Abstract
A Golang wrapper around some parts of Phabricator API

## Description
This library wraps around search APIs for the most common objects:
* Tickets
* Users
* Projects
* Revisions
* Repositories

A `struct` of the same name is defined for each of of the search results.
See examples for details.

## Modifying and Contributing
The library doesn't constrain you when it comes to unsupported endpoints.  If
you need to consume an endpoint that's not supported yet, you can write the
response/request types yourself, analogous to the type definitions in this
library. Please provide them for others via a Merge Request :) .

## Architecture
The library is inspired by
[disqus/python-phabricator](https://github.com/disqus/python-phabricator).
Just like the Python library, it uses the `conduit.query` endpoint to
dynamically find out about all endpoints.

All Phabricator API interactions are handled by the type `Phabricator`.  The
user provides callbacks to handle the expected results. See the code example
below.

## Code Example

```go
package main

import (
	"encoding/json"
	"fmt"
	"log"
	"time"

	"go.showmax.cc/phabricator"
)

/*
This callback gets fed the results of the API calls as unparsed JSON messages.
It's up to the callback to parse the objects.
*/
func TicketResponseCallback(tickets chan<- interface{}, data <-chan json.RawMessage) error {
	defer close(tickets)
	for json_data := range data {
		var t phabricator.Ticket
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
	phab.Init("https://phabricator.showmax.cc/api/", "<API_TOKEN>")

	/* Create constraints for the search:
	 */
	ticket_args := phabricator.TicketSearchArgs{}

	// Include the PHIDs of people watching the tickets in the results
	ticket_args.Attachments.Subscribers = true
	// Only search the tickets the current user has authored
	ticket_args.QueryKey = "authored"
	// Only consider tickets created in the past week in the search
	now := time.Now()
	seven_days_ago := now.AddDate(0, 0, -7)
	ticket_args.Constraints.CreatedStart = seven_days_ago.Unix()

	tickets, err := phab.Call("maniphest.search", ticket_args, TicketResponseCallback)
	if err != nil {
		log.Fatal(err)
	}
	for t := range tickets {
		ticket := t.(phabricator.Ticket)
		fmt.Printf("T%d: %s\n", ticket.Id, ticket.Fields.Name)
	}
}
```

## Shortcomings
* Only \*.search endpoints are supported for now.
* Only Project and Ticket searches can be properly constrained for now.
* Probably many more...
