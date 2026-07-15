// Package errors defines reusable domain error contracts.
package errors

import "fmt"

type InvalidTransitionError struct {
	Entity string
	From   string
	To     string
}

func (e InvalidTransitionError) Error() string {
	return fmt.Sprintf("invalid %s transition: %s -> %s", e.Entity, e.From, e.To)
}
