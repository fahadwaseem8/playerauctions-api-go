package playerauctions

import (
	"context"
	"fmt"
	"net/http"
	"strings"
)

type Game struct {
	ID          int    `json:"gameId"`
	Name        string `json:"gameName"`
	ProductType string `json:"productType"`
}

func (g Game) SupportsBoosting() bool {
	for _, product := range strings.Split(g.ProductType, ",") {
		if strings.TrimSpace(product) == "powerleveling" {
			return true
		}
	}
	return false
}

type Category struct {
	ID             int        `json:"id"`
	Name           string     `json:"name"`
	ParentID       int        `json:"parentId"`
	IsLastCategory bool       `json:"isLastCategory"`
	SubCategories  []Category `json:"subCategorys"`
}

type Prevalidation struct {
	MemberID              int64 `json:"memberId"`
	Status                int   `json:"status"`
	SellerLevel           int   `json:"sellerLevel"`
	IsSeller              bool  `json:"isSeller"`
	IsWarningTipSanctions bool  `json:"isWarningTipSanctions"`
}

func (c *Client) Games(ctx context.Context) ([]Game, error) {
	var result []Game
	err := c.do(ctx, http.MethodGet, "/api/v1/games", nil, &result)
	return result, err
}

func (c *Client) BoostingCategories(ctx context.Context, gameID int) ([]Category, error) {
	if gameID <= 0 {
		return nil, fmt.Errorf("game ID must be positive")
	}
	var result []Category
	err := c.do(ctx, http.MethodGet, fmt.Sprintf("/api/v1/games/%d/powerleveling/categories", gameID), nil, &result)
	return result, err
}

func (c *Client) PrevalidateCreation(ctx context.Context) (Prevalidation, error) {
	var result Prevalidation
	err := c.do(ctx, http.MethodGet, "/api/v1/offers/creation-prevalidation", nil, &result)
	return result, err
}

// CheckCategory accepts the documented root/optional-child hierarchy, including
// APIs returning children as flat rows instead of nested subCategorys.
func CheckCategory(categories []Category, parentID int, itemID *int) error {
	var parent *Category
	var visit func([]Category)
	visit = func(nodes []Category) {
		for i := range nodes {
			if nodes[i].ID == parentID && nodes[i].ParentID == 0 {
				parent = &nodes[i]
			}
			visit(nodes[i].SubCategories)
		}
	}
	visit(categories)
	if parent == nil {
		return fmt.Errorf("parent category %d is not a root category for this game", parentID)
	}
	children := append([]Category(nil), parent.SubCategories...)
	for _, category := range categories {
		if category.ParentID == parentID {
			children = append(children, category)
		}
	}
	if itemID == nil {
		if len(children) > 0 {
			return fmt.Errorf("parent category %d requires an itemId", parentID)
		}
		return nil
	}
	for _, child := range children {
		if child.ID == *itemID {
			return nil
		}
	}
	return fmt.Errorf("itemId %d does not belong to parent category %d", *itemID, parentID)
}
