package discordbot_test

import (
	"os"
	"testing"

	"github.com/gdewald/discordbot"
)

func TestGetGateway(t *testing.T) {
	testToken, ok := os.LookupEnv("TEST_BOT_AUTH_TOKEN")
	if !ok {
		t.SkipNow()
	}

	client := discordbot.DiscordClient{AuthToken: testToken}
	gateway, err := client.GetGateway()

	t.Log(gateway, err)
}
