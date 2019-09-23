package rollbar

type ExampleError struct {
	error
	wrapped error
}

func (e ExampleError) GetWrappedError() error {
	return e.wrapped
}

func ExampleSetUnwrapper() {
	SetUnwrapper(func(err error) error {
		// preserve the default behavior for other types of errors
		if unwrapped := DefaultUnwrapper(err); unwrapped != nil {
			return unwrapped
		}

		if ex, ok := err.(ExampleError); ok {
			return ex.GetWrappedError()
		}

		return nil
	})
}
