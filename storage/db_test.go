package storage

import "testing"

func TestSetDb(t *testing.T) {
	defer RevokeDb()
	if err := SetDb(); err != nil {
		t.Fatalf("set up db error %v\n", err)
	}
}
