package ast

import "github.com/0disoft/fairyc/internal/diagnostic"

// CallArgumentLabels is sparse source metadata; values remain ordinary children.
type CallArgumentLabels struct {
	Labels                []ArgumentLabel
	ExplicitTypeArguments bool
}

type ArgumentLabel struct {
	Index int
	Name  string
	Span  diagnostic.Span
}
