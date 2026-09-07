package ast

import "github.com/0disoft/fairyc/internal/syntax"

func lowerTypeParameters(tree *syntax.Tree, node syntax.Node) []string {
	var result []string
	for _, childID := range node.Children {
		child, found := tree.Node(childID)
		if !found || child.Kind != syntax.KindTypeParameters {
			continue
		}
		for _, parameterID := range child.Children {
			if parameter, found := tree.Node(parameterID); found {
				result = append(result, parameter.Text)
			}
		}
	}
	return result
}
