package models

import (
	"time"

	"gorm.io/gorm"
)

type Deployment struct {
	ID          uint           `gorm:"primarykey" json:"id"`
	UserID      uint           `json:"user_id"`
	DeployID    string         `gorm:"uniqueIndex;not null" json:"deploy_id"`
	RepoURL     string         `gorm:"not null" json:"repo_url"`
	Status      string         `gorm:"default:'pending'" json:"status"` // pending, cloning, uploading, building, deployed, failed
	DeployedURL string         `json:"deployed_url,omitempty"`
	BuildLog    string         `gorm:"type:text" json:"build_log,omitempty"`
	ErrorMsg    string         `gorm:"type:text" json:"error_msg,omitempty"`
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
	DeletedAt   gorm.DeletedAt `gorm:"index" json:"-"`

	User User `gorm:"foreignKey:UserID" json:"user,omitempty"`
}
