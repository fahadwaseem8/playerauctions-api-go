package offers

import (
	"fmt"

	"github.com/fahadwaseem8/playerauctions-api-go/internal/playerauctions"
)

type CategoryMapping struct {
	ParentItemID int  `json:"parentItemId"`
	ItemID       *int `json:"itemId,omitempty"`
}

type Settings struct {
	GameID            int                        `json:"gameId"`
	OfferDuration     int                        `json:"offerDuration"`
	TotalUnit         int                        `json:"totalUnit"`
	ServiceCategories map[string]CategoryMapping `json:"serviceCategories"`
}

type PlannedOffer struct {
	ServiceID string
	Quantity  int
	Payload   playerauctions.CreateBoostingRequest
}

func Build(variants []Variant, settings Settings, agree bool) ([]PlannedOffer, error) {
	plan := make([]PlannedOffer, 0, len(variants))
	for _, variant := range variants {
		category, ok := settings.ServiceCategories[variant.ServiceID]
		if !ok {
			return nil, fmt.Errorf("service %s: add its PlayerAuctions category IDs to serviceCategories in the config", variant.ServiceID)
		}
		payload := playerauctions.CreateBoostingRequest{
			GameID: settings.GameID, Title: variant.Title, OfferDesc: variant.Description,
			OfferDuration: settings.OfferDuration, AgreeCheck: agree, Price: variant.Price,
			TotalUnit: settings.TotalUnit, ParentItemID: category.ParentItemID, ItemID: category.ItemID,
		}
		if err := payload.Validate(false); err != nil {
			return nil, fmt.Errorf("service %s: %w", variant.ServiceID, err)
		}
		plan = append(plan, PlannedOffer{ServiceID: variant.ServiceID, Quantity: variant.Quantity, Payload: payload})
	}
	return plan, nil
}
