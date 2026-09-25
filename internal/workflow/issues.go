package workflow

import (
	"errors"
	"strings"
)

type DefinitionError struct{ Cause error }

func (e *DefinitionError) Error() string { return e.Cause.Error() }
func (e *DefinitionError) Unwrap() error { return e.Cause }

func IsDefinitionError(err error) bool {
	var target *DefinitionError
	return errors.As(err, &target)
}

func Issue(err error) (path, reason string) {
	path, reason, ok := strings.Cut(err.Error(), ":")
	if !ok {
		return "definition", err.Error()
	}
	return strings.TrimSpace(path), strings.TrimSpace(reason)
}
