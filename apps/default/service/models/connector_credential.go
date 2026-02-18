package models

import "github.com/pitabwire/frame/data"

// ConnectorCredential stores encrypted credentials for connector adapters.
type ConnectorCredential struct {
	data.BaseModel `gorm:"embedded"`

	ConnectorType  string `gorm:"column:connector_type;not null"`
	CredentialBlob string `gorm:"column:credential_blob;not null"`
	KeyVersion     int    `gorm:"column:key_version;not null;default:1"`
}

// TableName returns the database table name.
func (ConnectorCredential) TableName() string {
	return "connector_credentials"
}
