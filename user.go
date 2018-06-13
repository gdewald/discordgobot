package discordbot

// Reference:
// https://discordapp.com/developers/docs/resources/user#user-object-user-structure
type User struct {
	Id            string
	Username      string
	Discriminator string
	Avatar        *string
	Bot           *bool
	MfaEnabled    *bool `json:"mfa_enabled"`
	Verified      *bool
	Email         *string
}
