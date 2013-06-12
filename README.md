go-rollbar
----------

`go-rollbar` is a Rollbar client for reporting errors to Rollbar. Errors are
reported asynchronously in a background goroutine.

Errors in Go don't have any information about where they were created, so stack
traces show where the error was reported, not created.

[godoc.org documentation](http://godoc.org/github.com/stvp/rollbar)

Usage
=====

    package main

    import (
      "github.com/stvp/rollbar"
    )

    func main() {
      rollbar.Token = "MY_TOKEN"
      rollbar.Environment = "production" // defaults to "development"

      ...

      result, err := DoSomething()
      if err != nil {
        // level should be one of: "critical", "error", "warning", "info", "debug"
        rollbar.Error("error", err)
      }

      ...

      rollbar.Message("info", "Message body goes here")
    }

Running Tests
=============

Set up a dummy project in Rollbar and pass the access token as an environment
variable to `go test`:

    TOKEN=f0df01587b8f76b2c217af34c479f9ea go test

And verify the reported errors manually.

