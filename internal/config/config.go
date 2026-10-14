package config

import (
	"encoding/json"
	"fmt"
	"io"
	"os"

	"github.com/fahadwaseem8/playerauctions-api-go/internal/offers"
	"github.com/joho/godotenv"
)

type Environment struct {
	SupabaseURL string
	SupabaseKey string
	APIKey      string
	APISecret   string
}

func LoadEnvironment() (Environment, error) {
	if err := godotenv.Load(".env"); err != nil && !os.IsNotExist(err) {
		return Environment{}, fmt.Errorf("load .env: %w", err)
	}
	return Environment{
		SupabaseURL: os.Getenv("SUPABASE_URL"), SupabaseKey: os.Getenv("SUPABASE_KEY"),
		APIKey: os.Getenv("PLAYERAUCTIONS_API_KEY"), APISecret: os.Getenv("PLAYERAUCTIONS_API_SECRET"),
	}, nil
}

func LoadOffers(path string) (offers.Settings, error) {
	var settings offers.Settings
	f, err := os.Open(path)
	if err != nil {
		return settings, fmt.Errorf("open offer config (copy offers.config.example.json and fill in your IDs): %w", err)
	}
	defer f.Close()
	decoder := json.NewDecoder(f)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&settings); err != nil {
		return settings, fmt.Errorf("decode %s: %w", path, err)
	}
	if err := decoder.Decode(new(any)); err != io.EOF {
		return settings, fmt.Errorf("%s must contain one JSON object", path)
	}
	if settings.GameID <= 0 || settings.TotalUnit <= 0 {
		return settings, fmt.Errorf("config gameId and totalUnit must be positive")
	}
	switch settings.OfferDuration {
	case 3, 7, 14, 30:
	default:
		return settings, fmt.Errorf("config offerDuration must be 3, 7, 14 or 30")
	}
	return settings, nil
}
