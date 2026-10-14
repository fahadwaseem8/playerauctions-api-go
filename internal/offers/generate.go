package offers

import (
	"fmt"
	"math"
	"strings"
)

type Service struct {
	ID                int     `json:"id"`
	BaseTitle         string  `json:"base_title"`
	UnitName          string  `json:"unit_name"`
	BaseQuantity      int     `json:"base_quantity"`
	StepIncrement     int     `json:"step_increment"`
	MaxSteps          int     `json:"max_steps"`
	BasePrice         float64 `json:"base_price"`
	DiscountModifier  float64 `json:"discount_modifier"`
	BaseDeliveryHours int     `json:"base_delivery_hours"`
	TimeScales        bool    `json:"time_scales"`
	Categories        []struct {
		Category struct {
			Name string `json:"name"`
		} `json:"playerauctions_categories"`
	} `json:"playerauctions_service_categories"`
}

type Description struct {
	Slug    string `json:"slug"`
	Content string `json:"content"`
}

type Variant struct {
	ServiceID   int
	Quantity    int
	Title       string
	Price       float64
	Description string
}

func Generate(services []Service, descriptions []Description) ([]Variant, error) {
	bySlug := make(map[string]string, len(descriptions))
	for _, description := range descriptions {
		bySlug[description.Slug] = description.Content
	}
	variants := make([]Variant, 0)
	for _, service := range services {
		if len(service.Categories) == 0 {
			continue
		}
		if service.BaseQuantity <= 0 || service.StepIncrement < 0 || service.MaxSteps <= 0 || service.BaseDeliveryHours <= 0 {
			return nil, fmt.Errorf("service %d has invalid quantity, step count or delivery hours", service.ID)
		}
		if service.BasePrice <= 0 || service.DiscountModifier < 0 || math.IsNaN(service.BasePrice) || math.IsInf(service.BasePrice, 0) || math.IsNaN(service.DiscountModifier) || math.IsInf(service.DiscountModifier, 0) {
			return nil, fmt.Errorf("service %d has invalid pricing", service.ID)
		}
		description, ok := bySlug[strings.ToLower(service.UnitName)]
		if !ok {
			description = bySlug["general"]
		}
		for step := 1; step <= service.MaxSteps; step++ {
			quantity := service.BaseQuantity + (step-1)*service.StepIncrement
			hours := service.BaseDeliveryHours
			if service.TimeScales {
				hours *= step
			}
			// Use days only for whole-day durations; never promise 24h for a 36h service.
			delivery := fmt.Sprintf("%d Hours", hours)
			if hours%24 == 0 {
				delivery = fmt.Sprintf("%d Days", hours/24)
			}
			baseCents := math.Round(service.BasePrice * 100)
			modifier := math.Round(service.DiscountModifier * 100)
			priceCents := baseCents * (100 + float64(step-1)*modifier) / 100
			variants = append(variants, Variant{
				ServiceID: service.ID, Quantity: quantity,
				Title: fmt.Sprintf("%s - %dx %s - %s", service.BaseTitle, quantity, service.UnitName, delivery),
				Price: math.Round(priceCents) / 100, Description: description,
			})
		}
	}
	return variants, nil
}
