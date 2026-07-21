package semantic

const (
	resultVariantOK  = "ok"
	resultVariantErr = "err"
)

func resultVariantPath(value string) (string, bool) {
	switch value {
	case ".ok":
		return resultVariantOK, true
	case ".err":
		return resultVariantErr, true
	default:
		return "", false
	}
}

func isResultVariantPath(value string) bool {
	_, ok := resultVariantPath(value)
	return ok
}
