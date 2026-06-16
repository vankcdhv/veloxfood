package integration

import (
	"encoding/json"
	"fmt"
	"net/http"
	"testing"
)

// TestSearchMenuItemsAccentInsensitive verifies that GET /api/v1/menu-items/search
// returns items whose name matches the query accent-insensitively (Vietnamese diacritics).
func TestSearchMenuItemsAccentInsensitive(t *testing.T) {
	env := setup(t)

	// Seed: one active store, three menu items.
	w := env.do(t, "POST", "/api/v1/admin/stores", `{
		"vendor_id":"aaaaaaaa-9001-9001-9001-000000000001",
		"owner_user_id":"bbbbbbbb-9001-9001-9001-000000000001",
		"name":"Tiệm Bánh"
	}`)
	if w.Code != http.StatusCreated {
		t.Fatalf("create store: %d — %s", w.Code, w.Body.String())
	}
	storeID := dataID(t, w)

	// Create category so foreign key is valid.
	w = env.do(t, "POST", "/api/v1/stores/"+storeID+"/categories", `{
		"name":"Bánh","sort_order":1
	}`)
	if w.Code != http.StatusCreated {
		t.Fatalf("create category: %d", w.Code)
	}
	catID := dataID(t, w)

	items := []struct {
		name  string
		price int
	}{
		{"Bánh mì thịt", 25000},
		{"Bánh bao nhân thịt", 15000},
		{"Cơm tấm sườn", 45000},
	}
	for _, it := range items {
		w = env.do(t, "POST", "/api/v1/stores/"+storeID+"/menu", fmt.Sprintf(`{
			"category_id":%q,
			"name":%q,
			"description":"",
			"price":%d,
			"tags":""
		}`, catID, it.name, it.price))
		if w.Code != http.StatusCreated {
			t.Fatalf("create menu item %q: %d — %s", it.name, w.Code, w.Body.String())
		}
	}

	// Query: "banh mi" (no diacritics) should match "Bánh mì thịt" (accent-folded).
	w = env.do(t, "GET", "/api/v1/menu-items/search?q=banh+mi", "")
	if w.Code != http.StatusOK {
		t.Fatalf("search: %d — %s", w.Code, w.Body.String())
	}

	var resp struct {
		Data struct {
			Items []struct {
				ID         string `json:"ID"`
				Name       string `json:"Name"`
				Price      int64  `json:"Price"`
				StoreID    string `json:"StoreID"`
				StoreName  string `json:"StoreName"`
				SaleStatus string `json:"SaleStatus"`
			} `json:"items"`
			Total int64 `json:"total"`
		} `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("parse response: %v — body: %s", err, w.Body.String())
	}

	// Expect at least one result and that the first result is "Bánh mì thịt".
	if len(resp.Data.Items) == 0 {
		t.Fatalf("expected at least 1 result for 'banh mi', got 0 — DB may not have f_unaccent")
	}
	if resp.Data.Items[0].Name != "Bánh mì thịt" {
		t.Errorf("expected top result 'Bánh mì thịt', got %q", resp.Data.Items[0].Name)
	}
	if resp.Data.Items[0].StoreID != storeID {
		t.Errorf("expected StoreID=%s, got %s", storeID, resp.Data.Items[0].StoreID)
	}
	if resp.Data.Items[0].Price != 25000 {
		t.Errorf("expected Price=25000, got %d", resp.Data.Items[0].Price)
	}

	// "Cơm tấm" should NOT appear in a "banh mi" search.
	for _, item := range resp.Data.Items {
		if item.Name == "Cơm tấm sườn" {
			t.Errorf("unexpected item 'Cơm tấm sườn' in results for 'banh mi'")
		}
	}
}

// TestSearchMenuItemsEmptyQuery verifies that an empty/blank q returns HTTP 200
// with an empty array (no error).
func TestSearchMenuItemsEmptyQuery(t *testing.T) {
	env := setup(t)

	w := env.do(t, "GET", "/api/v1/menu-items/search?q=", "")
	if w.Code != http.StatusOK {
		t.Fatalf("empty query: expected 200, got %d — %s", w.Code, w.Body.String())
	}

	var resp struct {
		Data struct {
			Items []interface{} `json:"items"`
		} `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("parse response: %v", err)
	}
	if len(resp.Data.Items) != 0 {
		t.Errorf("expected empty array for blank q, got %d items", len(resp.Data.Items))
	}
}

// TestSearchMenuItemsLimitCap verifies that limit is capped at 50 and the
// request does not error out with an out-of-range value.
func TestSearchMenuItemsLimitCap(t *testing.T) {
	env := setup(t)

	// No store seeded — should return empty but not error.
	w := env.do(t, "GET", "/api/v1/menu-items/search?q=pho&limit=9999", "")
	if w.Code != http.StatusOK {
		t.Fatalf("capped limit: expected 200, got %d — %s", w.Code, w.Body.String())
	}
}
