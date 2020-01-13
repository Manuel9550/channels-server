package main

import (
	"fmt"
	"github.com/gorilla/websocket"
	"net/http"
	"sync"
)


type PlayerMessage struct{
	content string
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
		_, msg, _ := c.conn.ReadMessage()
		println(string(msg))
	}
}

// the method responsible for writing to the players websocket
func (c *Player) writeSocket(message string) {
	c.conn.WriteJSON(message)
}

// represents a "pool" of players, who can indirectly compete with each other
type PlayPool struct {
	players []*Player // represents the current players in the play pool
	MaxPlayers int
	lock sync.Mutex // to ensure that only one player can be added to the channel at once
	messageReadyLock sync.Mutex // mutex for the channel message condition
	MessagereadyCond sync.Cond // condition for when a message is ready to be read from the channel
	PoolChannel chan PlayerMessage // the channel where the players will write to, and the PlayerPool will listen to
}

// initialize the player pool with default values
func (c *PlayPool) Init() {

	c.players = make([]*Player,0)
	c.MaxPlayers = 10
	c.lock = sync.Mutex{}
	c.messageReadyLock = sync.Mutex{}
	c.PoolChannel = make(chan PlayerMessage)

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
		return true
	} else {
		return false
	}
}

// broadcasting a message to all players
func (c *PlayPool) BroadCastToPlayers(message string, playerId int) {
	// lock the pool
	c.lock.Lock()
	defer c.lock.Unlock()

	// iterate through all the player messages
	for _, player := range c.players {
		// send this message to the player, as long as it didn't originate from this player
		if player.id != playerId {
			player.writeSocket(message)
		}
	}

}

// listens to the pool channel for incoming messages
func (c *PlayPool) ListenToChannel(){

}


var upgrader = websocket.Upgrader{}

func main(){

	// create our 'pool' of players that will be interacting with each other
	MainPlayPool := new(PlayPool)


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

			addPlayerSuccess := MainPlayPool.AddPlayer(currentPlayer)
			fmt.Println(addPlayerSuccess)
			if !addPlayerSuccess {
				// there wasn't enough space, close the connection
				conn.Close()
			} else {
				// start reading from the players connection
				currentPlayer.readSocket()
			}
		}(conn)

	})


	// serving the css and scripts directories in their "proper" location
	http.Handle("/css/", http.StripPrefix("/css/", http.FileServer(http.Dir("channels/css"))))
	http.Handle("/scripts/", http.StripPrefix("/scripts/", http.FileServer(http.Dir("channels/scripts"))))

	http.ListenAndServe(":3000", nil)
}