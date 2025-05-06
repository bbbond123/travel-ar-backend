package model

import "time"

type RefreshToken struct {
	TokenID      int       `gorm:"column:token_id;primaryKey" json:"token_id"`
	UserID       int       `gorm:"column:user_id;not null" json:"user_id"`
	RefreshToken string    `gorm:"column:refresh_token;type:varchar(255);not null" json:"refresh_token"`
	ExpiresAt    time.Time `gorm:"column:expires_at;not null" json:"expires_at"`
	CreatedAt    time.Time `gorm:"column:created_at;not null;default:CURRENT_TIMESTAMP" json:"created_at"`
	Revoked      bool      `gorm:"column:revoked;not null;default:false" json:"revoked"`
}
