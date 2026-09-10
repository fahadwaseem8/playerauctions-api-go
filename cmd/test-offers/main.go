package main

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"
	"github.com/joho/godotenv"
)

func main() {
	// Load .env file from the current directory (where go run is executed)
	_ = godotenv.Load(".env") 
	
	// 1. Get credentials from environment (or hardcode for testing)
	apiKey := os.Getenv("PLAYERAUCTIONS_API_KEY")
	apiSecret := os.Getenv("PLAYERAUCTIONS_API_SECRET")

	if apiKey == "" || apiSecret == "" {
		fmt.Println("Please set PLAYERAUCTIONS_API_KEY and PLAYERAUCTIONS_API_SECRET in your .env file")
		return
	}

	// 2. Prepare the request
	url := "https://seller-api.playerauctions.com/api/v1/offers"
	method := "GET"
	bodyStr := "" // Empty body for GET requests

	// 3. Generate Timestamp
	timestamp := fmt.Sprintf("%d", time.Now().Unix())

	// 4. Generate Signature
	// The docs usually imply concatenating api_key + timestamp + body
	// Make sure this matches exactly how PlayerAuctions expects the string to be formed.
	message := apiKey + timestamp + bodyStr
	
	h := hmac.New(sha256.New, []byte(apiSecret))
	h.Write([]byte(message))
	signature := hex.EncodeToString(h.Sum(nil))

	fmt.Printf("Timestamp: %s\n", timestamp)
	fmt.Printf("Message to sign: %s\n", message)
	fmt.Printf("Signature: %s\n\n", signature)

	// 5. Create HTTP Request
	req, err := http.NewRequest(method, url, nil)
	if err != nil {
		fmt.Printf("Error creating request: %v\n", err)
		return
	}

	// 6. Set Headers
	req.Header.Set("X-PA-API-KEY", apiKey)
	req.Header.Set("X-PA-TIMESTAMP", timestamp)
	req.Header.Set("X-PA-SIGN", signature)
	req.Header.Set("Content-Type", "application/json") // Standard for most APIs

	// 7. Execute Request
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		fmt.Printf("Error executing request: %v\n", err)
		return
	}
	defer resp.Body.Close()

	// 8. Read and Print Response
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Printf("Error reading response: %v\n", err)
		return
	}

	fmt.Printf("Status Code: %d\n", resp.StatusCode)
	fmt.Printf("Response: %s\n", string(respBody))
}
