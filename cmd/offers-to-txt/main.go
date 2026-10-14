package main

import (
	"encoding/binary"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"strings"
	"unicode/utf16"
)

// Pokemon GO category IDs returned by PlayerAuctions.
var categories = map[int]string{
	3295: "Custom Offer", 4066: "Item Farming", 523: "Levelling",
	4068: "PokeCoins Farming", 4615: "Pokemon Catching", 4069: "Pokemon Trading",
	5683: "Shundo Farming", 4067: "Stardust Farming",
}

func main() {
	input := flag.String("input", "offers.json", "input JSON file")
	output := flag.String("output", "offers.txt", "output text file (overwritten)")
	flag.Parse()
	if err := convert(*input, *output); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func convert(input, output string) error {
	data, err := os.ReadFile(input)
	if err != nil {
		return err
	}
	// Windows PowerShell redirects native command output as UTF-16 with a BOM.
	if len(data) >= 2 && ((data[0] == 0xff && data[1] == 0xfe) || (data[0] == 0xfe && data[1] == 0xff)) {
		if len(data)%2 != 0 {
			return fmt.Errorf("invalid UTF-16 input length")
		}
		var order binary.ByteOrder = binary.LittleEndian
		if data[0] == 0xfe {
			order = binary.BigEndian
		}
		units := make([]uint16, (len(data)-2)/2)
		for i := range units {
			units[i] = order.Uint16(data[2+i*2:])
		}
		data = []byte(string(utf16.Decode(units)))
	}
	var offers []struct {
		Title        string  `json:"title"`
		Price        float64 `json:"price"`
		ParentItemID int     `json:"parentItemId"`
	}
	// Accept a UTF-8 BOM from PowerShell output.
	if err := json.Unmarshal([]byte(strings.TrimPrefix(string(data), "\uFEFF")), &offers); err != nil {
		return fmt.Errorf("read offers: %w", err)
	}
	if offers == nil {
		return fmt.Errorf("expected a JSON array of offers")
	}
	var text strings.Builder
	for i, offer := range offers {
		category, ok := categories[offer.ParentItemID]
		if !ok {
			return fmt.Errorf("offer %d: unknown category ID %d; add it to the categories map", i+1, offer.ParentItemID)
		}
		if strings.TrimSpace(offer.Title) == "" || strings.ContainsAny(offer.Title, "|\r\n") {
			return fmt.Errorf("offer %d: title must be nonempty and cannot contain pipes or newlines", i+1)
		}
		if offer.Price <= 0 {
			return fmt.Errorf("offer %d: price must be positive", i+1)
		}
		fmt.Fprintf(&text, "%s | %.2f | %s\n", offer.Title, offer.Price, category)
	}
	if err := os.WriteFile(output, []byte(text.String()), 0644); err != nil {
		return err
	}
	fmt.Fprintf(os.Stderr, "Wrote %d offers to %s\n", len(offers), output)
	return nil
}
