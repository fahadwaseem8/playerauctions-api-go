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
		`{"gameId":123,"totalUnit":10,"offerDuration":30,"serviceCategories":{"1":{"parentItemId":516}}}`,
		`{"gameId":123,"totalUnit":10,"offerDuration":30,"serviceCategories":{"invalid-uuid":{"parentItemId":516}}}`,
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

func TestServiceUUIDMapping(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.json")
	body := `{"gameId":123,"totalUnit":10,"offerDuration":30,"serviceCategories":{"AAAAAAAA-AAAA-4AAA-8AAA-AAAAAAAAAAAA":{"parentItemId":516}}}`
	if err := os.WriteFile(path, []byte(body), 0600); err != nil {
		t.Fatal(err)
	}
	settings, err := LoadOffers(path)
	if err != nil {
		t.Fatal(err)
	}
	if settings.ServiceCategories["aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa"].ParentItemID != 516 {
		t.Fatal(settings.ServiceCategories)
	}
}
