/*
	main.go
	the main entry point for the application
	starts the Go server and starts listening for requests
*/


package main

import (
	"fmt"
	"github.com/Manuel9550/server/pkg/player"
	"github.com/gorilla/websocket"
	"net/http"
	"sync"
	"github.com/Manuel9550/server/pkg/playpool"
)



var upgrader = websocket.Upgrader{}

func CreatePlayerPool() *playpool.PlayPool {
	NewPlayPool := new(playpool.PlayPool)
	NewPlayPool.Init()

	return NewPlayPool
}


type PoolPair struct {
	Pool *playpool.PlayPool
	PoolLock sync.Mutex
}
func main(){

	// our "pool" of PlayPools
	PlayPools := make([]*playpool.PlayPool, 1)

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
			currentPlayer := &player.Player{
				Conn:  conn,
				Score: 0,
				Id: 0,
				PlayerChannel: nil,
			}

			PoolLock.Lock()
			finalResult := false
			// loop through every player pool, and check if there are any that can hold the new player
			for _, currentPool := range PlayPools{

				// lock this pool so others cannot add to it while we are using it
				finalResult = currentPool.AddPlayer(currentPlayer)

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
	http.Handle("/images/", http.StripPrefix("/images/", http.FileServer(http.Dir("channels/images"))))

	http.ListenAndServe(":3000", nil)
}