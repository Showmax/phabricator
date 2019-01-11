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

## Massive Concurrency
MC can be turned on by the last argument to `Call()`.
As opposed to standard mode, it doesn't wait for you to start
consuming the data at all, instead it fires off consecutive
requests to an endpoint at the earliest possibility - once the
field `after` is known from the original Phab response.

For example, if you requested an endpoint with just a few shy of 700 results,
Phabricator will paginate them and force you to make 7 requests to get all of them.
The regular mode would get blocked by you not consuming the results from the outpu
channel and naturally hold off from further calls to Phabricator.
MC mode don't care ;) MC mode just gets more data.

## Shortcomings
* Only \*.search endpoints are supported for now.
* Probably many more...
