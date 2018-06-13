package discordbot_test

import (
	"encoding/json"
	"log"
	"os"
	"testing"
	"time"

	"github.com/gdewald/discordbot"
)

// Test should only be run manually - there is a limit on number of identify requests in a time period.
// TODO: make more generic.
func TestConnectAndIdentify(t *testing.T) {
	testToken, ok := os.LookupEnv("TEST_BOT_AUTH_TOKEN")
	if !ok {
		t.SkipNow()
	}

	client := discordbot.DiscordClient{AuthToken: testToken}
	gatewayInfo, err := client.GetGateway()

	if err != nil {
		t.Fatal(err)
	}

	gateway := &discordbot.DiscordGateway{
		DiscordClient: client,
		GatewayInfo:   gatewayInfo,
	}

	err = gateway.Connect()

	gateway.RegisterEventListener(discordbot.EventGuildCreate, func(payload discordbot.GatewayPayload) {
		log.Print("Event listener called.")
		eventData := payload.EventData

		guildCreate := discordbot.Guild{}
		json.Unmarshal(eventData, &guildCreate)
		log.Printf("Guild create: %+v", guildCreate)

		if err != nil {
			log.Print(err)
			return
		}

		if channels := *guildCreate.Channels; channels != nil {
			message := discordbot.OutgoingMessage{
				Content: "3 monitors and a neck beard.",
				Tts:     false,
			}
			log.Printf("Sending message: %+v", message)

			for _, channel := range channels {
				if channel.Name != nil && *channel.Name == "general" {
					sentMessage, err := client.SendMessage(channel.Id, message)

					if err != nil {
						log.Print(err)
						return
					}

					log.Print(sentMessage)

				}
			}
		} else {
			log.Print("No channels!")
		}
	})

	if err != nil {
		t.Fatal(err)
	}

	t.Log(gateway)

	user, err := gateway.Identify(nil)

	if err != nil {
		t.Fatal(err)
	}

	t.Log(user)

	time.Sleep(time.Duration(5) * time.Minute)
}
