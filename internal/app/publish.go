package app

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"time"

	"github.com/fahadwaseem8/playerauctions-api-go/internal/offers"
	"github.com/fahadwaseem8/playerauctions-api-go/internal/playerauctions"
)

type sellerAPI interface {
	PrevalidateCreation(context.Context) (playerauctions.Prevalidation, error)
	Games(context.Context) ([]playerauctions.Game, error)
	BoostingCategories(context.Context, int) ([]playerauctions.Category, error)
	CreateBoostingOffer(context.Context, playerauctions.CreateBoostingRequest) (playerauctions.CreatedOffer, error)
}

func publish(ctx context.Context, seller sellerAPI, plan []offers.PlannedOffer, stdout, stderr io.Writer, pause func(context.Context, time.Duration) error) error {
	if len(plan) == 0 {
		fmt.Fprintln(stderr, "No offers to post.")
		return nil
	}
	// Validate the entire batch before the first write.
	for _, offer := range plan {
		if err := offer.Payload.Validate(true); err != nil {
			return fmt.Errorf("service %d: %w", offer.ServiceID, err)
		}
	}
	eligibility, err := seller.PrevalidateCreation(ctx)
	if err != nil {
		return err
	}
	if !eligibility.IsSeller {
		return fmt.Errorf("PlayerAuctions account is not a seller")
	}
	if eligibility.IsWarningTipSanctions {
		return fmt.Errorf("PlayerAuctions reports an account sanctions warning; review it before posting")
	}
	if err := validateMetadata(ctx, seller, plan); err != nil {
		return err
	}
	// 19s stays below 200/hour; 10s stays below 400/hour and above the 5s cooldown.
	// Other processes and earlier runs may have consumed quota; errors stop this run.
	interval := 19 * time.Second
	if eligibility.SellerLevel >= 4 {
		interval = 10 * time.Second
	}
	encoder := json.NewEncoder(stdout)
	for i, offer := range plan {
		if i > 0 {
			if err := pause(ctx, interval); err != nil {
				return fmt.Errorf("stopped after %d successful creations: %w", i, err)
			}
		}
		if err := ctx.Err(); err != nil {
			return err
		}
		created, err := seller.CreateBoostingOffer(ctx, offer.Payload)
		if err != nil {
			return fmt.Errorf("stopped after %d successful creations at service %d, package quantity %d: %w; check your listings before retrying (the last request may have succeeded)", i, offer.ServiceID, offer.Quantity, err)
		}
		result := struct {
			ServiceID int `json:"serviceId"`
			Quantity  int `json:"packageQuantity"`
			playerauctions.CreatedOffer
		}{offer.ServiceID, offer.Quantity, created}
		if err := encoder.Encode(result); err != nil {
			return fmt.Errorf("offer %d was created but writing its result failed: %w", created.OfferID, err)
		}
		fmt.Fprintf(stderr, "Created %d/%d: offer %d\n", i+1, len(plan), created.OfferID)
	}
	return nil
}

func validateMetadata(ctx context.Context, seller sellerAPI, plan []offers.PlannedOffer) error {
	games, err := seller.Games(ctx)
	if err != nil {
		return err
	}
	boostingGames := make(map[int]bool)
	for _, game := range games {
		boostingGames[game.ID] = game.SupportsBoosting()
	}
	byGame := make(map[int][]playerauctions.Category)
	for _, offer := range plan {
		payload := offer.Payload
		if !boostingGames[payload.GameID] {
			return fmt.Errorf("game %d is missing or does not support boosting", payload.GameID)
		}
		categories, ok := byGame[payload.GameID]
		if !ok {
			categories, err = seller.BoostingCategories(ctx, payload.GameID)
			if err != nil {
				return err
			}
			byGame[payload.GameID] = categories
		}
		if err := playerauctions.CheckCategory(categories, payload.ParentItemID, payload.ItemID); err != nil {
			return fmt.Errorf("service %d: %w", offer.ServiceID, err)
		}
	}
	return nil
}

func wait(ctx context.Context, duration time.Duration) error {
	timer := time.NewTimer(duration)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}
