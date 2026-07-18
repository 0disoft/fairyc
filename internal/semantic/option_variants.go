package semantic

const (
	optionVariantSome = "some"
	optionVariantNone = "none"
)

func optionVariantPath(value string) (string, bool) {
	switch value {
	case "Option::Some":
		return optionVariantSome, true
	case "Option::None":
		return optionVariantNone, true
	default:
		return "", false
	}
}

func isBuiltinVariantPath(value string) bool {
	if _, ok := resultVariantPath(value); ok {
		return true
	}
	_, ok := optionVariantPath(value)
	return ok
}
