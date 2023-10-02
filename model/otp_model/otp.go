package otp_model

type OTPModel struct {
	Id          int    `json:"id"`
	PhoneNumber string `json:"phone_number"`
	UserID      int    `json:"user_id"`
	OTP         string `json:"otp"`
	Createdt    string `json:"createdt"`
	Expireddt   string `json:"expireddt"`
}
