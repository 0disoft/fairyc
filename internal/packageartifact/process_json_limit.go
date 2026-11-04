package packageartifact

import (
	"reflect"
	"strconv"
)

// Only the two closed PackageID-to-digest process maps reach this counter.
// Map order does not affect JSON length; use MapRange without sorted key copies.
func processJSONMapCount(v reflect.Value, remaining *uint64) bool {
	if v.IsNil() {
		return stableJSONTake(remaining, 4)
	}
	if !stableJSONTake(remaining, 2) || v.Len() > 0 && !stableJSONTake(remaining, uint64(v.Len()-1)) {
		return false
	}
	iterator := v.MapRange()
	for iterator.Next() {
		var digits [32]byte
		key := strconv.AppendUint(digits[:0], iterator.Key().Uint(), 10)
		if !stableJSONTake(remaining, uint64(len(key)+3)) || !stableJSONCount(iterator.Value(), remaining) {
			return false
		}
	}
	return true
}
