// cmd/seed-catalog seeds store_db with 20 demo stores, each holding 45-50 menu
// items across 4 categories, 3 ship cutoffs (ca giao), a building-scope ship-fee
// rule (so delivery resolves), and per-(item × cutoff × day) slot quotas for the
// next 10 days. Idempotent: re-running skips already-seeded stores unless -reset.
//
// Usage: go run ./cmd/seed-catalog [-reset]
package main

import (
	"flag"
	"fmt"
	"log/slog"
	"math/rand"
	"os"
	"time"

	"project/pkg/config"
	"project/services/store/internal/entity"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

const (
	ownerUserID = "a835f18b-4877-4bda-99d1-dbf8ae09d649" // reuse existing owner so names resolve
	buildingID  = "654f59c8-585b-4580-8e82-907f68574409" // Toà A — building-scope ship fee covers all rooms
	quotaDays   = 10                                      // seed slot quotas for today..+9
)

var cutoffs = []struct {
	Time string
	Lead int
}{{"11:00", 30}, {"17:30", 30}, {"22:00", 30}}

func seedVendorID(i int) string { return fmt.Sprintf("5eed0000-0000-0000-0000-0000000000%02d", i) }

func main() {
	reset := flag.Bool("reset", false, "delete previously seeded stores then re-seed")
	flag.Parse()

	cfg, err := config.Load("config/store.yaml")
	if err != nil {
		fail("load config", err)
	}
	d := cfg.Database
	dsn := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		d.Host, d.Port, d.User, d.Password, d.Name, orDisable(d.SSLMode))
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		fail("open db", err)
	}

	vendorIDs := make([]string, 20)
	for i := 1; i <= 20; i++ {
		vendorIDs[i-1] = seedVendorID(i)
	}

	var existing int64
	db.Model(&entity.Store{}).Where("vendor_id IN ?", vendorIDs).Count(&existing)
	if existing > 0 {
		if !*reset {
			slog.Info("[seed] already seeded — skipping (use -reset to rebuild)", "stores", existing)
			return
		}
		deleteSeeded(db, vendorIDs)
		slog.Info("[seed] reset: removed previously seeded stores")
	}

	today := time.Now().Truncate(24 * time.Hour)
	totItems, totQuotas := 0, 0

	for i := 1; i <= 20; i++ {
		theme := Themes[i-1]
		rng := rand.New(rand.NewSource(int64(i) * 7919))

		store := &entity.Store{
			OwnerUserID:   ownerUserID,
			VendorID:      seedVendorID(i),
			Name:          theme.Store,
			BusinessType:  theme.BusinessType,
			Address:       "Toà A – Khu ẩm thực tầng trệt",
			Phone:         fmt.Sprintf("09%08d", rng.Intn(100000000)),
			SaleStatus:    "OPEN",
			PickupEnabled: i%2 == 0,
		}
		if err := db.Create(store).Error; err != nil {
			fail("create store "+theme.Store, err)
		}

		// Categories (always all four).
		catID := map[string]string{}
		for order, name := range []string{catMain, catSide, catDrink, catDessert} {
			c := &entity.Category{StoreID: store.ID, Name: name, SortOrder: order}
			if err := db.Create(c).Error; err != nil {
				fail("create category", err)
			}
			catID[name] = c.ID
		}

		// Ship cutoffs (ca giao) + collect IDs for quota rows.
		cutoffIDs := make([]string, 0, len(cutoffs))
		for _, co := range cutoffs {
			sc := &entity.ShipCutoff{StoreID: store.ID, CutoffTime: co.Time, LeadMinutes: co.Lead}
			if err := db.Create(sc).Error; err != nil {
				fail("create cutoff", err)
			}
			cutoffIDs = append(cutoffIDs, sc.ID)
		}

		// Building-scope ship fee so delivery to any room in Toà A resolves.
		fee := &entity.ShipFeeRule{StoreID: store.ID, Scope: "building", RefID: buildingID,
			UnitFee: int64(12000 + rng.Intn(7)*1000)}
		if err := db.Create(fee).Error; err != nil {
			fail("create ship fee", err)
		}

		// Compose 45-50 unique items: all themed mains + a rotated slice of the
		// shared pools (so each store's drinks/sides/desserts differ).
		menu := composeMenu(theme, i)
		var quotas []*entity.MenuItemSlotQuota
		for idx, mi := range menu {
			item := &entity.MenuItem{
				StoreID:     store.ID,
				CategoryID:  catID[mi.Cat],
				Name:        mi.Dish.Name,
				Description: mi.Dish.Desc,
				Price:       priceIn(rng, mi.Dish),
				ImageURL:    fmt.Sprintf("https://loremflickr.com/400/400/%s?lock=%d", mi.Dish.Keyword, i*100+idx),
				Status:      "on",
				Tags:        tagFor(idx),
			}
			if err := db.Create(item).Error; err != nil {
				fail("create menu item", err)
			}
			totItems++
			for dayOff := 0; dayOff < quotaDays; dayOff++ {
				date := today.AddDate(0, 0, dayOff)
				for _, cid := range cutoffIDs {
					quotas = append(quotas, &entity.MenuItemSlotQuota{
						MenuItemID: item.ID, Date: date, CutoffID: cid,
						Quota: 25 + rng.Intn(36), SoldCount: 0,
					})
				}
			}
		}
		if err := db.CreateInBatches(quotas, 500).Error; err != nil {
			fail("create slot quotas", err)
		}
		totQuotas += len(quotas)
		slog.Info("[seed] store created", "name", theme.Store, "items", len(menu))
	}

	slog.Info("[seed] done", "stores", 20, "items", totItems, "quotas", totQuotas)
}

// menuItem pairs a Dish with the category it belongs to in the seeded store.
type menuItem struct {
	Dish Dish
	Cat  string
}

// composeMenu returns 45-50 unique items: every themed main plus a rotated
// slice of the shared pools (rotation varies the selection per store).
func composeMenu(theme Theme, storeIdx int) []menuItem {
	target := 45 + (storeIdx % 6) // 45..50

	out := make([]menuItem, 0, target)
	for _, dsh := range theme.Mains {
		out = append(out, menuItem{dsh, catMain})
	}

	shared := make([]menuItem, 0, len(Sides)+len(Drinks)+len(Desserts))
	for _, dsh := range Sides {
		shared = append(shared, menuItem{dsh, catSide})
	}
	for _, dsh := range Drinks {
		shared = append(shared, menuItem{dsh, catDrink})
	}
	for _, dsh := range Desserts {
		shared = append(shared, menuItem{dsh, catDessert})
	}

	// Rotate so different stores drop different shared items.
	off := (storeIdx * 5) % len(shared)
	rotated := append(append([]menuItem{}, shared[off:]...), shared[:off]...)

	for _, mi := range rotated {
		if len(out) >= target {
			break
		}
		out = append(out, mi)
	}
	return out
}

// priceIn returns a price within the dish's range, rounded to the nearest 1000đ.
func priceIn(rng *rand.Rand, d Dish) int64 {
	if d.PriceMax <= d.PriceMin {
		return d.PriceMin
	}
	p := d.PriceMin + rng.Int63n(d.PriceMax-d.PriceMin+1)
	return (p / 1000) * 1000
}

func tagFor(idx int) string {
	switch {
	case idx%9 == 0:
		return "bestseller"
	case idx%13 == 0:
		return "new"
	default:
		return ""
	}
}

// deleteSeeded hard-deletes seeded stores and their children in FK order.
func deleteSeeded(db *gorm.DB, vendorIDs []string) {
	var storeIDs []string
	db.Model(&entity.Store{}).Where("vendor_id IN ?", vendorIDs).Pluck("id", &storeIDs)
	if len(storeIDs) == 0 {
		return
	}
	db.Exec(`DELETE FROM menu_item_slot_quotas WHERE menu_item_id IN (SELECT id FROM menu_items WHERE store_id IN ?)`, storeIDs)
	for _, tbl := range []string{"menu_items", "categories", "ship_cutoffs", "ship_fee_rules"} {
		db.Exec(fmt.Sprintf("DELETE FROM %s WHERE store_id IN ?", tbl), storeIDs)
	}
	db.Exec(`DELETE FROM stores WHERE id IN ?`, storeIDs)
}

func orDisable(s string) string {
	if s == "" {
		return "disable"
	}
	return s
}

func fail(what string, err error) {
	slog.Error("[seed] "+what+" failed", "error", err)
	os.Exit(1)
}
