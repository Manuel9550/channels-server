/*
	playpool.go
	contains the code for the playpool
*/


package playpool

import (
	"fmt"
	"github.com/Manuel9550/server/pkg/player"
	"sync"
)

// represents a "pool" of players, who can indirectly compete with each other
type PlayPool struct {
	players []*player.Player // represents the current players in the play pool
	MaxPlayers int
	lock sync.Mutex // to ensure that only one player can be added to the channel at once
	PoolChannel chan player.Message // the channel where the players will write to, and the PlayerPool will listen to
}

// initialize the player pool with default values
func (c *PlayPool) Init() {

	c.players = make([]*player.Player,0)
	c.MaxPlayers = 10
	c.lock = sync.Mutex{}
	c.PoolChannel = make(chan player.Message)

	// start listening to the channel that the players will be sending information to
	go c.ListenToChannel()
}

// adding a player to the player pool
func (c *PlayPool) AddPlayer(newPlayer *player.Player) bool {
	// lock the pool
	c.lock.Lock()
	defer c.lock.Unlock()
	fmt.Println(len(c.players))
	if len(c.players) + 1 <= c.MaxPlayers {
		// add the player to the pool
		newPlayer.Id = len(c.players)
		newPlayer.PlayerChannel = c.PoolChannel
		c.players =append(c.players, newPlayer)

		// start listening to the players websocket for info
		go newPlayer.ReadSocket()

		// since the player has been added, broadcast a 'new player message' to the other players


		playerJoinedMessage := player.Message{MessageType:player.Connected,PlayerId:newPlayer.Id}
		c.PoolChannel <- playerJoinedMessage
		return true
	} else {
		return false
	}
}

// broadcasting a message to all players in the pool
func (c *PlayPool) BroadCastToPlayers(message player.Message) {
	// lock the pool
	c.lock.Lock()
	defer c.lock.Unlock()


	// iterate through all the player messages
	for _, currentPlayer := range c.players {
		// send this message to the current player, as long as it makes sense for them to receive it
		switch message.MessageType {
		case player.Attack:
			if currentPlayer.Id != message.PlayerId {
				currentPlayer.WriteSocket(message)
			}
		case player.Connected:
			if len(c.players) > 1 {
				currentPlayer.WriteSocket(message)
			} else {
				// break out of the loop
				break
			}
		case player.Disconnected:
			currentPlayer.WriteSocket(message)
		}

	}

}

// listens to the pool channel for incoming messages
func (c *PlayPool) ListenToChannel(){

	// we are waiting until there is a message to read from
	for {
		message := <- c.PoolChannel
		// the only message we don't care about sending to other players are if a player scored a point
		if message.MessageType != player.Score {

			// if the player disconnected, delete their spot in the PlayPool
			if message.MessageType == player.Disconnected{
				// lock the Player pool slice
				c.lock.Lock()

				// remove the player that has disconnected

				// temp slice to store the head of the new array

				if len(c.players) > 1 {
					newPlayerSlice := append(c.players[:message.PlayerId], c.players[message.PlayerId+1:]...)

					// set each players ID in the pool to be their original id
					for i, player := range newPlayerSlice {
						player.Id = i + 1
					}

					// the new array is ready
					c.players = newPlayerSlice

				} else {
					c.players = make([]*player.Player,0)
				}


				c.lock.Unlock()
			}

			c.BroadCastToPlayers(message)
		} else {
			// increase the players score by 1
			for _, player := range c.players {
				if player.Id == message.PlayerId {
					player.Score++
				}
			}
		}
	}
}

