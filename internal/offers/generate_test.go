package offers

import (
	"encoding/json"
	"math"
	"strings"
	"testing"
)

func fixture(t *testing.T) []Service {
	t.Helper()
	var services []Service
	if err := json.Unmarshal([]byte(`[{"id":"11111111-1111-4111-8111-111111111111","base_title":"Raid service","unit_name":"Raids","base_quantity":10,"step_increment":5,"max_steps":3,"base_price":10,"discount_modifier":0.95,"base_delivery_days":0.5,"time_scales":true,"playerauctions_service_categories":[{"playerauctions_categories":{"name":"Raids"}}]}]`), &services); err != nil {
		t.Fatal(err)
	}
	return services
}

func TestDeliveryDaysAndSharedDescription(t *testing.T) {
	for _, test := range []struct {
		days float64
		want string
	}{{11, "11 Days"}, {0.125, "3 Hours"}, {1.5, "36 Hours"}, {1.0 / 24, "1 Hours"}} {
		services := fixture(t)
		services[0].BaseDeliveryDays = test.days
		services[0].TimeScales = false
		other := services[0]
		other.ID = "22222222-2222-4222-8222-222222222222"
		other.UnitName = "Package"
		other.BaseQuantity = 1
		other.MaxSteps = 1
		variants, err := Generate(append(services, other), "Shared HTML")
		if err != nil {
			t.Fatal(err)
		}
		for _, v := range variants {
			if !strings.HasSuffix(v.Title, test.want) || v.Description != "Shared HTML" {
				t.Fatalf("days=%v variant=%+v", test.days, v)
			}
		}
		if strings.Contains(variants[len(variants)-1].Title, "1x Package") {
			t.Fatal(variants)
		}
	}
}

func TestInvalidDeliveryDays(t *testing.T) {
	for _, days := range []float64{0, -1, math.NaN(), math.Inf(1)} {
		services := fixture(t)
		services[0].BaseDeliveryDays = days
		if _, err := Generate(services, "Shared"); err == nil {
			t.Fatalf("accepted %v days", days)
		}
	}
}

func TestGenerateAndBuildAPIRequests(t *testing.T) {
	variants, err := Generate(fixture(t), "<p>Raids & more</p>")
	if err != nil || len(variants) != 3 {
		t.Fatalf("variants=%v err=%v", variants, err)
	}
	if variants[1].Quantity != 15 || variants[1].Price != 19.5 || variants[2].Price != 29 || !strings.HasSuffix(variants[2].Title, "36 Hours") {
		t.Fatal(variants)
	}
	settings := Settings{GameID: 123, OfferDuration: 30, TotalUnit: 1, ServiceCategories: map[string]CategoryMapping{"11111111-1111-4111-8111-111111111111": {ParentItemID: 516}}}
	plan, err := Build(variants, settings, false)
	if err != nil {
		t.Fatal(err)
	}
	if plan[2].Payload.TotalUnit != 1 || plan[2].Payload.AgreeCheck || plan[2].Payload.OfferDesc != "<p>Raids & more</p>" {
		t.Fatal(plan[2])
	}
	delete(settings.ServiceCategories, "11111111-1111-4111-8111-111111111111")
	if _, err := Build(variants, settings, true); err == nil {
		t.Fatal("missing mapping accepted")
	}
}

func TestFixedDeliveryFallbackAndRounding(t *testing.T) {
	services := fixture(t)
	services[0].TimeScales = false
	services[0].BasePrice = .29
	services[0].DiscountModifier = .5
	variants, err := Generate(services, "Fallback")
	if err != nil {
		t.Fatal(err)
	}
	if variants[1].Price != .44 || variants[1].Description != "Fallback" || !strings.HasSuffix(variants[2].Title, "12 Hours") {
		t.Fatal(variants)
	}
	services[0].Categories = nil
	variants, err = Generate(services, "")
	if err != nil || len(variants) != 0 {
		t.Fatal(variants, err)
	}
}
