package packagecache

import (
	"fmt"
	"testing"
	"time"
)

func TestDiskReadLimitDoesNotOverflow(t *testing.T) {
	const maxRead = int64(^uint64(0) >> 1)
	for _, test := range []struct {
		limit uint64
		want  int64
	}{
		{0, 1}, {1, 2}, {1024, 1025},
		{uint64(maxRead) - 2, maxRead - 1}, {uint64(maxRead) - 1, maxRead},
		{uint64(maxRead), maxRead}, {uint64(maxRead) + 1, maxRead}, {^uint64(0), maxRead},
	} {
		if got := diskReadLimit(test.limit); got != test.want {
			t.Errorf("diskReadLimit(%d) = %d, want %d", test.limit, got, test.want)
		}
	}
}

func TestDiskStoreReadsWithLargeBounds(t *testing.T) {
	for _, limit := range []uint64{1<<63 - 1, 1 << 63, ^uint64(0)} {
		t.Run(fmt.Sprintf("limit-%d", limit), func(t *testing.T) {
			store, err := NewDiskStore(t.TempDir(), DiskLimits{
				MaxEntries: 1, MaxBytes: limit, MaxRecordBytes: limit, TempMaxAge: time.Minute,
			})
			if err != nil {
				t.Fatal(err)
			}
			t.Cleanup(func() { store.Close() })
			record := diskOwnershipRecord()
			if err := store.Put(record); err != nil {
				t.Fatal(err)
			}
			got, found := store.Get(record.Key)
			if !found || !recordsEqual(got, record) {
				t.Error("large caller limit rejected a small valid generation")
			}
			stats, err := store.Stats()
			if err != nil || stats.Entries != 1 || stats.Rejected != 0 {
				t.Errorf("valid generation was removed or rejected: stats=%+v, error=%v", stats, err)
			}
			// Republish so quarantine is checked independently of lookup failure.
			if err := store.Put(record); err != nil {
				t.Fatal(err)
			}
			if !store.DeleteIfMatch(record) {
				t.Error("large caller limit prevented matching generation quarantine")
			}
		})
	}
}
