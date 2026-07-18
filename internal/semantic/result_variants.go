package semantic

const (
	resultVariantOK  = "ok"
	resultVariantErr = "err"
)

func resultVariantPath(value string) (string, bool) {
	switch value {
	case "Result::Ok":
		return resultVariantOK, true
	case "Result::Err":
		return resultVariantErr, true
	default:
		return "", false
	}
}

func isResultVariantPath(value string) bool {
	_, ok := resultVariantPath(value)
	return ok
}
