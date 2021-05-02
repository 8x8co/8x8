package mw

import "github.com/justinas/alice"

func New() alice.Chain {
	a := alice.New()
	return a.Append(
		Log,
	)
}
