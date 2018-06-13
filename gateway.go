package discordbot

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"runtime"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

// Opcode constants.
// https://discordapp.com/developers/docs/topics/opcodes-and-status-codes#gateway-gateway-opcodes
const (
	// Receive: dispatches an event
	OpcodeDispatch = 0
	// Send/Receive: used for ping checking
	OpcodeHeartbeat = 1
	// Send: used for client handshake
	OpcodeIdentify = 2
	// Send: used to update the client status
	OpcodeStatusUpdate = 3
	// Send: used to join/move/leave voice channels
	OpcodeVoiceStateUpdate = 4
	// Send: used for voice ping checking
	OpcodeVoiceServerPing = 5
	// Send: used to resume a closed connection
	OpcodeResume = 6
	// Receive: used to tell clients to reconnect to the gateway
	OpcodeReconnect = 7
	// Send: used to request guild members
	OpcodeRequestGuildMembers = 8
	// Receive: used to notify client they have an invalid session id
	OpcodeInvalidSession = 9
	// Receive: sent immediately after connecting, contains heartbeat and server debug information
	OpcodeHello = 10
	// Receive: sent immediately following a client heartbeat that was received
	OpcodeHeartbeatACK = 11
)

// Error code constants.
// https://discordapp.com/developers/docs/topics/opcodes-and-status-codes#gateway-gateway-close-event-codes
const (
	// We're not sure what went wrong. Try reconnecting?
	CloseUnknownError = 4000
	// You sent an invalid Gateway opcode or an invalid payload for an opcode. Don't do that!
	CloseUnknownOpcode = 4001
	// You sent an invalid payload to us. Don't do that!
	CloseDecodeError = 4002
	// You sent us a payload prior to identifying.
	CloseNotAuthenticated = 4003
	// The account token sent with your identify payload is incorrect.
	CloseAuthenticationFailed = 4004
	// You sent more than one identify payload. Don't do that!
	CloseAlreadyAuthenticated = 4005
	// The sequence sent when resuming the session was invalid. Reconnect and start a new session.
	CloseInvalidSeq = 4007
	// Woah nelly! You're sending payloads to us too quickly. Slow it down!
	CloseRateLimited = 4008
	// Your session timed out. Reconnect and start a new one.
	CloseSessionTimeout = 4009
	// You sent us an invalid shard when identifying.
	CloseInvalidShard = 4010
	// The session would have handled too many guilds - you are required to shard your connection in order to connect.
	CloseCloseShardingRequired = 4011
)

// https://discordapp.com/developers/docs/topics/gateway#gateways-gateway-versions
const gatewayVersion = 6
const gatewayEncoding = "json"

type DiscordGateway struct {
	DiscordClient
	GatewayInfo gatewayInfo
	// TODO: use sync.Map
	opcodeListeners map[int][]GatewayMessageListener
	eventListeners  map[string][]GatewayMessageListener
	conn            *websocket.Conn
	connMutex       *sync.Mutex
	heartbeat       *discordHeartbeat
	sessionId       *string
	sequenceNumber  *int
}

func (g *DiscordGateway) SendPayload(payload *GatewayPayload) (err error) {
	g.connMutex.Lock()

	log.Printf("Sending payload with Opcode [%v], event name [%s], data [%s], and sequenceNum [%v].",
		payload.Opcode, payload.EventName, payload.EventData, payload.SequenceNumber)
	err = g.conn.WriteJSON(payload)
	if err != nil {
		log.Print("Error during send.", err)
	} else {
		log.Printf("Done.")
	}

	g.connMutex.Unlock()
	return
}

func (g *DiscordGateway) SendControl(messageType int, data []byte, deadline time.Time) (err error) {
	g.connMutex.Lock()

	err = g.conn.WriteControl(messageType, data, deadline)

	g.connMutex.Unlock()
	return
}

// Reference: https://discordapp.com/developers/docs/topics/gateway#payloads-gateway-payload-structure
type GatewayPayload struct {
	Opcode         int             `json:"op"`
	EventName      string          `json:"t,omitempty"`
	EventData      json.RawMessage `json:"d"`
	SequenceNumber *int            `json:"s,omitempty"`
}

type GatewayMessageListener func(GatewayPayload)

// Register listener that is called when the opcode is received.
func (g *DiscordGateway) RegisterOpcodeListener(opcode int, listener GatewayMessageListener) {
	if g.opcodeListeners == nil {
		g.opcodeListeners = make(map[int][]GatewayMessageListener)
	}
	g.opcodeListeners[opcode] = append(g.opcodeListeners[opcode], listener)
}

// Register listener that is called when a named event is received (OpcodeDispatch only).
func (g *DiscordGateway) RegisterEventListener(event string, listener GatewayMessageListener) {
	if g.eventListeners == nil {
		g.eventListeners = make(map[string][]GatewayMessageListener)
	}
	g.eventListeners[event] = append(g.eventListeners[event], listener)
}

// Connects to gateway, starts heartbeat, initializes listeners for gateway.
func (g *DiscordGateway) Connect() (err error) {
	dialer := websocket.Dialer{}

	connectUrl := g.GatewayInfo.Url + fmt.Sprintf("/?v=%d&encoding=%s", gatewayVersion, gatewayEncoding)
	connectHeader := http.Header{}

	connectHeader.Add("Authorization", fmt.Sprintf("%s %s", authTokenType, g.AuthToken))
	connectHeader.Add("User-Agent", userAgent)

	var resp *http.Response
	g.conn, resp, err = dialer.Dial(connectUrl, connectHeader)
	g.connMutex = new(sync.Mutex)
	log.Printf("Response: [%+v].", resp)

	if err != nil {
		return fmt.Errorf("failed to dial gateway: %v", err)
	}

	// First message should be a hello with heartbeat details.
	helloResp := new(GatewayPayload)
	err = g.conn.ReadJSON(helloResp)

	if err != nil {
		return fmt.Errorf("did not receive hello: %v", err)
	}
	log.Printf("First recv: [%+v].", helloResp)

	if helloResp.Opcode != OpcodeHello {
		return fmt.Errorf("not a hello opcode. Instead got message [%+v]", helloResp)
	}

	helloMessage := gatewayHelloResponse{}
	err = json.Unmarshal(helloResp.EventData, &helloMessage)

	if err != nil {
		return fmt.Errorf("unable to parse gateway hello response [%+v]", helloResp.EventData)
	}

	heartbeatInterval := time.Duration(helloMessage.HeartbeatInterval) * time.Millisecond

	if heartbeatInterval <= 0 {
		return fmt.Errorf("invalid heartbeat interval [%v] in hello", heartbeatInterval)
	}

	g.heartbeat = &discordHeartbeat{
		gateway:  g,
		interval: heartbeatInterval,
		getSequenceNum: func() *int {
			return g.sequenceNumber
		},
	}

	startHeartbeat(g.heartbeat)
	g.RegisterOpcodeListener(OpcodeHeartbeatACK, g.heartbeat.heartbeatAckRecv)
	g.RegisterOpcodeListener(OpcodeHeartbeat, g.heartbeat.heartbeatRecv)

	g.RegisterOpcodeListener(OpcodeDispatch, func(payload GatewayPayload) {
		g.sequenceNumber = payload.SequenceNumber
		listeners := g.eventListeners[payload.EventName]
		log.Printf("Found [%d] listeners for event [%v]", len(listeners), payload.EventName)
		for _, eventListener := range listeners {
			log.Print("Calling event listener.", eventListener)
			go eventListener(payload)
		}
	})

	go func() {
		for {
			payload := GatewayPayload{}
			err := g.conn.ReadJSON(&payload)

			if err != nil {
				log.Fatal("Failure reading message.", err)
			}

			log.Printf("Received payload with Opcode [%v], event name [%s], data [%s], and sequenceNum [%v].",
				payload.Opcode, payload.EventName, payload.EventData, payload.SequenceNumber)

			log.Printf("Found [%d] listeners for opcode", len(g.opcodeListeners[payload.Opcode]))
			for _, opcodeListener := range g.opcodeListeners[payload.Opcode] {
				go opcodeListener(payload)
			}
		}
	}()

	return
}

// TODO: verify this works.
type GatewayTrace struct {
	Trace []string `json:"_trace"`
}

// Reference: https://discordapp.com/developers/docs/topics/gateway#hello-hello-structure
type gatewayHelloResponse struct {
	GatewayTrace
	HeartbeatInterval int `json:"heartbeat_interval"`
}

// Reference: https://discordapp.com/developers/docs/topics/gateway#identify-identify-connection-properties
type gatewayConnectionProperties struct {
	Os      string `json:"$os"`
	Browser string `json:"$browser"`
	Device  string `json:"$device"`
}

// Reference: https://discordapp.com/developers/docs/topics/gateway#update-status-gateway-status-update-structure
type GatewayStatusUpdate struct {
	Since int `json:"since"`
	// Game *Activity
	Status string `json:"status"`
	Afk    bool   `json:"afk"`
}

// Reference: https://discordapp.com/developers/docs/topics/gateway#update-status-status-types
const (
	StatusOnline       = "online"
	StatusDoNotDisturb = "dnd"
	StatusIdle         = "idle"
	StatusInvisible    = "invisible"
	StatusOffline      = "offline"
)

// Identify should be called once heartbeat is established.
// Reference: https://discordapp.com/developers/docs/topics/gateway#identify-identify-structure
type gatewayIdentifyRequest struct {
	Token          string                      `json:"token"`
	Properties     gatewayConnectionProperties `json:"properties"`
	Compress       *bool                       `json:",omitempty"`
	LargeThreshold *int                        `json:"large_threshold"`
	Shard          *[]int                      `json:",omitempty"`
	Presence       *GatewayStatusUpdate        `json:",omitempty"`
}

// Ready is the event corresponding to the identify request.
// Reference: https://discordapp.com/developers/docs/topics/gateway#ready-ready-event-fields
type gatewayReadyResponse struct {
	GatewayTrace
	Version         int
	User            User
	PrivateChannels []Channel `json:"private_channels"`
	Guilds          []UnavailableGuild
	SessionId       string `json:"session_id"`
}

// How long to wait before giving up on identify event.
const identifyTimeoutSeconds = time.Duration(30) * time.Second

// Enable for packet compression over connection (used in identify request).
const enablePacketCompression = false

// Sends identify to server and returns user from the ready response.
func (g *DiscordGateway) Identify(initialStatus *GatewayStatusUpdate) (user User, err error) {

	compress := enablePacketCompression
	identifyRequest := gatewayIdentifyRequest{
		Token: g.AuthToken,
		Properties: gatewayConnectionProperties{
			Os:      runtime.GOOS,
			Browser: "none",
			Device:  "computer",
		},
		Compress: &compress,
		Presence: initialStatus,
	}

	var requestJsonBytes json.RawMessage
	requestJsonBytes, err = json.Marshal(&identifyRequest)

	if err != nil {
		return user, fmt.Errorf("failed to marshal identify request: %v", err)
	}

	messageReceieved := make(chan error)
	readyMessage := gatewayReadyResponse{}
	g.RegisterEventListener(EventReady, func(readyPayload GatewayPayload) {
		err = json.Unmarshal(readyPayload.EventData, &readyMessage)
		messageReceieved <- err
	})

	err = g.SendPayload(&GatewayPayload{
		Opcode:    OpcodeIdentify,
		EventData: requestJsonBytes,
	})

	if err != nil {
		return user, fmt.Errorf("failed to send identify request: %v", err)
	}

	select {
	case err = <-messageReceieved:
		if err != nil {
			return
		}

		g.sessionId = &readyMessage.SessionId
		user = readyMessage.User
	case <-time.After(identifyTimeoutSeconds):
		err = fmt.Errorf("Failed to get ready response for identify before timeout.")
	}

	return
}
