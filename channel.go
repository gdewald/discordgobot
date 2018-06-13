package discordbot

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
)

// Reference:
// https://discordapp.com/developers/docs/resources/channel#channel-object-channel-structure
type Channel struct {
	Id                   string       `json:"id"`
	Type                 int          `json:"type"`
	GuildId              *string      `json:"guild_id,omitempty"`
	Position             *int         `json:"position,omitempty"`
	PermissionOverwrites *[]Overwrite `json:"permission_overwrites,omitempty"`
	Name                 *string      `json:"name,omitempty"`
	Topic                *string      `json:"topic,omitempty"`
	Nsfw                 *bool        `json:"nswf,omitempty"`
	LastMessageId        *string      `json:"last_message_id,omitempty"`
	Bitrate              *int         `json:"bitrate,omitempty"`
	UserLimit            *int         `json:"user_limit,omitempty"`
	Recipients           *[]User      `json:"recipients,omitempty"`
	Icon                 *string      `json:"icon,omitempty"`
	OwnerId              *string      `json:"owner_id,omitempty"`
	ApplicationId        *string      `json:"application_id,omitempty"`
	ParentId             *string      `json:"parent_id,omitempty"`
	LastPinTimestamp     *string      `json:"last_pin_timestamp,omitempty"`
}

// Reference:
// https://discordapp.com/developers/docs/resources/channel#overwrite-object-overwrite-structure
type Overwrite struct {
	Id    string `json:"id"`
	Type  string `json:"type"`
	Allow int    `json:"allow"`
	Deny  int    `json:"deny"`
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

type Message struct {
	Id              string  `json:"id"`
	ChannelId       string  `json:"channel_id"`
	Author          User    `json:"author*,omitempty"`
	Content         string  `json:"content"`
	Timestamp       string  `json:"timestamp"`
	EditedTimestamp *string `json:"edited_timestamp,omitempty"`
	Tts             bool    `json:"tts"`
	MentionEveryone bool    `json:"mention_everyone"`
	Mentions        []User  `json:"mentions"`
	// Mention role IDs
	MentionRoles []string `json:"mention_roles"`
	// Attachments []Attachment `json:"attachments,omitempty"`
	// Embeds []Embed `json:"embeds,omitempty"`
	// Reactions *[]Reaction `json:"reactions,omitempty"`
	Nonce      *string `json:"nonce,omitempty"`
	Pinned     bool    `json:"pinned"`
	Webhook_id *string `json:"webhook_id,omitempty"`
	Type       int     `json:"type"`
	// Activity *MessageActivity `json:"activity,omitempty"`
	// Application *MessageApplication `json:"application,omitempty"`
}

type OutgoingMessage struct {
	Content string  `json:"content"`
	Nonce   *string `json:"nonce,omitempty"`
	Tts     bool    `json:"tts"`
	// File multipart.File `json:"file,omitempty"`
	// Embed Embed `json:"embed,omitempty"``
	// PayloadJson multipart.Form `json:"payload_string,omitempty"`
}

const channelsEnpoint = "/channels"

// Send message on channel
// TODO: fix up, migrate common logic into central client function.
func (client *DiscordClient) SendMessage(channelId string, message OutgoingMessage) (sentMessage Message, err error) {
	url := fmt.Sprintf("%s/v%d/channels/%s/messages", baseUrl, apiVersion, channelId)

	log.Print("Create message URL: ", url)

	var bodyBytes []byte
	bodyBytes, err = json.Marshal(&message)

	var req *http.Request
	req, err = http.NewRequest(http.MethodPost, url, bytes.NewReader(bodyBytes))

	if err != nil {
		return
	}

	req.Header.Add("Authorization", fmt.Sprintf("%s %s", authTokenType, client.AuthToken))
	req.Header.Add("User-Agent", userAgent)
	req.Header.Add("Content-Type", "application/json")

	var resp *http.Response
	resp, err = http.DefaultClient.Do(req)

	body, _ := ioutil.ReadAll(resp.Body)
	log.Printf("Response: [%+v]. Body: [%s]", resp, body)

	if err != nil {
		return sentMessage, fmt.Errorf("failed to send message: %v", err)
	}

	err = json.NewDecoder(resp.Body).Decode(&sentMessage)

	if err == nil {
		log.Print("Sent message response: ", sentMessage)
	}

	return sentMessage, err
}
