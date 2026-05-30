package entity

import (
	"time"

	"gorm.io/gorm"
)

// Category groups MenuItems within a store.
type Category struct {
	ID        string         `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	StoreID   string         `gorm:"type:uuid;not null;index"`
	Name      string         `gorm:"type:varchar(150);not null"`
	SortOrder int            `gorm:"not null;default:0"`
	CreatedAt time.Time      `gorm:"type:timestamptz;not null;default:now()"`
	UpdatedAt time.Time      `gorm:"type:timestamptz;not null;default:now()"`
	DeletedAt gorm.DeletedAt `gorm:"type:timestamptz;index"`
}

func (Category) TableName() string { return "categories" }

// MenuItem is a dish/product sold by a store.
// Status values: on | off.
// Tags is a comma-separated string (e.g. "bestseller,new").
type MenuItem struct {
	ID          string         `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	StoreID     string         `gorm:"type:uuid;not null;index"`
	CategoryID  string         `gorm:"type:uuid;index"`
	Name        string         `gorm:"type:varchar(150);not null"`
	Description string         `gorm:"type:text"`
	Price       int64          `gorm:"not null"`
	ImageURL    string         `gorm:"type:varchar(512)"`
	Status      string         `gorm:"type:varchar(8);not null;default:'on'"`
	Tags        string         `gorm:"type:varchar(255)"`
	CreatedAt   time.Time      `gorm:"type:timestamptz;not null;default:now()"`
	UpdatedAt   time.Time      `gorm:"type:timestamptz;not null;default:now()"`
	DeletedAt   gorm.DeletedAt `gorm:"type:timestamptz;index"`
}

func (MenuItem) TableName() string { return "menu_items" }

// MenuItemSlotQuota tracks per-(item × cutoff × date) quota and sold count.
// Unique on (menu_item_id, date, cutoff_id).
type MenuItemSlotQuota struct {
	ID         string    `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	MenuItemID string    `gorm:"type:uuid;not null;uniqueIndex:uq_slot_quota,composite:menu_item_id_date_cutoff"`
	Date       time.Time `gorm:"type:date;not null;uniqueIndex:uq_slot_quota,composite:menu_item_id_date_cutoff"`
	CutoffID   string    `gorm:"type:uuid;not null;uniqueIndex:uq_slot_quota,composite:menu_item_id_date_cutoff"`
	Quota      int       `gorm:"not null"`
	SoldCount  int       `gorm:"not null;default:0"`
	CreatedAt  time.Time `gorm:"type:timestamptz;not null;default:now()"`
	UpdatedAt  time.Time `gorm:"type:timestamptz;not null;default:now()"`
}

func (MenuItemSlotQuota) TableName() string { return "menu_item_slot_quotas" }

// OptionGroup defines a topping/size selector attached to a store.
type OptionGroup struct {
	ID        string         `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	StoreID   string         `gorm:"type:uuid;not null;index"`
	Name      string         `gorm:"type:varchar(150);not null"`
	MinSelect int            `gorm:"not null;default:0"`
	MaxSelect int            `gorm:"not null;default:1"`
	Required  bool           `gorm:"not null;default:false"`
	CreatedAt time.Time      `gorm:"type:timestamptz;not null;default:now()"`
	UpdatedAt time.Time      `gorm:"type:timestamptz;not null;default:now()"`
	DeletedAt gorm.DeletedAt `gorm:"type:timestamptz;index"`
}

func (OptionGroup) TableName() string { return "option_groups" }

// Option is a single choice within an OptionGroup.
type Option struct {
	ID            string         `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	OptionGroupID string         `gorm:"type:uuid;not null;index"`
	Name          string         `gorm:"type:varchar(150);not null"`
	ExtraPrice    int64          `gorm:"not null;default:0"`
	CreatedAt     time.Time      `gorm:"type:timestamptz;not null;default:now()"`
	UpdatedAt     time.Time      `gorm:"type:timestamptz;not null;default:now()"`
	DeletedAt     gorm.DeletedAt `gorm:"type:timestamptz;index"`
}

func (Option) TableName() string { return "options" }

// MenuItemOption is the junction between a MenuItem and an OptionGroup.
// Unique on (menu_item_id, option_group_id).
type MenuItemOption struct {
	MenuItemID    string    `gorm:"type:uuid;not null;index;uniqueIndex:uq_menu_item_option"`
	OptionGroupID string    `gorm:"type:uuid;not null;index;uniqueIndex:uq_menu_item_option"`
	CreatedAt     time.Time `gorm:"type:timestamptz;not null;default:now()"`
}

func (MenuItemOption) TableName() string { return "menu_item_options" }

// Combo is a bundled deal offered by a store.
type Combo struct {
	ID        string         `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	StoreID   string         `gorm:"type:uuid;not null;index"`
	Name      string         `gorm:"type:varchar(150);not null"`
	Price     int64          `gorm:"not null"`
	CreatedAt time.Time      `gorm:"type:timestamptz;not null;default:now()"`
	UpdatedAt time.Time      `gorm:"type:timestamptz;not null;default:now()"`
	DeletedAt gorm.DeletedAt `gorm:"type:timestamptz;index"`
}

func (Combo) TableName() string { return "combos" }

// ComboItem is a line in a Combo.
type ComboItem struct {
	ComboID    string    `gorm:"type:uuid;not null;index"`
	MenuItemID string    `gorm:"type:uuid;not null"`
	Quantity   int       `gorm:"not null;default:1"`
	CreatedAt  time.Time `gorm:"type:timestamptz;not null;default:now()"`
}

func (ComboItem) TableName() string { return "combo_items" }
