package discordbot

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
)

// Refer to https://discordapp.com/developers/docs/reference
const baseUrl = "https://discordapp.com/api"
const apiVersion = 6
const authTokenType = "Bot"
const userAgent = "DiscordGoBot 0.0.1"

type DiscordClient struct {
	AuthToken string
}

const botGetGatewayEndpoint = "/gateway/bot"

// Gateway connection details.
// https://discordapp.com/developers/docs/topics/gateway#get-gateway-bot
type gatewayInfo struct {
	Url    string
	Shards int
}

func (client *DiscordClient) GetGateway() (gateway gatewayInfo, err error) {
	url := baseUrl + "/v" + strconv.FormatInt(apiVersion, 10) + botGetGatewayEndpoint

	log.Print("Get gateway URL: ", url)

	var req *http.Request
	req, err = http.NewRequest(http.MethodGet, url, nil)

	if err != nil {
		return
	}

	req.Header.Add("Authorization", fmt.Sprintf("%s %s", authTokenType, client.AuthToken))
	req.Header.Add("User-Agent", userAgent)

	var resp *http.Response
	resp, err = http.DefaultClient.Do(req)

	if err != nil {
		return gateway, fmt.Errorf("failed to get gateway: %v", err)
	}

	err = json.NewDecoder(resp.Body).Decode(&gateway)
	log.Print("Gateway response: ", gateway)
	return gateway, err
}

// TODO: add webhook support
// type WebhookData struct {
// 	Username  *string
// 	AvatarURL *string `json:"avatar_url"`
// 	TTS       *bool   `json:tts`
// 	File      *multipart.File
// 	Content   *string
// 	// TODO: add embed structs
// 	// Embeds []Embed
// }

// func (d DiscordClient) executeWebhook(waitForConfirm bool) string, error {

// }
