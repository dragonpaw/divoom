// Package food has food-and-drink widgets. Currently just Cocktail, which
// pulls a random drink from TheCocktailDB.
package food

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
)

const (
	cocktailURL = "https://www.thecocktaildb.com/api/json/v1/1/random.php"
	userAgent   = "divoom-dashboard/0.1 (github.com/dragonpaw/divoom)"

	// Ingredient list caps: at most 5 names, or until total length passes
	// ingredientMaxLen, whichever happens first. Keeps the row from
	// overflowing the FontSize 28 line.
	ingredientMaxCount = 5
	ingredientMaxLen   = 80
)

// Cocktail fetches a random drink from TheCocktailDB and emits a
// pipe-separated "<image_url>|<name>|<ingredient_list>" line. The image
// URL goes into the scene's Image element via Mount.Geometry; the name
// and ingredient list go to Text elements.
type Cocktail struct{ HTTP *http.Client }

func New() *Cocktail {
	return &Cocktail{HTTP: &http.Client{Timeout: 15 * time.Second}}
}

func (c *Cocktail) Name() string { return "food/cocktail" }

// cocktailResp models only the fields we use. The API returns 15
// ingredient slots; we flatten them into a slice for easy iteration.
type cocktailResp struct {
	Drinks []struct {
		StrDrink       string `json:"strDrink"`
		StrDrinkThumb  string `json:"strDrinkThumb"`
		StrIngredient1 string `json:"strIngredient1"`
		StrIngredient2 string `json:"strIngredient2"`
		StrIngredient3 string `json:"strIngredient3"`
		StrIngredient4 string `json:"strIngredient4"`
		StrIngredient5 string `json:"strIngredient5"`
		StrIngredient6 string `json:"strIngredient6"`
		StrIngredient7 string `json:"strIngredient7"`
		StrIngredient8 string `json:"strIngredient8"`
		StrIngredient9 string `json:"strIngredient9"`
		StrIngredient10 string `json:"strIngredient10"`
		StrIngredient11 string `json:"strIngredient11"`
		StrIngredient12 string `json:"strIngredient12"`
		StrIngredient13 string `json:"strIngredient13"`
		StrIngredient14 string `json:"strIngredient14"`
		StrIngredient15 string `json:"strIngredient15"`
	} `json:"drinks"`
}

func (c *Cocktail) Fetch(ctx context.Context) (string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, cocktailURL, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("User-Agent", userAgent)
	resp, err := c.HTTP.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode/100 != 2 {
		return "", fmt.Errorf("http %d", resp.StatusCode)
	}
	var body cocktailResp
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		return "", err
	}
	if len(body.Drinks) == 0 {
		return "", fmt.Errorf("no drinks in response")
	}
	d := body.Drinks[0]

	slots := []string{
		d.StrIngredient1, d.StrIngredient2, d.StrIngredient3,
		d.StrIngredient4, d.StrIngredient5, d.StrIngredient6,
		d.StrIngredient7, d.StrIngredient8, d.StrIngredient9,
		d.StrIngredient10, d.StrIngredient11, d.StrIngredient12,
		d.StrIngredient13, d.StrIngredient14, d.StrIngredient15,
	}
	var picked []string
	for _, s := range slots {
		s = strings.TrimSpace(s)
		if s == "" {
			// API convention: first empty ingredient terminates the list.
			break
		}
		picked = append(picked, s)
		if len(picked) >= ingredientMaxCount {
			break
		}
		// Stop once joining would overflow the line.
		if len(strings.Join(picked, ", ")) >= ingredientMaxLen {
			break
		}
	}
	ingredients := strings.Join(picked, ", ")

	return d.StrDrinkThumb + "|" + d.StrDrink + "|" + ingredients, nil
}
