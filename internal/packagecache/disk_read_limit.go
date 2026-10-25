package packagecache

// diskReadLimit reserves one byte for oversize detection when representable.
// Larger caller bounds saturate rather than wrapping io.LimitReader's signed
// byte count to zero or a negative value.
func diskReadLimit(maxBytes uint64) int64 {
	const maxRead = int64(^uint64(0) >> 1)
	if maxBytes >= uint64(maxRead) {
		return maxRead
	}
	return int64(maxBytes) + 1
}
