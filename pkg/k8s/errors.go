package k8s

import (
	"context"

	"k8s.io/apimachinery/pkg/util/runtime"
)

// When this package is used it will inject a context error handler into the errors handlers
func init() {
	runtime.ErrorHandlers = append(runtime.ErrorHandlers, ContextChannelErrorHandler) //nolint:reassign // deliberate: route client-go errors to the watching store
}

type RuntimeError struct {
	Error         error
	Message       string
	KeysAndValues []any
}

type errorReporter struct {
	ErrorChannel chan RuntimeError
}

type errorReporterKeyType struct{}

var errorReporterKey = errorReporterKeyType{}

func ContextChannelErrorHandler(ctx context.Context, err error, msg string, keysAndValues ...interface{}) {

	if er, ok := ctx.Value(errorReporterKey).(*errorReporter); ok {
		er.ErrorChannel <- RuntimeError{
			Error:         err,
			Message:       msg,
			KeysAndValues: keysAndValues,
		}
	}
}

func WithErrorReporter(ctx context.Context, errorChan chan RuntimeError) context.Context {
	return context.WithValue(ctx, errorReporterKey, &errorReporter{errorChan})
}
