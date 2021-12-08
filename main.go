package main

import (
	"encoding/csv"
	"fmt"
	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"
	"log"
	"net/http"
	"time"
)

const port = "8080"

var users = make(map[string]*websocket.Conn)
var botCh = make(chan Message, 1)

func main() {
	// listening command event from chat
	go func(ch chan Message) {
		startStockChatBot(ch)
	}(botCh)
	defer close(botCh)
	router := mux.NewRouter()
	upgrader := websocket.Upgrader{
		HandshakeTimeout: 2 * time.Second,
		ReadBufferSize:   1024,
		WriteBufferSize:  1024,
		CheckOrigin: func(r *http.Request) bool {
			//todo better check origin for prod
			return true
		},
	}
	router.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		handleWS(w, r, upgrader)
	})

	router.PathPrefix("/").Handler(http.FileServer(http.Dir("./static")))

	addr := "127.0.0.1:" + port
	fmt.Printf("starting server http://%s\n", addr)

	srv := &http.Server{
		Handler:      router,
		Addr:         addr,
		WriteTimeout: 15 * time.Second,
		ReadTimeout:  15 * time.Second,
	}

	log.Fatal(srv.ListenAndServe())

}

func handleWS(w http.ResponseWriter, r *http.Request, upgrader websocket.Upgrader) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		// here you can check possible errors and handle them differently , ex: check for headers! or origin!
		log.Println("Error upgrading connection, err:", err)
		return
	}
	fmt.Println("new connection ........remoteAddress: ", conn.RemoteAddr())

	var username string
	defer conn.Close()

	for {

		var message Message
		err = conn.ReadJSON(&message)
		if err != nil {
			log.Printf("Error reading message, disconnecting user: %s err: %v\n", username, err)
			if username != "" {
				//delete the user from db
				delete(users, username)

				notifyUserLeft(Message{
					Content:     nil,
					Username:    username,
					MessageType: SomeoneLeft,
					Date:        time.Now(),
				})
			}
			return
		}
		username = message.Username

		switch message.MessageType {
		// if someone is sending command, we notify the chatBot
		case Command:
			botCh <- message
			break

		case Join:
			// we send two events,
			//- joined - for setting up the profile and cookies/localStorage on the front, and
			// - someoneJoined - to update the online user panel.
			fmt.Println("joining.....", message.Username)
			if _, ok := users[message.Username]; ok {
				// handle already registered
				message.MessageType = AlreadyExists
				message.Content = "username already exists"
				conn.WriteJSON(message)
				//do not end the connection (loop), just let him try to join again with another name.
				break
			}
			users[message.Username] = conn

			message.MessageType = Joined
			err = conn.WriteJSON(message)
			if err != nil {
				log.Println("Error writing message, err: ", err)
				return
			}
			// we only send the newly joined users to update online-user-panel
			notifySomeOneHasJoined(message.Username)
			break
			// after the user joined, the front will ask for online-users, and we send the whole list.
		case OnlineUsers:
			msg := Message{
				MessageType: OnlineUsers,
				Content:     getUserList(),
			}
			conn := users[message.Username]
			conn.WriteJSON(msg)
			fmt.Printf("sending online users to %s, %v\n", message.Username, getUserList())
			break

		case Leave:
			if _, ok := users[message.Username]; ok {
				notifyUserLeft(message)
			}
			break

		case Chat:

			log.Println("received message: ", message)
			broadcast(message)
			break
		default:
			fmt.Println("MessageType invalid, msg: ", message)
			break

		}

	}

}

func notifyUserLeft(message Message) {
	message.MessageType = SomeoneLeft
	broadcast(message)
}

func broadcast(message Message) {
	for user, conn := range users {
		fmt.Println("notifying user: ", user)
		conn.WriteJSON(message)
	}

}

func getUserList() []string {
	var userList []string
	for username, _ := range users {
		userList = append(userList, username)
	}
	return userList
}

func notifySomeOneHasJoined(newUser string) {
	msg := Message{
		MessageType: SomeoneJoin,
		Content:     newUser,
	}
	for usr, conn := range users {
		if usr != newUser {
			conn.WriteJSON(msg)
		}
	}
}

type Message struct {
	Content     interface{}
	Username    string
	MessageType MsgType
	Date        time.Time
}
type MsgType string

const (
	Join          MsgType = "JOIN"
	Joined        MsgType = "JOINED"
	Leave         MsgType = "LEAVE"
	SomeoneLeft   MsgType = "SOMEONE_LEFT"
	Chat          MsgType = "CHAT"
	OnlineUsers   MsgType = "ONLINE_USERS"
	AlreadyExists         = "ALREADY_EXISTS"
	SomeoneJoin           = "SOMEONE_JOIN"
	Command               = "COMMAND"
)

// example command send from front is '/stock=aapl.us'
func startStockChatBot(ch chan Message) {
	select {
	case msg := <-ch:
		fmt.Println("[BOT], receive MSG: ", msg)
		result := msg.Content.([]interface{})
		fmt.Println("result: ", result)
		response, err := http.Get(fmt.Sprintf("https://stooq.com/q/l/?s=%s&f=sd2t2ohlcv&h&e=csv", result[1]))
		if err != nil {
			// ignoring errors
			fmt.Println("error on GET:", err)
			return
		}
		defer response.Body.Close()

		reader := csv.NewReader(response.Body)
		all, err := reader.ReadAll()
		if err != nil {
			fmt.Println("error on reading body, err:", err)
			return
		}
		if len(all) >= 2 {
			stockName := all[1][0]
			stockPrice := all[1][3]
			reply := fmt.Sprintf("%s quote is $%s per share", stockName, stockPrice)
			fmt.Println(reply)
			broadcast(Message{
				Content:     reply,
				Username:    "Stock-BOT",
				MessageType: Chat,
				Date:        time.Now(),
			})
		}
	}
}
