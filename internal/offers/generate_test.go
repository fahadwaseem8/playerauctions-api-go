package offers

import (
	"encoding/json"
	"strings"
	"testing"
)

func fixture(t *testing.T) []Service {
	t.Helper()
	var services []Service
	if err := json.Unmarshal([]byte(`[{"id":1,"base_title":"Raid service","unit_name":"Raids","base_quantity":10,"step_increment":5,"max_steps":3,"base_price":10,"discount_modifier":0.95,"base_delivery_hours":12,"time_scales":true,"playerauctions_service_categories":[{"playerauctions_categories":{"name":"Raids"}}]}]`), &services); err != nil {
		t.Fatal(err)
	}
	return services
}

func TestGenerateAndBuildAPIRequests(t *testing.T) {
	variants, err := Generate(fixture(t), []Description{{"raids", "<p>Raids & more</p>"}, {"general", "Fallback"}})
	if err != nil || len(variants) != 3 {
		t.Fatalf("variants=%v err=%v", variants, err)
	}
	if variants[1].Quantity != 15 || variants[1].Price != 19.5 || variants[2].Price != 29 || !strings.HasSuffix(variants[2].Title, "36 Hours") {
		t.Fatal(variants)
	}
	settings := Settings{GameID: 123, OfferDuration: 30, TotalUnit: 1, ServiceCategories: map[int]CategoryMapping{1: {ParentItemID: 516}}}
	plan, err := Build(variants, settings, false)
	if err != nil {
		t.Fatal(err)
	}
	if plan[2].Payload.TotalUnit != 1 || plan[2].Payload.AgreeCheck || plan[2].Payload.OfferDesc != "<p>Raids & more</p>" {
		t.Fatal(plan[2])
	}
	delete(settings.ServiceCategories, 1)
	if _, err := Build(variants, settings, true); err == nil {
		t.Fatal("missing mapping accepted")
	}
}

func TestFixedDeliveryFallbackAndRounding(t *testing.T) {
	services := fixture(t)
	services[0].TimeScales = false
	services[0].BasePrice = .29
	services[0].DiscountModifier = .5
	variants, err := Generate(services, []Description{{"general", "Fallback"}})
	if err != nil {
		t.Fatal(err)
	}
	if variants[1].Price != .44 || variants[1].Description != "Fallback" || !strings.HasSuffix(variants[2].Title, "12 Hours") {
		t.Fatal(variants)
	}
	services[0].Categories = nil
	variants, err = Generate(services, nil)
	if err != nil || len(variants) != 0 {
		t.Fatal(variants, err)
	}
}
