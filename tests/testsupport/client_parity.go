package testsupport

import "testing"

func RequireOrderedRevisions(t *testing.T, revisions []int64) {
	t.Helper()
	for index := 1; index < len(revisions); index++ {
		if revisions[index] <= revisions[index-1] {
			t.Fatalf("revisions not ordered: %v", revisions)
		}
	}
}
