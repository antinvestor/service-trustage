package models

import "github.com/pitabwire/frame/data"

// FormDefinition is an optional schema definition per form type.
type FormDefinition struct {
	data.BaseModel `gorm:"embedded"`

	FormID      string `gorm:"column:form_id;not null"`
	Name        string `gorm:"column:name;not null"`
	Description string `gorm:"column:description"`
	JSONSchema  string `gorm:"column:json_schema;type:jsonb"`
	Active      bool   `gorm:"column:active;not null;default:true"`
}

// TableName returns the database table name.
func (FormDefinition) TableName() string {
	return "form_definitions"
}
