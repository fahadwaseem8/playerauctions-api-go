package offers

import (
	"fmt"
	"math"
	"strings"
)

type Service struct {
	ID               string  `json:"id"`
	BaseTitle        string  `json:"base_title"`
	UnitName         string  `json:"unit_name"`
	BaseQuantity     int     `json:"base_quantity"`
	StepIncrement    int     `json:"step_increment"`
	MaxSteps         int     `json:"max_steps"`
	BasePrice        float64 `json:"base_price"`
	DiscountModifier float64 `json:"discount_modifier"`
	BaseDeliveryDays float64 `json:"base_delivery_days"`
	TimeScales       bool    `json:"time_scales"`
	Categories       []struct {
		Category struct {
			Name string `json:"name"`
		} `json:"playerauctions_categories"`
	} `json:"playerauctions_service_categories"`
}

type Variant struct {
	ServiceID   string
	Quantity    int
	Title       string
	Price       float64
	Description string
}

func Generate(services []Service, description string) ([]Variant, error) {
	variants := make([]Variant, 0)
	for _, service := range services {
		if len(service.Categories) == 0 {
			continue
		}
		if service.BaseQuantity <= 0 || service.StepIncrement < 0 || service.MaxSteps <= 0 || service.BaseDeliveryDays <= 0 || math.IsNaN(service.BaseDeliveryDays) || math.IsInf(service.BaseDeliveryDays, 0) {
			return nil, fmt.Errorf("service %s has invalid quantity, step count or delivery days", service.ID)
		}
		if service.BasePrice <= 0 || service.DiscountModifier < 0 || math.IsNaN(service.BasePrice) || math.IsInf(service.BasePrice, 0) || math.IsNaN(service.DiscountModifier) || math.IsInf(service.DiscountModifier, 0) {
			return nil, fmt.Errorf("service %s has invalid pricing", service.ID)
		}
		for step := 1; step <= service.MaxSteps; step++ {
			quantity := service.BaseQuantity + (step-1)*service.StepIncrement
			hours := service.BaseDeliveryDays * 24
			if service.TimeScales {
				hours *= float64(step)
			}
			// Remove floating-point conversion noise without rounding partial days down.
			hours = math.Round(hours*1e9) / 1e9
			if math.IsInf(hours, 0) || hours <= 0 {
				return nil, fmt.Errorf("service %s has invalid scaled delivery duration", service.ID)
			}
			delivery := fmt.Sprintf("%g Hours", hours)
			if math.Mod(hours, 24) == 0 {
				delivery = fmt.Sprintf("%g Days", hours/24)
			}
			baseCents := math.Round(service.BasePrice * 100)
			modifier := math.Round(service.DiscountModifier * 100)
			priceCents := baseCents * (100 + float64(step-1)*modifier) / 100
			title := service.BaseTitle
			if quantity != 1 || !strings.EqualFold(strings.TrimSpace(service.UnitName), "Package") {
				title += fmt.Sprintf(" - %dx %s", quantity, service.UnitName)
			}
			title += " - " + delivery
			variants = append(variants, Variant{
				ServiceID: service.ID, Quantity: quantity,
				Title: title,
				Price: math.Round(priceCents) / 100, Description: description,
			})
		}
	}
	return variants, nil
}
