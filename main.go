package main

import (
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"net/url"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/joho/godotenv"
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

// OfferPayload is a preview shape, not a verified PlayerAuctions API contract.
type OfferPayload struct {
	Title        string  `json:"title"`
	Price        float64 `json:"price"`
	Quantity     int     `json:"quantity"`
	DeliveryTime string  `json:"deliveryTime"`
	Categories   string  `json:"categories"`
	Description  string  `json:"description"`
}

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run() error {
	if err := godotenv.Load(".env"); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("load .env: %w", err)
	}
	baseURL := strings.TrimRight(os.Getenv("SUPABASE_URL"), "/")
	key := os.Getenv("SUPABASE_KEY")
	if baseURL == "" || key == "" {
		return fmt.Errorf("set SUPABASE_URL and SUPABASE_KEY in .env or the environment")
	}
	parsed, err := url.Parse(baseURL)
	if err != nil || parsed.Host == "" || (parsed.Scheme != "http" && parsed.Scheme != "https") || parsed.RawQuery != "" || parsed.Fragment != "" {
		return fmt.Errorf("SUPABASE_URL must be an HTTP(S) project URL without a query or fragment")
	}
	client := &http.Client{Timeout: 30 * time.Second}
	services, err := fetchRows[Service](client, baseURL, key, "playerauctions_services", "*,playerauctions_service_categories(playerauctions_categories(name))", "id")
	if err != nil {
		return err
	}
	descriptions, err := fetchRows[Description](client, baseURL, key, "playerauctions_descriptions", "slug,content", "slug")
	if err != nil {
		return err
	}
	bySlug := make(map[string]string, len(descriptions))
	for _, description := range descriptions {
		bySlug[description.Slug] = description.Content
	}

	offers := generateOffers(services, bySlug)
	fmt.Fprintf(os.Stderr, "Preview: %d offers generated at runtime; nothing posted.\n", len(offers))
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	encoder.SetEscapeHTML(false)
	return encoder.Encode(offers)
}

// Fetch every page, including when the server's row limit is below our page size.
func fetchRows[T any](client *http.Client, baseURL, key, table, selectColumns, order string) ([]T, error) {
	rows := make([]T, 0)
	for {
		query := url.Values{
			"select": {selectColumns},
			"order":  {order + ".asc"},
			"limit":  {"1000"},
			"offset": {strconv.Itoa(len(rows))},
		}
		req, err := http.NewRequest(http.MethodGet, baseURL+"/rest/v1/"+table+"?"+query.Encode(), nil)
		if err != nil {
			return nil, fmt.Errorf("build %s request: %w", table, err)
		}
		req.Header.Set("apikey", key)
		req.Header.Set("Authorization", "Bearer "+key)
		resp, err := client.Do(req)
		if err != nil {
			return nil, fmt.Errorf("fetch %s: %w", table, err)
		}
		if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusPartialContent {
			resp.Body.Close()
			return nil, fmt.Errorf("fetch %s: HTTP %d", table, resp.StatusCode)
		}
		var page []T
		err = json.NewDecoder(io.LimitReader(resp.Body, 32<<20)).Decode(&page)
		resp.Body.Close()
		if err != nil {
			return nil, fmt.Errorf("decode %s: %w", table, err)
		}
		if len(page) == 0 {
			return rows, nil
		}
		rows = append(rows, page...)
	}
}

// Generate the same quantity, price and delivery steps as the former SQL view.
func generateOffers(services []Service, descriptions map[string]string) []OfferPayload {
	offers := make([]OfferPayload, 0)
	for _, service := range services {
		// The former view used inner joins, excluding uncategorized services.
		if len(service.Categories) == 0 {
			continue
		}
		categories := make([]string, 0, len(service.Categories))
		for _, link := range service.Categories {
			categories = append(categories, link.Category.Name)
		}
		sort.Strings(categories)
		description, ok := descriptions[strings.ToLower(service.UnitName)]
		if !ok {
			description = descriptions["general"]
		}
		for step := 1; step <= service.MaxSteps; step++ {
			quantity := service.BaseQuantity + (step-1)*service.StepIncrement
			hours := service.BaseDeliveryHours
			if service.TimeScales {
				hours *= step
			}
			delivery := fmt.Sprintf("%d Hours", hours)
			if hours >= 24 {
				delivery = fmt.Sprintf("%d Days", hours/24)
			}
			// Both database values have two decimal places. Scale them first to
			// keep half-cent rounding from drifting due to float multiplication.
			baseCents := math.Round(service.BasePrice * 100)
			modifier := math.Round(service.DiscountModifier * 100)
			priceCents := baseCents * (100 + float64(step-1)*modifier) / 100
			offers = append(offers, OfferPayload{
				Title:        fmt.Sprintf("%s - %dx %s - %s", service.BaseTitle, quantity, service.UnitName, delivery),
				Price:        math.Round(priceCents) / 100,
				Quantity:     quantity,
				DeliveryTime: delivery,
				Categories:   strings.Join(categories, " | "),
				Description:  description,
			})
		}
	}
	return offers
}
