package discordbot_test

import (
	"os"
	"testing"
	"time"

	"github.com/gdewald/discordbot"
)

func TestConnect(t *testing.T) {
	testToken, ok := os.LookupEnv("TEST_BOT_AUTH_TOKEN")
	if !ok {
		t.SkipNow()
	}

	client := discordbot.DiscordClient{AuthToken: testToken}
	gatewayInfo, err := client.GetGateway()

	if err != nil {
		t.Fatal(err)
	}

	gateway := discordbot.DiscordGateway{
		DiscordClient: client,
		GatewayInfo:   gatewayInfo,
	}

	err = gateway.Connect()

	if err != nil {
		t.Fatal(err)
	}

	time.Sleep(time.Duration(10) * time.Second)

	t.Log(gateway, err)
}
