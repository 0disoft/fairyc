package mir

// DefaultVerificationReportLimit bounds admitted failure facts before canonical
// sorting/deduplication. One terminal marker is retained in addition to this
// limit, so incomplete verification can never be mistaken for success.
const DefaultVerificationReportLimit = 256
const CodeVerificationReportLimit = "V1004"

func verificationReportsFull(failures []VerificationError) bool {
	return len(failures) != 0 && failures[len(failures)-1].Code == CodeVerificationReportLimit
}

// Every helper and parent merge uses the same bounded append. A child's terminal
// marker propagates, and parents do not build an unbounded intermediate slice.
// All state is caller-owned; independent verifications share no mutable budget.
func appendVerificationErrors(failures []VerificationError, next ...VerificationError) []VerificationError {
	if verificationReportsFull(failures) {
		return failures
	}
	for _, failure := range next {
		if len(failures) == DefaultVerificationReportLimit || failure.Code == CodeVerificationReportLimit {
			return append(failures, VerificationError{
				Code: CodeVerificationReportLimit, Entity: "program",
				Detail: "MIR verification report limit exceeded; later failures omitted",
			})
		}
		failures = append(failures, failure)
	}
	return failures
}
