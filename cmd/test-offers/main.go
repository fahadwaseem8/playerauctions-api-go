package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"

	"github.com/joho/godotenv"
)

// GeneratedOffer represents a row returned from our Supabase view
type GeneratedOffer struct {
	ServiceID         int     `json:"service_id"`
	FinalQuantity     int     `json:"final_quantity"`
	FinalDeliveryTime string  `json:"final_delivery_time"`
	GeneratedTitle    string  `json:"generated_title"`
	FinalPrice        float64 `json:"final_price"`
	Categories        string  `json:"categories"`
	DescriptionID     string  `json:"description_id"`
}

// PAOfferPayload represents the body we send to PlayerAuctions
type PAOfferPayload struct {
	Title        string  `json:"title"`
	Price        float64 `json:"price"`
	Quantity     int     `json:"quantity"`
	DeliveryTime string  `json:"deliveryTime"` 
	Description  string  `json:"description"`
}

func main() {
	// Load .env file from the current directory
	_ = godotenv.Load(".env")

	sbURL := os.Getenv("SUPABASE_URL")
	sbKey := os.Getenv("SUPABASE_KEY")

	if sbURL == "" || sbKey == "" {
		fmt.Println("Please set SUPABASE_URL and SUPABASE_KEY in your .env file")
		return
	}

	fmt.Println("========================================")
	fmt.Println("1. Fetching generated offers from Supabase API...")
	fmt.Println("========================================")

	offersURL := fmt.Sprintf("%s/rest/v1/playerauctions_generated_offers?select=*", sbURL)
	req, _ := http.NewRequest("GET", offersURL, nil)
	req.Header.Set("apikey", sbKey)
	req.Header.Set("Authorization", "Bearer "+sbKey)

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		fmt.Printf("Error fetching from Supabase: %v\n", err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		fmt.Printf("Supabase returned HTTP %d\n", resp.StatusCode)
		body, _ := io.ReadAll(resp.Body)
		fmt.Println(string(body))
		return
	}

	var generatedOffers []GeneratedOffer
	if err := json.NewDecoder(resp.Body).Decode(&generatedOffers); err != nil {
		fmt.Printf("Error decoding Supabase response: %v\n", err)
		return
	}

	fmt.Printf("Successfully fetched %d generated offers.\n\n", len(generatedOffers))

	fmt.Println("========================================")
	fmt.Println("2. Printing Generated Offers Payload...")
	fmt.Println("========================================")

	for i, offer := range generatedOffers {
		// Populate payload mapped from our database
		payload := PAOfferPayload{
			Title:        offer.GeneratedTitle,
			Price:        offer.FinalPrice,
			Quantity:     offer.FinalQuantity,
			DeliveryTime: offer.FinalDeliveryTime, 
			Description:  "Description ID: " + offer.DescriptionID, 
		}

		// Disable HTML escaping so '&' prints normally instead of '\u0026'
		var buf bytes.Buffer
		encoder := json.NewEncoder(&buf)
		encoder.SetIndent("", "  ")
		encoder.SetEscapeHTML(false)
		_ = encoder.Encode(payload)
		
		fmt.Printf("[%d/%d] JSON Payload for %s:\n%s\n", i+1, len(generatedOffers), offer.GeneratedTitle, buf.String())
	}
}
