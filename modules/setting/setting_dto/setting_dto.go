// Package setting_dto memuat DTO request/response modul setting.
// Seluruhnya murni struct data (tanpa function/method).
package setting_dto

// Response adalah representasi setting untuk API.
type Response struct {
	SettingID    int64  `json:"settingID"`
	SettingGroup string `json:"settingGroup"`
	SettingKey   string `json:"settingKey"`
	SettingLabel string `json:"settingLabel"`
	SettingValue string `json:"settingValue"`
	UpdatedDate  string `json:"updatedDate"`
}

// UpdateRequest adalah body memperbarui nilai satu setting.
type UpdateRequest struct {
	SettingValue string `json:"settingValue" validate:"max=1000"`
}
