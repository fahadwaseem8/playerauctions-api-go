package app

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"

	"github.com/fahadwaseem8/playerauctions-api-go/internal/config"
	"github.com/fahadwaseem8/playerauctions-api-go/internal/offers"
	"github.com/fahadwaseem8/playerauctions-api-go/internal/playerauctions"
	"github.com/fahadwaseem8/playerauctions-api-go/internal/supabase"
)

func Run(ctx context.Context, args []string, stdout, stderr io.Writer) error {
	flags := flag.NewFlagSet("offers", flag.ContinueOnError)
	flags.SetOutput(stderr)
	mode := flags.String("mode", "preview", "preview, post, games, categories, or prevalidate")
	configPath := flags.String("config", "offers.config.json", "offer settings and per-service category IDs")
	gameID := flags.Int("game-id", 0, "game ID for categories mode")
	serviceID := flags.Int("service-id", 0, "generate only this Supabase service (0 means all)")
	limit := flags.Int("limit", 0, "maximum generated offers (0 means all)")
	agree := flags.Bool("agree", false, "accept the PlayerAuctions Secure Seller Delivery Agreement; required to post")
	if err := flags.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return nil
		}
		return err
	}
	if flags.NArg() != 0 {
		return fmt.Errorf("unexpected arguments: %v; use -help", flags.Args())
	}
	if *limit < 0 || *serviceID < 0 {
		return fmt.Errorf("limit and service-id cannot be negative")
	}
	switch *mode {
	case "preview", "post", "games", "categories", "prevalidate":
	default:
		return fmt.Errorf("unknown mode %q; use -help", *mode)
	}
	if *mode == "post" && !*agree {
		return fmt.Errorf("posting requires -agree to accept the Secure Seller Delivery Agreement")
	}
	if *mode == "categories" && *gameID <= 0 {
		return fmt.Errorf("categories mode requires -game-id")
	}
	env, err := config.LoadEnvironment()
	if err != nil {
		return err
	}
	var seller *playerauctions.Client
	if *mode != "preview" {
		seller, err = playerauctions.NewClient(env.APIKey, env.APISecret)
		if err != nil {
			return err
		}
	}
	switch *mode {
	case "games":
		result, err := seller.Games(ctx)
		if err != nil {
			return err
		}
		return writeJSON(stdout, result)
	case "categories":
		result, err := seller.BoostingCategories(ctx, *gameID)
		if err != nil {
			return err
		}
		return writeJSON(stdout, result)
	case "prevalidate":
		result, err := seller.PrevalidateCreation(ctx)
		if err != nil {
			return err
		}
		return writeJSON(stdout, result)
	}
	settings, err := config.LoadOffers(*configPath)
	if err != nil {
		return err
	}
	db, err := supabase.NewClient(env.SupabaseURL, env.SupabaseKey)
	if err != nil {
		return err
	}
	plan, err := prepare(ctx, db, settings, *serviceID, *limit, *agree)
	if err != nil {
		return err
	}
	if *mode == "post" {
		return publish(ctx, seller, plan, stdout, stderr, wait)
	}
	payloads := make([]playerauctions.CreateBoostingRequest, 0, len(plan))
	for _, offer := range plan {
		payloads = append(payloads, offer.Payload)
	}
	fmt.Fprintf(stderr, "Preview: %d offers; nothing posted. Category IDs are checked against PlayerAuctions in post mode.\n", len(plan))
	if !*agree {
		fmt.Fprintln(stderr, "agreeCheck is false until you accept the Secure Seller Delivery Agreement with -agree.")
	}
	return writeJSON(stdout, payloads)
}

type repository interface {
	Services(context.Context) ([]offers.Service, error)
	SharedDescription(context.Context) (string, error)
}

func prepare(ctx context.Context, db repository, settings offers.Settings, serviceID, limit int, agree bool) ([]offers.PlannedOffer, error) {
	services, err := db.Services(ctx)
	if err != nil {
		return nil, err
	}
	if serviceID > 0 {
		selected := make([]offers.Service, 0, 1)
		for _, service := range services {
			if service.ID == serviceID {
				selected = append(selected, service)
			}
		}
		if len(selected) == 0 {
			return nil, fmt.Errorf("service %d not found", serviceID)
		}
		services = selected
	}
	description, err := db.SharedDescription(ctx)
	if err != nil {
		return nil, err
	}
	variants, err := offers.Generate(services, description)
	if err != nil {
		return nil, err
	}
	if limit > 0 && len(variants) > limit {
		variants = variants[:limit]
	}
	return offers.Build(variants, settings, agree)
}

func writeJSON(w io.Writer, data any) error {
	encoder := json.NewEncoder(w)
	encoder.SetIndent("", "  ")
	encoder.SetEscapeHTML(false)
	return encoder.Encode(data)
}
