/*
Package rollbar is a Golang Rollbar client that makes it easy to report errors to Rollbar with full stacktraces.

Basic Usage

This package is designed to be used via the functions exposed at the root of the `rollbar` package. These work by managing a single instance of the `Client` type that is configurable via the setter functions at the root of the package.

  package main

  import (
    "github.com/rollbar/rollbar-go"
    "time"
  )

  func main() {
    rollbar.SetToken("MY_TOKEN")
    rollbar.SetEnvironment("production") // defaults to "development"
    rollbar.SetCodeVersion("v2")         // optional Git hash/branch/tag (required for GitHub integration)
    rollbar.SetServerHost("web.1")       // optional override; defaults to hostname
    rollbar.SetServerRoot("/")           // local path of project (required for GitHub integration and non-project stacktrace collapsing)

    rollbar.Info("Message body goes here")
    rollbar.WrapAndWait(doSomething)
  }

  func doSomething() {
    var timer *time.Timer = nil
    timer.Reset(10) // this will panic
  }

If you wish for more fine grained control over the client or you wish to have multiple independent clients then you can create and manage your own instances of the `Client` type.

We provide two implementations of the `Transport` interface, `AsyncTransport` and `SyncTransport`. These manage the communication with the network layer. The Async version uses a buffered channel to communicate with the Rollbar API in a separate go routine. The Sync version is fully synchronous. It is possible to create your own `Transport` and configure a Client to use your preferred implementation.

Handling Panics

Go does not provide a mechanism for handling all panics automatically, therefore we provide two functions `Wrap` and `WrapAndWait` to make working with panics easier. They both take a function and then report to Rollbar if that function panics. They use the recover mechanism to capture the panic, and therefore if you wish your process to have the normal behaviour on panic (i.e. to crash), you will need to re-panic the result of calling `Wrap`. For example,

  package main

  import (
    "errors"
    "github.com/rollbar/rollbar-go"
  )

  func PanickyFunction() {
    panic(errors.New("AHHH!!!!"))
  }

  func main() {
    rollbar.SetToken("MY_TOKEN")
    err := rollbar.Wrap(PanickyFunction)
    if err != nil {
      // This means our function panic'd
      // Uncomment the next line to get normal
      // crash on panic behaviour
      // panic(err)
    }
    rollbar.Wait()
  }

The above pattern of calling `Wrap(...)` and then `Wait(...)` can be combined via `WrapAndWait(...)`. When `WrapAndWait(...)` returns if there was a panic it has already been sent to the Rollbar API. The error is still returned by this function if there is one.

Tracing Errors

Due to the nature of the `error` type in Go, it can be difficult to attribute errors to their original origin without doing some extra work. To account for this, we provide multiple ways of configuring the client to unwrap errors and extract stack traces.

The client will automatically unwrap any error type which implements the `Unwrap() error` method specified in Go 1.13. (See https://golang.org/pkg/errors/ for details.) This behavior can be extended for other types of errors by calling `SetUnwrapper`.

For stack traces, we provide the `Stacker` interface, which can be implemented on custom error types:

    type Stacker interface {
      Stack() []runtime.Frame
    }

If you cannot implement the `Stacker` interface on your error type (which is common for third-party error libraries), you can provide a custom tracing function by calling `SetStackTracer`.

See the documentation of `SetUnwrapper` and `SetStackTracer` for more information and examples.
*/
package rollbar
