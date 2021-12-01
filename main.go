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

	defer conn.Close()

	for {

		var message Message
		err = conn.ReadJSON(&message)
		if err != nil {
			log.Println("Error reading message, err: ", err)
			return
		}

		switch message.MessageType {

		case Join:
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
			// update online user_list
			sendUserList()
			break

		case Leave:
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

func broadcast(message Message) {
	for _, conn := range users {
		conn.WriteJSON(message)
	}

}

func getUserList() []string {
	var userlist []string
	for k, _ := range users {
		userlist = append(userlist, k)
	}
	return userlist
}

func sendUserList() {
	for _, conn := range users {
		msg := Message{
			Content:     getUserList(),
			Username:    "",
			MessageType: UserList,
		}
		conn.WriteJSON(msg)
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
	Chat          MsgType = "CHAT"
	UserList      MsgType = "USER_LIST"
	AlreadyExists         = "ALREADY_EXISTS"
)
