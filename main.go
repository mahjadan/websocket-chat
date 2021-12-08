package main

import (
	"embed"
	"fmt"
	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"
	"log"
	"net/http"
	"time"
)

//go:embed static
var fileStatic embed.FS

const port = "8080"

var users = make(map[string]*websocket.Conn)

func main() {
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
		case Ping:
			fmt.Println("Ping - Pong")
			conn.WriteJSON(Message{
				MessageType: "PONG",
			})
			break

		case Join:
			// we send two events,
			//- joined - for setting up the profile and cookies/localStorage on the front, and
			// - Online_users - to update the online user panel.
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
			// how to identify its response of registration? its the only msg that server send to client or NOT?
			message.MessageType = Joined

			err = conn.WriteJSON(message)
			if err != nil {
				log.Println("Error writing message, err: ", err)
				return
			}
			// you can not just send the newly joined user, you have to send all online users, because
			// if only send newly joined user, the new user will not be able to see the users who have joined before him.
			// this sends all the online users ( front has to filter to not include himself)
			//sendOnlineUserList()// THIS step should be requested by the user after joining
			//todo notify others that someone has joined
			notifySomeOneHasJoined(message.Username)
			break
		case OnlineUsers:
			msg := Message{
				MessageType: OnlineUsers,
				Content:     getUserList(),
			}
			conn := users[message.Username]
			conn.WriteJSON(msg)
			fmt.Printf("sending online users to %s, %v\n", message.Username, getUserList())
			//sendOnlineUserList()
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

//func handleHome(w http.ResponseWriter, r *http.Request) {
//	if r.URL.Path != "/" {
//		w.WriteHeader(http.StatusNotFound)
//		w.Write([]byte("NOT FOUND"))
//		return
//	}
//	t := template.Must(template.ParseFS(filesTempl, "template/template.html", "template/navbar.html", "template/home.html"))
//	t.ExecuteTemplate(w, "layout", map[string]string{"Page": "home"})
//}

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
	Ping                  = "PING"
)
