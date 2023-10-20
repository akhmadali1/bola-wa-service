package payment_model

type Payment struct {
	Id              int    `json:"id"`
	PhoneNumber     string `json:"phone_number"`
	UserID          int    `json:"user_id"`
	FieldMasterName string `json:"field_master_name"`
	FieldName       string `json:"field_name"`
	SubFieldName    string `json:"subfield_name"`
	CategoryField   string `json:"category_field"`
	CustomerName    string `json:"customer_name"`
	AmountFormatted string `json:"amount_formatted"`
	MatchStart      string `json:"match_start"`
	MatchEnd        string `json:"match_end"`
	CountHours      int    `json:"count_hours"`
}
