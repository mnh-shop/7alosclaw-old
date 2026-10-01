package store

import (
	"encoding/json"
	"testing"
)

func TestMergeMetadataJSONPreservesUnpatchedSecrets(t *testing.T) {
	merged, err := MergeMetadataJSON([]byte(`{"privateKey":"secret","host":"old"}`), []byte(`{"host":"new"}`))
	if err != nil {
		t.Fatal(err)
	}
	var got map[string]string
	if err := json.Unmarshal(merged, &got); err != nil {
		t.Fatal(err)
	}
	if got["privateKey"] != "secret" || got["host"] != "new" {
		t.Fatalf("merged metadata = %s", merged)
	}
}
