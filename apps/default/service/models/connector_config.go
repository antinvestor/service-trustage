package models

import "github.com/pitabwire/frame/data"

// ConnectorConfig stores connector adapter configuration.
type ConnectorConfig struct {
	data.BaseModel `gorm:"embedded"`

	ConnectorType string `gorm:"column:connector_type;not null"`
	Name          string `gorm:"column:name;not null"`
	Config        string `gorm:"column:config;type:jsonb;default:'{}'"`
	Active        bool   `gorm:"column:active;not null;default:true"`
}

// TableName returns the database table name.
func (ConnectorConfig) TableName() string {
	return "connector_configs"
}
