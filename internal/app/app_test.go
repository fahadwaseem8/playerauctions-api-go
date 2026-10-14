package app

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/fahadwaseem8/playerauctions-api-go/internal/offers"
	"github.com/fahadwaseem8/playerauctions-api-go/internal/playerauctions"
)

func TestPreviewRunsWithoutPlayerAuctionsCredentials(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" {
			t.Error("preview tried a write")
		}
		if r.URL.Query().Get("offset") != "0" {
			fmt.Fprint(w, `[]`)
			return
		}
		switch r.URL.Path {
		case "/rest/v1/playerauctions_services":
			fmt.Fprint(w, `[{"id":"11111111-1111-4111-8111-111111111111","base_title":"Raid","unit_name":"Raids","base_quantity":10,"step_increment":10,"max_steps":2,"base_price":5,"discount_modifier":1,"base_delivery_days":1,"time_scales":false,"playerauctions_service_categories":[{"playerauctions_categories":{"name":"Raids"}}]}]`)
		case "/rest/v1/playerauctions_descriptions":
			fmt.Fprint(w, `[{"content":"<p>Test & description</p>"}]`)
		default:
			t.Errorf("unexpected endpoint %s", r.URL.Path)
		}
	}))
	defer server.Close()
	t.Setenv("SUPABASE_URL", server.URL)
	t.Setenv("SUPABASE_KEY", "test-key")
	t.Setenv("PLAYERAUCTIONS_API_KEY", "")
	t.Setenv("PLAYERAUCTIONS_API_SECRET", "")
	path := filepath.Join(t.TempDir(), "offers.json")
	if err := os.WriteFile(path, []byte(`{"gameId":123,"offerDuration":30,"totalUnit":1,"serviceCategories":{"11111111-1111-4111-8111-111111111111":{"parentItemId":516}}}`), 0600); err != nil {
		t.Fatal(err)
	}
	var stdout, stderr bytes.Buffer
	err := Run(context.Background(), []string{"-config", path, "-service-id", "11111111-1111-4111-8111-111111111111", "-limit", "1"}, &stdout, &stderr)
	if err != nil {
		t.Fatal(err)
	}
	var payloads []playerauctions.CreateBoostingRequest
	if err := json.Unmarshal(stdout.Bytes(), &payloads); err != nil {
		t.Fatal(err)
	}
	if len(payloads) != 1 || payloads[0].GameID != 123 || payloads[0].TotalUnit != 1 || payloads[0].AgreeCheck || payloads[0].OfferDesc != "<p>Test & description</p>" {
		t.Fatal(payloads)
	}
	if !strings.Contains(stderr.String(), "nothing posted") {
		t.Fatal(stderr.String())
	}
}

type fakeSeller struct {
	level     int
	notSeller bool
	sanctions bool
	failAt    int
	created   int
}

func (s *fakeSeller) PrevalidateCreation(context.Context) (playerauctions.Prevalidation, error) {
	return playerauctions.Prevalidation{IsSeller: !s.notSeller, SellerLevel: s.level, IsWarningTipSanctions: s.sanctions}, nil
}
func (s *fakeSeller) Games(context.Context) ([]playerauctions.Game, error) {
	return []playerauctions.Game{{ID: 123, ProductType: "powerleveling"}}, nil
}
func (s *fakeSeller) BoostingCategories(context.Context, int) ([]playerauctions.Category, error) {
	return []playerauctions.Category{{ID: 516}}, nil
}
func (s *fakeSeller) CreateBoostingOffer(_ context.Context, r playerauctions.CreateBoostingRequest) (playerauctions.CreatedOffer, error) {
	s.created++
	if s.created == s.failAt {
		return playerauctions.CreatedOffer{}, errors.New("throttled")
	}
	return playerauctions.CreatedOffer{OfferID: int64(100 + s.created), Title: r.Title}, nil
}

func batch() []offers.PlannedOffer {
	request := playerauctions.CreateBoostingRequest{GameID: 123, Title: "Test", OfferDuration: 30, AgreeCheck: true, Price: 5, TotalUnit: 1, ParentItemID: 516}
	return []offers.PlannedOffer{{ServiceID: "11111111-1111-4111-8111-111111111111", Quantity: 10, Payload: request}, {ServiceID: "11111111-1111-4111-8111-111111111111", Quantity: 20, Payload: request}, {ServiceID: "11111111-1111-4111-8111-111111111111", Quantity: 30, Payload: request}}
}

func TestPostingPacingAndStopOnFailure(t *testing.T) {
	for _, level := range []int{0, 3, 4, 5} {
		seller := &fakeSeller{level: level, failAt: 2}
		var stdout, stderr bytes.Buffer
		var waits []time.Duration
		err := publish(context.Background(), seller, batch(), &stdout, &stderr, func(_ context.Context, d time.Duration) error { waits = append(waits, d); return nil })
		if err == nil || !strings.Contains(err.Error(), "1 successful creations") || seller.created != 2 {
			t.Fatalf("created=%d err=%v", seller.created, err)
		}
		want := 19 * time.Second
		if level >= 4 {
			want = 10 * time.Second
		}
		if len(waits) != 1 || waits[0] != want {
			t.Fatal(waits)
		}
		var result struct {
			OfferID   int64  `json:"offerId"`
			ServiceID string `json:"serviceId"`
		}
		if err := json.Unmarshal(stdout.Bytes(), &result); err != nil || result.OfferID != 101 {
			t.Fatalf("result=%s err=%v", stdout.String(), err)
		}
		if result.ServiceID != "11111111-1111-4111-8111-111111111111" {
			t.Fatal(result)
		}
	}
}

func TestEntireBatchCheckedBeforePosting(t *testing.T) {
	for _, reason := range []string{"agreement", "category", "game", "seller", "sanctions"} {
		t.Run(reason, func(t *testing.T) {
			seller := &fakeSeller{}
			plan := batch()
			switch reason {
			case "agreement":
				plan[2].Payload.AgreeCheck = false
			case "category":
				plan[2].Payload.ParentItemID = 999
			case "game":
				plan[2].Payload.GameID = 999
			case "seller":
				seller.notSeller = true
			case "sanctions":
				seller.sanctions = true
			}
			var output bytes.Buffer
			err := publish(context.Background(), seller, plan, &output, &output, func(context.Context, time.Duration) error { t.Error("unexpected wait"); return nil })
			if err == nil || seller.created != 0 {
				t.Fatalf("created=%d err=%v", seller.created, err)
			}
		})
	}
}

func TestCancellationStopsRemainingPosts(t *testing.T) {
	var output bytes.Buffer
	seller := &fakeSeller{}
	err := publish(context.Background(), seller, batch(), &output, &output, func(context.Context, time.Duration) error { return context.Canceled })
	if !errors.Is(err, context.Canceled) || seller.created != 1 {
		t.Fatalf("created=%d err=%v", seller.created, err)
	}
}

func TestCLIRejectsUnsafeOrInvalidArguments(t *testing.T) {
	for _, args := range [][]string{{"-mode", "post"}, {"-mode", "unknown"}, {"-mode", "categories"}, {"-limit", "-1"}, {"-service-id", "1"}, {"-service-id", "invalid"}, {"unexpected"}} {
		var output bytes.Buffer
		if err := Run(context.Background(), args, &output, &output); err == nil {
			t.Fatalf("accepted %v", args)
		}
	}
}
