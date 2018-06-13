package discordbot

// Reference:
// https://discordapp.com/developers/docs/resources/channel#channel-object-channel-structure
type Channel struct {
	Id                   int
	Type                 int
	GuildId              *string
	Position             *int
	PermissionOverwrites *[]Overwrite
	Name                 *string
	Topic                *string
	Nsfw                 *bool
	LastMessageId        *string
	Bitrate              *int
	UserLimit            *int
	Recipients           *[]User
	Icon                 *string
	OwnerId              *string
	ApplicationId        *string
	ParentId             *string
	LastPinTimestamp     *string
}

// Reference:
// https://discordapp.com/developers/docs/resources/channel#overwrite-object-overwrite-structure
type Overwrite struct {
	Id    string
	Type  string
	Allow int
	Deny  int
}

// Reference
// https://discordapp.com/developers/docs/resources/channel#channel-object-channel-types
const (
	ChannelTypeGuildText     = 0
	ChannelTypeDm            = 1
	ChannelTypeGuildVoice    = 2
	ChannelTypeGroupDm       = 3
	ChannelTypeGuildCategory = 4
)
