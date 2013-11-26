go-rollbar
----------

`go-rollbar` is a Rollbar client for reporting errors to Rollbar. Errors are
reported asynchronously in a goroutine.

Keep in mind that Go's `error` type doesn't contain stack trace
information. `go-rollbar` reports the stack trace of the location that the
error was reported, not created.

Documentation
=============

[API docs on godoc.org](http://godoc.org/github.com/stvp/rollbar)

Usage
=====

    package main

    import (
      "github.com/stvp/rollbar"
    )

    func main() {
      rollbar.Token = "MY_TOKEN"
      rollbar.Environment = "production" // defaults to "development"

      result, err := DoSomething()
      if err != nil {
        // level should be one of: "critical", "error", "warning", "info", "debug"
        rollbar.Error("error", err)
      }

      rollbar.Message("info", "Message body goes here")

      rollbar.Wait()
    }

Changelog
=========

* **0.0.4** - Don't send payloads to Rollbar when an empty API token is
  supplied.
* **0.0.3** - Remove incorrect "root" value that was being sent. That setting
  is meant for the path to the root code directory, but we're running in a
  compiled environment.
* **0.0.2** - Add `Wait()` command to wait for all errors to be sent to
  Rollbar, simplify reported error class for `errors.errorString` errors.
* **0.0.1** - Initial release.

Running Tests
=============

Set up a dummy project in Rollbar and pass the access token as an environment
variable to `go test`:

    TOKEN=f0df01587b8f76b2c217af34c479f9ea go test

And verify the reported errors manually.

