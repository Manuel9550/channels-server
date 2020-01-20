/*
	player.go
	contains the player code, which represents a client that has connected to the server
*/


package player

import (
	"encoding/json"
	"github.com/gorilla/websocket"
)
// the message types that we can send or receive from the players
type MessageType int

const (
	Connected MessageType = 0 // Player has connected to the game
	Disconnected   MessageType = 1 // player has disconnected from the game
	Attack MessageType = 2 // A player has sent an "attack" (spawns another entity) to all other players
	Score MessageType = 3 // updating a players score
)

type Message struct{
	MessageType MessageType
	PlayerId int
}

type ClientNotification struct {
	MessageType MessageType
}

// keeps track of the player information
type Player struct {
	Conn *websocket.Conn // the websocket connection for the player
	Score int // the player score
	Id int // the id of the player (inside the game pool)
	PlayerChannel chan Message // where the player will write to

}

// the method responsible for reading from the players websocket
func (c *Player) ReadSocket() {
	for {

		// create the message that this player will send to the channel
		messageReceived := ClientNotification{}
		messageType := Disconnected
		_, msg, err := c.Conn.ReadMessage()
		if err != nil {
			// close the connection and remove the player from the queue
			_ = c.Conn.Close()
			messageType = Disconnected
		} else {
			// unmarshal the json
			marshalError := json.Unmarshal(msg, &messageReceived)
			if marshalError != nil {
				// improper message found, close the connection
				_ = c.Conn.Close()
				messageType = Disconnected
			} else {
				// determine what type of message it is
				messageType = messageReceived.MessageType
			}
		}

		PlayerMessageReceived := Message{
			MessageType: messageType,
			PlayerId:    c.Id,
		}
		c.PlayerChannel <- PlayerMessageReceived

		// if the connection is closing, we need to break out of the loop
		if messageType == Disconnected {
			break
		}
	}
}

// the method responsible for writing to the players websocket
func (c *Player) WriteSocket(message Message) {
	c.Conn.WriteJSON(message)
}