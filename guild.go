package discordbot

// Reference:
// https://discordapp.com/developers/docs/resources/guild#guild-embed-object-guild-embed-structure
type UnavailableGuild struct {
	Enabled   bool    `json:"enabled"`
	ChannelId *string `json:"channel_id"`
}

// TODO: add remaining fields
type Guild struct {
	Channels *[]Channel `json:"channels"`
}
