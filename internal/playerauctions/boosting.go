package playerauctions

import (
	"context"
	"fmt"
	"math"
	"net/http"
	"strings"
)

// CreateBoostingRequest follows the official Create Boosting Offer contract.
// TotalUnit is stock of this package; package quantity belongs in the title.
type CreateBoostingRequest struct {
	GameID        int     `json:"gameId"`
	Title         string  `json:"title"`
	OfferDesc     string  `json:"offerDesc,omitempty"`
	OfferDuration int     `json:"offerDuration"`
	AgreeCheck    bool    `json:"agreeCheck"`
	Price         float64 `json:"price"`
	TotalUnit     int     `json:"totalUnit"`
	ParentItemID  int     `json:"parentItemId"`
	ItemID        *int    `json:"itemId,omitempty"`
}

func (r CreateBoostingRequest) Validate(requireAgreement bool) error {
	if r.GameID <= 0 {
		return fmt.Errorf("gameId must be positive")
	}
	if strings.TrimSpace(r.Title) == "" {
		return fmt.Errorf("title is required")
	}
	switch r.OfferDuration {
	case 3, 7, 14, 30:
	default:
		return fmt.Errorf("offerDuration must be 3, 7, 14 or 30 days")
	}
	if requireAgreement && !r.AgreeCheck {
		return fmt.Errorf("accept the Secure Seller Delivery Agreement with -agree before posting")
	}
	if math.IsNaN(r.Price) || math.IsInf(r.Price, 0) || r.Price <= 0 {
		return fmt.Errorf("price must be positive and finite")
	}
	if r.TotalUnit <= 0 {
		return fmt.Errorf("totalUnit must be positive")
	}
	if r.ParentItemID <= 0 {
		return fmt.Errorf("parentItemId must be positive")
	}
	if r.ItemID != nil && *r.ItemID <= 0 {
		return fmt.Errorf("itemId must be positive or omitted")
	}
	return nil
}

type CreatedOffer struct {
	OfferID     int64  `json:"offerId"`
	NavigateURL string `json:"navigateURL"`
	Title       string `json:"title"`
	ProductType string `json:"productType"`
}

func (c *Client) CreateBoostingOffer(ctx context.Context, request CreateBoostingRequest) (CreatedOffer, error) {
	var result CreatedOffer
	if err := request.Validate(true); err != nil {
		return result, err
	}
	err := c.do(ctx, http.MethodPost, "/api/v1/offers/powerleveling", request, &result)
	if err == nil && result.OfferID <= 0 {
		err = fmt.Errorf("creation response has no offerId; check your listings before retrying")
	}
	return result, err
}
