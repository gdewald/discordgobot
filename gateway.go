package discordbot

import (
	"fmt"
	"log"
	"net/http"
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
	listeners map[int][]GatewayMessageListener
	conn      *websocket.Conn
	heartbeat *discordHeartbeat
}

type GatewayMessageListener func(gatewayPayload)

func (g *DiscordGateway) RegisterOpcodeListener(opcode int, listener GatewayMessageListener) {
	if g.listeners == nil {
		g.listeners = make(map[int][]GatewayMessageListener)
	}
	g.listeners[opcode] = append(g.listeners[opcode], listener)
}

func (g *DiscordGateway) Connect() (err error) {
	dialer := websocket.Dialer{}

	connectUrl := g.GatewayInfo.Url + fmt.Sprintf("/?v=%d&encoding=%s", gatewayVersion, gatewayEncoding)
	connectHeader := http.Header{}

	connectHeader.Add("Authorization", fmt.Sprintf("%s %s", authTokenType, g.AuthToken))
	connectHeader.Add("User-Agent", userAgent)

	var resp *http.Response
	g.conn, resp, err = dialer.Dial(connectUrl, connectHeader)
	log.Printf("Response: [%+v].", resp)

	if err != nil {
		return fmt.Errorf("failed to dial gateway: %v", err)
	}

	// First message should be a hello with heartbeat details.
	helloResp := new(gatewayPayload)
	helloResp.EventData = new(gatewayHelloResponse)
	err = g.conn.ReadJSON(helloResp)

	if err != nil {
		return fmt.Errorf("did not receive hello: %v", err)
	}
	log.Printf("First recv: [%+v].", helloResp)

	if helloResp.Opcode != OpcodeHello {
		return fmt.Errorf("not a hello opcode. Instead got message [%+v]", helloResp)
	}

	helloMessage, ok := helloResp.EventData.(*gatewayHelloResponse)

	if !ok {
		return fmt.Errorf("unable to parse gateway hello response [%+v]", helloResp.EventData)
	}
	heartbeatInterval := time.Duration(helloMessage.HeartbeatInterval) * time.Millisecond

	if heartbeatInterval <= 0 {
		return fmt.Errorf("invalid heartbeat interval [%v] in hello", heartbeatInterval)
	}

	g.heartbeat = &discordHeartbeat{
		conn:     g.conn,
		interval: heartbeatInterval,
	}

	startHeartbeat(g.heartbeat)
	g.RegisterOpcodeListener(OpcodeHeartbeatACK, g.heartbeat.heartbeatAckRecv)
	g.RegisterOpcodeListener(OpcodeHeartbeat, g.heartbeat.heartbeatRecv)

	go func() {
		for {
			payload := gatewayPayload{}
			err := g.conn.ReadJSON(&payload)

			if err != nil {
				log.Fatal("Failure reading message.", err)
			}

			log.Printf("Received payload [%+v].", payload)

			for _, opcodeListener := range g.listeners[payload.Opcode] {
				go opcodeListener(payload)
			}
		}
	}()

	return
}

const closeTimeoutSeconds = time.Duration(5) * time.Second

// TODO: retry functionality
func startHeartbeat(heartbeat *discordHeartbeat) {
	heartbeat.heartbeatAck = make(chan *int)
	heartbeatMessage := gatewayPayload{
		Opcode: OpcodeHeartbeat,
	}

	go func() {
		log.Printf("Starting heartbeat with interval: [%v].", heartbeat.interval)
		for {
			heartbeatMessage.SequenceNumber = heartbeat.sequenceNum

			log.Print("Sending heartbeat.")
			err := heartbeat.conn.WriteJSON(heartbeatMessage)

			if err != nil {
				log.Fatal("Failed to send heartbeat: ", err)
			}

			lastHeartbeat := time.Now()

			select {
			case heartbeatMessage.SequenceNumber = <-heartbeat.heartbeatAck:
				log.Printf("Received ack with sequence number [%v].", heartbeatMessage.SequenceNumber)
				timeSinceLast := time.Now().Sub(lastHeartbeat)
				time.Sleep(heartbeat.interval - timeSinceLast)
			case <-time.After(heartbeat.interval):
				err := heartbeat.conn.WriteControl(
					websocket.CloseMessage,
					websocket.FormatCloseMessage(CloseSessionTimeout, ""),
					time.Now().Add(closeTimeoutSeconds),
				)

				if err != nil {
					log.Printf("gateway failed to close with error: %v", err)
				}

				log.Fatal("Heartbeat ack not received within time window.")
			}
		}
	}()
}

// https://discordapp.com/developers/docs/topics/gateway#payloads-gateway-payload-structure
type gatewayPayload struct {
	Opcode         int         `json:"op"`
	EventName      string      `json:"t,omitempty"`
	EventData      interface{} `json:"d"`
	SequenceNumber *int        `json:"s,omitempty"`
}

// https://discordapp.com/developers/docs/topics/gateway#hello
type gatewayHelloResponse struct {
	HeartbeatInterval int `json:"heartbeat_interval"`
	// _trace omitted
}

type discordHeartbeat struct {
	conn          *websocket.Conn
	interval      time.Duration
	heartbeatAck  chan *int
	heartbeat     chan time.Time
	lastHeartbeat time.Time
	sequenceNum   *int
}

// Called when a heartbeat ACK is received. Forwards the current sequence num to the ACK channel.
func (d *discordHeartbeat) heartbeatAckRecv(gatewayPayload) {
	log.Print("Received heartbeat ack")
	d.heartbeatAck <- d.sequenceNum
}

// Called when a heartbeat is received. Updates the sequence num and responds with an ACK.
func (d *discordHeartbeat) heartbeatRecv(payload gatewayPayload) {
	log.Print("Received heartbeat with payload:", payload)
	d.sequenceNum = payload.SequenceNumber
	d.conn.WriteJSON(gatewayPayload{Opcode: OpcodeHeartbeatACK})
}
