/*
Package rollbar is a Golang Rollbar client that makes it easy to report errors to Rollbar with full stacktraces.

Basic Usage

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
      rollbar.Error(rollbar.ERR, err)
    }

    rollbar.Message("info", "Message body goes here")

    rollbar.Wait()
  }

The interface exposed via the functions at the root of the `rollbar` package provide a convient way to interact with Rollbar without having to instantiate and manage your own instance of the `Client` type. There are two implementations of the `Client` type, `AsyncClient` and `SyncClient`.
*/
package rollbar
