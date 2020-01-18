package main

import (
	"encoding/json"
	"fmt"
	"github.com/gorilla/websocket"
	"net/http"
	"sync"
)

// the message types that we can send or receive from the players
type MessageType int

const (
	Connected MessageType = 0 // Player has connected to the game
	Disconnected   MessageType = 1 // player has disconnected from the game
	Attack MessageType = 2 // A player has sent an "attack" (spawns another entity) to all other players
	Score MessageType = 3 // updating a players score
)

type PlayerMessage struct{
	messageType MessageType
	playerId int
}

// keeps track of the player information
type Player struct {
	conn *websocket.Conn // the websocket connection for the player
	score int // the player score
	id int // the id of the player (inside the game pool)
	playerChannel chan PlayerMessage // where the player will write to

}

// the method responsible for reading from the players websocket
func (c *Player) readSocket() {
	for {

		// create the message that this player will send
		messageReceived := PlayerMessage{}
		messageType := Disconnected
		_, msg, err := c.conn.ReadMessage()
		if err != nil {
			// close the connection and remove the player from the queue
			_ = c.conn.Close()
			messageType = Disconnected
		} else {
			// unmarshal the json
			marshalError := json.Unmarshal(msg, &messageReceived)
			if marshalError != nil {
				// improper message found, close the connection
				_ = c.conn.Close()
				messageType = Disconnected
			} else {
				// determine what type of message it is
				messageType = messageReceived.messageType
			}
		}

		messageReceived.messageType = messageType
		messageReceived.playerId = c.id
		c.playerChannel <- messageReceived


	}
}

// the method responsible for writing to the players websocket
func (c *Player) writeSocket(message PlayerMessage) {
	c.conn.WriteJSON(message)
}

// represents a "pool" of players, who can indirectly compete with each other
type PlayPool struct {
	players []*Player // represents the current players in the play pool
	MaxPlayers int
	lock sync.Mutex // to ensure that only one player can be added to the channel at once
	PoolChannel chan PlayerMessage // the channel where the players will write to, and the PlayerPool will listen to
}

// initialize the player pool with default values
func (c *PlayPool) Init() {

	c.players = make([]*Player,0)
	c.MaxPlayers = 10
	c.lock = sync.Mutex{}
	c.PoolChannel = make(chan PlayerMessage)

	// start listening to the channel that the players will be sending information to
	c.ListenToChannel()

}

// adding a player to the player pool
func (c *PlayPool) AddPlayer(player *Player) bool {
	// lock the pool
	c.lock.Lock()
	defer c.lock.Unlock()
	fmt.Println(len(c.players))
	if len(c.players) + 1 <= c.MaxPlayers {
		// add the player to the pool
		player.id = len(c.players)
		player.playerChannel = c.PoolChannel
		c.players =append(c.players, player)

		// start listening to the players websocket for info
		go player.readSocket()

		return true
	} else {
		return false
	}
}

// broadcasting a message to all players
func (c *PlayPool) BroadCastToPlayers(message PlayerMessage) {
	// lock the pool
	c.lock.Lock()
	defer c.lock.Unlock()

	// iterate through all the player messages
	for _, player := range c.players {
		// send this message to the current player, as long as it makes sense for them to receive it
		switch message.messageType {
		case Attack:
			if player.id != message.playerId {
				player.writeSocket(message)
			}
		case Connected:
			if len(c.players) > 1 {
				player.writeSocket(message)
			} else {
				// break out of the loop
				break
			}
		case Disconnected:
			player.writeSocket(message)
		}

	}

}

// listens to the pool channel for incoming messages
func (c *PlayPool) ListenToChannel(){

	// we are waiting until there is a message to read from
	for {
		message := <- c.PoolChannel
		// the only message we don't care about sending to other players are if a player scored a point
		if message.messageType != Score {
			c.BroadCastToPlayers(message)
		} else {
			// increase the players score by 1
			for _, player := range c.players {
				if player.id == message.playerId {
					player.score++
				}
			}
		}
	}
}


var upgrader = websocket.Upgrader{}

func CreatePlayerPool() *PlayPool {
	NewPlayPool := new(PlayPool)
	NewPlayPool.Init()

	return NewPlayPool
}


type PoolPair struct {
	Pool *PlayPool
	PoolLock sync.Mutex
}

/*
func AddNewPlayer(PoolSlice *[]PlayPool, player *Player) {
	finalResult := false

	// loop through every player pool, and check if there are any that can hold the new player
	for _, currentPool := range *PoolSlice {

		// lock this pool so others cannot add to it while we are using it
		currentPool.lock.Lock()
		finalResult = currentPool.AddPlayer(player)
		currentPool.lock.Unlock()

		if finalResult {
			// success, player was added
			break
		}
	}

	// if there was no room in any of the player pools, a new pool must be created
	newPool := CreatePlayerPool()

	newPool.AddPlayer(player)



}
*/


func main(){

	// our "pool" of PlayPools
	PlayPools := make([]*PlayPool, 1)

	// add our initial PlayPool to the PlayPool pool
	MainPlayPool := CreatePlayerPool()

	PlayPools[0] = MainPlayPool

	// the mutex for our main Pool object
	PoolLock := sync.Mutex{}

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w,r,"channels/channels.html")
	})

	// accept a new connection
	http.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		var conn, _ = upgrader.Upgrade(w,r,nil)
		go func(conn *websocket.Conn){
			fmt.Println("A user has connected")


			// create the player and add them to the pool
			currentPlayer := &Player{
				conn:  conn,
				score: 0,
				id: 0,
				playerChannel: nil,
			}

			PoolLock.Lock()
			finalResult := false
			// loop through every player pool, and check if there are any that can hold the new player
			for _, currentPool := range PlayPools{

				// lock this pool so others cannot add to it while we are using it
				currentPool.lock.Lock()
				finalResult = currentPool.AddPlayer(currentPlayer)
				currentPool.lock.Unlock()

				if finalResult {
					// success, player was added
					break
				}
			}

			if !finalResult {
				// a new pool must be created to hold the player
				newPool := CreatePlayerPool()

				newPool.AddPlayer(currentPlayer)
				PlayPools = append(PlayPools, newPool)
			}

			PoolLock.Unlock()

		}(conn)

	})


	// serving the css and scripts directories in their "proper" location
	http.Handle("/css/", http.StripPrefix("/css/", http.FileServer(http.Dir("channels/css"))))
	http.Handle("/scripts/", http.StripPrefix("/scripts/", http.FileServer(http.Dir("channels/scripts"))))

	http.ListenAndServe(":3000", nil)
}