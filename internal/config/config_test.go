package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestRejectInvalidOfferSettings(t *testing.T) {
	for _, body := range []string{
		`{"gameId":0,"totalUnit":1,"offerDuration":30}`,
		`{"gameId":123,"totalUnit":0,"offerDuration":30}`,
		`{"gameId":123,"totalUnit":1,"offerDuration":10}`,
		`{"gameId":123,"totalUnit":1,"offerDuration":30,"typo":true}`,
		`{"gameId":123,"totalUnit":1,"offerDuration":30} {}`,
	} {
		path := filepath.Join(t.TempDir(), "config.json")
		if err := os.WriteFile(path, []byte(body), 0600); err != nil {
			t.Fatal(err)
		}
		if _, err := LoadOffers(path); err == nil {
			t.Fatalf("accepted %s", body)
		}
	}
}
