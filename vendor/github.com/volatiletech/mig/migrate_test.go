package mig

import (
	"testing"
)

func newMigration(v int64, src string) *migration {
	return &migration{version: v, previous: -1, next: -1, source: src}
}

func TestMigrationSort(t *testing.T) {

	ms := migrations{}

	// insert in any order
	ms = append(ms, newMigration(20120000, "test"))
	ms = append(ms, newMigration(20128000, "test"))
	ms = append(ms, newMigration(20129000, "test"))
	ms = append(ms, newMigration(20127000, "test"))

	ms = sortAndConnectMigrations(ms)

	sorted := []int64{20120000, 20127000, 20128000, 20129000}

	validateMigrationSort(t, ms, sorted)
}

func validateMigrationSort(t *testing.T, ms migrations, sorted []int64) {

	for i, m := range ms {
		if sorted[i] != m.version {
			t.Error("incorrect sorted version")
		}

		var next, prev int64

		if i == 0 {
			prev = -1
			next = ms[i+1].version
		} else if i == len(ms)-1 {
			prev = ms[i-1].version
			next = -1
		} else {
			prev = ms[i-1].version
			next = ms[i+1].version
		}

		if m.next != next {
			t.Errorf("mismatched next. v: %v, got %v, wanted %v\n", m, m.next, next)
		}

		if m.previous != prev {
			t.Errorf("mismatched previous v: %v, got %v, wanted %v\n", m, m.previous, prev)
		}
	}

	t.Log(ms)
}
