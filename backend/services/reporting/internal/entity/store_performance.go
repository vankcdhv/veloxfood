package entity

// StorePerformance holds running rating aggregates for a store. Updated on
// each review.created event; avg_rating is recomputed inline to avoid drift.
type StorePerformance struct {
	StoreID     string  `gorm:"type:uuid;primaryKey;column:store_id"`
	RatingSum   int64   `gorm:"not null;default:0;column:rating_sum"`
	RatingCount int     `gorm:"not null;default:0;column:rating_count"`
	AvgRating   float64 `gorm:"type:numeric(3,2);not null;default:0;column:avg_rating"`
}

func (StorePerformance) TableName() string { return "store_performance" }
