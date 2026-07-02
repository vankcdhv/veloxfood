package entity

import "time"

// FavoriteTargetType discriminates what a favorite bookmarks.
type FavoriteTargetType string

const (
	FavoriteTargetStore FavoriteTargetType = "STORE"
	FavoriteTargetItem  FavoriteTargetType = "ITEM"
)

// Favorite is a customer's bookmark of a store or a menu item. target_id is a
// soft reference — no cross-table FK, deleted targets just drop out of joins.
type Favorite struct {
	ID         string             `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"ID"`
	UserID     string             `gorm:"type:uuid;not null" json:"UserID"`
	TargetType FavoriteTargetType `gorm:"type:varchar(10);not null" json:"TargetType"`
	TargetID   string             `gorm:"type:uuid;not null" json:"TargetID"`
	CreatedAt  time.Time          `json:"CreatedAt"`
}

func (Favorite) TableName() string { return "favorites" }
