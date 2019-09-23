package rollbar_test

import "github.com/rollbar/rollbar-go"

type CustomWrappingError struct {
	error
	wrapped error
}

func (e CustomWrappingError) GetWrappedError() error {
	return e.wrapped
}

func ExampleSetUnwrapper() {
	rollbar.SetUnwrapper(func(err error) error {
		// preserve the default behavior for other types of errors
		if unwrapped := rollbar.DefaultUnwrapper(err); unwrapped != nil {
			return unwrapped
		}

		if ex, ok := err.(CustomWrappingError); ok {
			return ex.GetWrappedError()
		}

		return nil
	})
}
