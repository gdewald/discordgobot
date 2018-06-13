package discordbot

// Reference:
// https://discordapp.com/developers/docs/resources/user#user-object-user-structure
type User struct {
	Id            string  `json:"id"`
	Username      string  `json:"username"`
	Discriminator string  `json:"discriminator"`
	Avatar        *string `json:"avatar"`
	Bot           *bool   `json:"bot"`
	MfaEnabled    *bool   `json:"mfa_enabled"`
	Verified      *bool   `json:"verified"`
	Email         *string `json:"email"`
}
