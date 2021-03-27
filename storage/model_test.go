package storage

import (
	"fmt"
	"testing"

	"github.com/fatih/structs"
)

func TestGetFields(t *testing.T) {
	var query TaskQuery

	s := structs.New(&query)
	for _, key := range s.Names() {
		tmp := s.Field(key)
		fields := tmp.Fields()
		for _, f := range fields {
			t.Log(f.Tag("json"))
			kind := fmt.Sprintf("%v", f.Kind())
			t.Log(kind)
			t.Log(f.Value())
		}
	}

}
