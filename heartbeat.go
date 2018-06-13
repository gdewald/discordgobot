package discordbot

import (
	"log"
	"time"

	"github.com/gorilla/websocket"
)

// Responsible for sending and receiving periodic discord heartbeats.
type discordHeartbeat struct {
	gateway       *DiscordGateway
	interval      time.Duration
	lastHeartbeat time.Time

	// Returns the current sequence number.
	getSequenceNum func() *int

	// Synchronization channels for heartbeats and acks
	heartbeatAck chan bool
	heartbeat    chan time.Time
}

// Called when a heartbeat ACK is received. Forwards the current sequence num to the ACK channel.
func (d *discordHeartbeat) heartbeatAckRecv(GatewayPayload) {
	log.Print("Received heartbeat ack")
	d.heartbeatAck <- true
}

// Called when a heartbeat is received. Updates the sequence num and responds with an ACK.
func (d *discordHeartbeat) heartbeatRecv(payload GatewayPayload) {
	log.Print("Received heartbeat with payload:", payload)
	d.gateway.SendPayload(&GatewayPayload{Opcode: OpcodeHeartbeatACK})
}

const closeTimeoutSeconds = time.Duration(5) * time.Second

// TODO: retry functionality.
func startHeartbeat(heartbeat *discordHeartbeat) {
	heartbeat.heartbeatAck = make(chan bool)
	heartbeatMessage := GatewayPayload{
		Opcode: OpcodeHeartbeat,
	}

	go func() {
		log.Printf("Starting heartbeat with interval: [%v].", heartbeat.interval)
		for {
			heartbeatMessage.SequenceNumber = heartbeat.getSequenceNum()

			log.Print("Sending heartbeat.")
			err := heartbeat.gateway.SendPayload(&heartbeatMessage)

			if err != nil {
				log.Fatal("Failed to send heartbeat: ", err)
			}

			lastHeartbeat := time.Now()

			select {
			case <-heartbeat.heartbeatAck:
				timeSinceLast := time.Now().Sub(lastHeartbeat)
				log.Printf("Sleeping after heartbeat ack.")
				time.Sleep(heartbeat.interval - timeSinceLast)
			case <-time.After(heartbeat.interval):
				err := heartbeat.gateway.SendControl(
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
