# Go-Phabricator

## Abstract
A Golang wrapper around the most common parts of Phabricator API

## Description
This library wraps around search APIs for the most common objects:
* Tickets
* Users
* Projects
* Diffs
* Revisions
* Repositories

A `struct` of the same name is defined for each of the search result types.
See examples for details.

You can also call any .edit API endpoint, providing you know the transactions
it can handle.

The calls to phabricator through this lib can be split into three categories:
* CallSearch - where you expect to get an array of zero or more results
* CallEdit - where you edit or create a single object
* WhoAmI - user.whoami

## Architecture
The library is inspired by
[disqus/python-phabricator](https://github.com/disqus/python-phabricator).
Just like the Python library, it uses the
[conduit.query](https://secure.phabricator.com/conduit/method/conduit.query/)
endpoint to dynamically find out about all endpoints.

All Phabricator API interactions are handled by the type `Phabricator`. See the
code example under examples.

## Limitations
* Only \*.search and \*.edit endpoints are supported for now.
* Support for edit endpoints is currently very bare-bones (but completely usable)
