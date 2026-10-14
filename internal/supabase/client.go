package supabase

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/fahadwaseem8/playerauctions-api-go/internal/offers"
)

type Client struct {
	baseURL string
	key     string
	http    *http.Client
}

func NewClient(baseURL, key string) (*Client, error) {
	if key == "" {
		return nil, fmt.Errorf("set SUPABASE_KEY")
	}
	u, err := url.Parse(baseURL)
	if err != nil || u.Host == "" || (u.Scheme != "http" && u.Scheme != "https") || u.User != nil || u.RawQuery != "" || u.Fragment != "" {
		return nil, fmt.Errorf("SUPABASE_URL must be an HTTP(S) project URL without credentials, query or fragment")
	}
	return &Client{
		baseURL: strings.TrimRight(baseURL, "/"), key: key,
		http: &http.Client{Timeout: 30 * time.Second, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }},
	}, nil
}

func (c *Client) Services(ctx context.Context) ([]offers.Service, error) {
	return fetchRows[offers.Service](ctx, c, "playerauctions_services", "*,playerauctions_service_categories(playerauctions_categories(name))", "id")
}

func (c *Client) Descriptions(ctx context.Context) ([]offers.Description, error) {
	return fetchRows[offers.Description](ctx, c, "playerauctions_descriptions", "slug,content", "slug")
}

// Keep fetching until an empty page, even when a server cap is below 1000 rows.
func fetchRows[T any](ctx context.Context, c *Client, table, columns, order string) ([]T, error) {
	rows := make([]T, 0)
	for {
		query := url.Values{"select": {columns}, "order": {order + ".asc"}, "limit": {"1000"}, "offset": {strconv.Itoa(len(rows))}}
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+"/rest/v1/"+table+"?"+query.Encode(), nil)
		if err != nil {
			return nil, err
		}
		req.Header.Set("apikey", c.key)
		req.Header.Set("Authorization", "Bearer "+c.key)
		resp, err := c.http.Do(req)
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
