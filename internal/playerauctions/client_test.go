package playerauctions

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestSignatureGolden(t *testing.T) {
	got := signature("test-secret", "test-key", "1780293232", []byte(`{"title":"A & B"}`))
	// Independently calculated with .NET HMACSHA256.
	const want = "77d2b8186ccfe83e090f0a7c20e897250755d57a4524ee761559fe852359c4a5"
	if got != want {
		t.Fatalf("signature = %s", got)
	}
}

func mockClient(t *testing.T, handler http.HandlerFunc) *Client {
	t.Helper()
	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)
	client, err := NewClient("test-key", "test-secret")
	if err != nil {
		t.Fatal(err)
	}
	client.baseURL = server.URL
	client.now = func() time.Time { return time.Unix(1780293232, 0) }
	return client
}

func validRequest() CreateBoostingRequest {
	return CreateBoostingRequest{GameID: 3637, Title: "A & B", OfferDesc: "<p>Test</p>", OfferDuration: 30, AgreeCheck: true, Price: 9.99, TotalUnit: 1, ParentItemID: 516}
}

func TestSignedCreateAndMetadata(t *testing.T) {
	calls := 0
	client := mockClient(t, func(w http.ResponseWriter, r *http.Request) {
		calls++
		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Error(err)
		}
		if r.Header.Get("X-PA-API-KEY") != "test-key" || r.Header.Get("X-PA-TIMESTAMP") != "1780293232" {
			t.Error("wrong auth headers")
		}
		mac := hmac.New(sha256.New, []byte("test-secret"))
		mac.Write(append([]byte("test-key1780293232"), body...))
		if r.Header.Get("X-PA-SIGN") != hex.EncodeToString(mac.Sum(nil)) {
			t.Error("signature doesn't match transmitted bytes")
		}
		switch r.URL.Path {
		case "/api/v1/offers/powerleveling":
			if r.Method != "POST" || r.Header.Get("Content-Type") != "application/json" {
				t.Error("invalid create request")
			}
			var payload map[string]any
			if err := json.Unmarshal(body, &payload); err != nil {
				t.Error(err)
			}
			for _, key := range []string{"quantity", "description", "deliveryTime", "categories", "itemId"} {
				if _, ok := payload[key]; ok {
					t.Errorf("unexpected field %s", key)
				}
			}
			if payload["offerDesc"] != "<p>Test</p>" || payload["totalUnit"] != float64(1) {
				t.Error(payload)
			}
			fmt.Fprint(w, `{"code":10000,"data":{"offerId":71394387,"navigateURL":"https://example.com/offer","productType":"powerleveling"}}`)
		case "/api/v1/games":
			if r.Method != "GET" || len(body) != 0 {
				t.Error("GET must sign empty body")
			}
			fmt.Fprint(w, `{"code":10000,"data":[{"gameId":3637,"gameName":"League of Legends","productType":"currency,powerleveling"}]}`)
		case "/api/v1/games/3637/powerleveling/categories":
			fmt.Fprint(w, `{"code":10000,"data":[{"id":516,"parentId":0,"subCategorys":[{"id":517,"parentId":516}]}]}`)
		case "/api/v1/offers/creation-prevalidation":
			fmt.Fprint(w, `{"code":10000,"data":{"isSeller":true,"sellerLevel":3}}`)
		default:
			t.Errorf("unexpected path %s", r.URL.Path)
		}
	})
	ctx := context.Background()
	created, err := client.CreateBoostingOffer(ctx, validRequest())
	if err != nil || created.OfferID != 71394387 {
		t.Fatalf("created=%+v err=%v", created, err)
	}
	games, err := client.Games(ctx)
	if err != nil || len(games) != 1 || !games[0].SupportsBoosting() {
		t.Fatalf("games=%+v err=%v", games, err)
	}
	categories, err := client.BoostingCategories(ctx, 3637)
	if err != nil || len(categories[0].SubCategories) != 1 {
		t.Fatalf("categories=%+v err=%v", categories, err)
	}
	eligibility, err := client.PrevalidateCreation(ctx)
	if err != nil || !eligibility.IsSeller || calls != 4 {
		t.Fatalf("eligibility=%+v calls=%d err=%v", eligibility, calls, err)
	}
}

func TestAPIFailuresNeverRetryCreation(t *testing.T) {
	for _, test := range []struct {
		name     string
		status   int
		body     string
		apiError bool
	}{
		{"business", 200, `{"code":40001,"message":"quota exceeded","requestId":"req-1"}`, true},
		{"http", 429, `{"code":40001,"message":"throttled","requestId":"req-1"}`, true},
		{"html", 502, `<html>upstream down</html>`, false},
		{"missing-data", 200, `{"code":10000}`, false},
		{"null", 200, `{"code":10000,"data":null}`, false},
		{"missing-id", 200, `{"code":10000,"data":{}}`, false},
	} {
		t.Run(test.name, func(t *testing.T) {
			calls := 0
			client := mockClient(t, func(w http.ResponseWriter, r *http.Request) {
				calls++
				w.WriteHeader(test.status)
				fmt.Fprint(w, test.body)
			})
			_, err := client.CreateBoostingOffer(context.Background(), validRequest())
			if err == nil || calls != 1 {
				t.Fatalf("calls=%d err=%v", calls, err)
			}
			if test.apiError {
				var apiErr *APIError
				if !errors.As(err, &apiErr) || apiErr.RequestID != "req-1" || apiErr.Code != 40001 || apiErr.StatusCode != test.status {
					t.Fatal(err)
				}
			}
		})
	}
}

func TestCategoryHierarchy(t *testing.T) {
	child, wrong := 2, 99
	categories := []Category{{ID: 1, SubCategories: []Category{{ID: 2, ParentID: 1}}}, {ID: 3}}
	for _, test := range []struct {
		parent int
		item   *int
		valid  bool
	}{
		{1, &child, true}, {1, nil, false}, {1, &wrong, false}, {3, nil, true}, {2, nil, false}, {99, nil, false},
	} {
		err := CheckCategory(categories, test.parent, test.item)
		if (err == nil) != test.valid {
			t.Fatalf("parent=%d item=%v err=%v", test.parent, test.item, err)
		}
	}
	if err := CheckCategory([]Category{{ID: 1}, {ID: 2, ParentID: 1}}, 1, &child); err != nil {
		t.Fatal(err)
	}
}

func TestMissingAgreementDoesNotSend(t *testing.T) {
	client := mockClient(t, func(http.ResponseWriter, *http.Request) { t.Error("unexpected HTTP request") })
	request := validRequest()
	request.AgreeCheck = false
	if _, err := client.CreateBoostingOffer(context.Background(), request); err == nil {
		t.Fatal("missing agreement accepted")
	}
}
