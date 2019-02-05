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

You can also call any .edit API endpoint, providing you know the transactions
it can handle.

## Architecture
The library is inspired by
[disqus/python-phabricator](https://github.com/disqus/python-phabricator).
Just like the Python library, it uses the `conduit.query` endpoint to
dynamically find out about all endpoints.

All Phabricator API interactions are handled by the type `Phabricator`. See the
code example under examples.

## Shortcomings
* Only \*.search and \*.edit endpoints are supported for now.
* Support for edit endpoints is currently very bare-bones (but completely usable)
* Probably many more...
