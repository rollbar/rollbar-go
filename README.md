rollbar [![Build Status](https://travis-ci.org/rollbar/rollbar-go.svg?branch=master)](https://travis-ci.org/rollbar/rollbar-go) [![Go Report Card](https://goreportcard.com/badge/github.com/rollbar/rollbar-go)](https://goreportcard.com/report/github.com/rollbar/rollbar-go)
-------

ALPHA STATUS
------------
This library is currently being actively worked on and should be considered pre-release. We will
update this message when the library is ready for production consumption.


`rollbar` is a Golang Rollbar client that makes it easy to report errors to
Rollbar with full stacktraces. Errors are sent to Rollbar asynchronously in a
background goroutine.

Because Go's `error` type doesn't include stack information from when it was set
or allocated, we use the stack information from where the error was reported.

Documentation
=============

[API docs on godoc.org](http://godoc.org/github.com/rollbar/rollbar-go)

Usage
=====

```go
package main

import (
  "github.com/rollbar/rollbar-go"
)

func main() {
  rollbar.SetToken("MY_TOKEN")
  rollbar.SetEnvironment("production")                 // defaults to "development"
  rollbar.SetCodeVersion("v2")                         // optional Git hash/branch/tag (required for GitHub integration)
  rollbar.SetServerHost("web.1")                       // optional override; defaults to hostname
  rollbar.SetServerRoot("github.com/heroku/myproject") // path of project (required for GitHub integration and non-project stacktrace collapsing)

  result, err := DoSomething()
  if err != nil {
    rollbar.Critical(err)
  }

  rollbar.Info("Message body goes here")

  rollbar.Wait()
}
```

Running Tests
=============

For full integation tests, set up a dummy project in Rollbar and pass the
access token as an environment variable to `go test`:

    TOKEN=POST_SERVER_ITEM_ACCESS_TOKEN go test

And verify the reported errors manually.

For coverage results, run:

    TOKEN=POST_SERVER_ITEM_ACCESS_TOKEN go test -coverprofile=cover.out
    go tool cover -html=cover.out -o cover.html

History
=======

This library originated with this project
[github.com/stvp/rollbar](https://github.com/stvp/rollbar).
This was subsequently forked by Heroku, [github.com/heroku/rollbar](https://github.com/heroku/rollbar),
and extended. Those two libraries diverged as features were added independently to both. This
official library is actually a fork of the Heroku fork with some git magic to make it appear as a
standalone repository along with all of that history. We then also went back to the original stvp
library and brought over most of the divergent changes. Since then we have moved forward to add more
functionality to this library and it is the recommended notifier for Go going forward.
