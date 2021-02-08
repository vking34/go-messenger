package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/joho/godotenv"

	"github.com/gin-gonic/gin"

	socketio "github.com/googollee/go-socket.io"

	uuid "github.com/satori/go.uuid"
)

type ChatQuery struct {
	token  string
	userId string
}

type message struct {
	Id        string `json:"id"`
	RoomID    string `json:"room_id"`
	From      string `json:"from"`
	To        string `json:"to"`
	MsgType   string `json:"type"`
	Content   string `json:"content"`
	CreatedAt string `json:"created_at`
}

func init() {
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file")
	}
}

func main() {
	router := gin.New()

	server, _ := socketio.NewServer(nil)

	messengerNs := os.Getenv("MESSENGER_NS")

	server.OnConnect(messengerNs, func(s socketio.Conn) error {
		fmt.Println("socket id:", s.ID())
		fmt.Println("url:", s.URL().RawQuery)

		params := strings.Split(s.URL().RawQuery, "&")
		var chatQuery ChatQuery
		for _, param := range params {
			parts := strings.Split(param, "=")
			switch parts[0] {
			case "token":
				chatQuery.token = parts[1]
			case "user_id":
				chatQuery.userId = parts[1]
			}
		}

		fmt.Println("query:", chatQuery)
		s.SetContext("")
		s.Join(chatQuery.userId)
		return nil
	})

	server.OnEvent(messengerNs, "new_message", func(s socketio.Conn, msgStr string) {
		fmt.Println("msg string:", msgStr)

		msg := message{}
		err := json.Unmarshal([]byte(msgStr), &msg)
		if err != nil {
			fmt.Println("err:", err.Error())
		}

		msg.Id = uuid.NewV4().String()
		msg.CreatedAt = time.Now().Format(time.RFC3339)
		fmt.Println("msg:", msg)
		server.BroadcastToRoom(messengerNs, msg.From, "new_message", msg)
		server.BroadcastToRoom(messengerNs, msg.To, "new_message", msg)
	})

	// server.OnConnect("/", func(s socketio.Conn) error {
	// 	s.SetContext("")
	// 	fmt.Println("connected:", s.ID())
	// 	return nil
	// })

	// server.OnEvent("/", "notice", func(s socketio.Conn, msg string) {
	// 	fmt.Println("notice:", msg)
	// 	s.Emit("reply", "have "+msg)
	// })

	// server.OnEvent("/chat", "msg", func(s socketio.Conn, msg string) string {
	// 	s.SetContext(msg)
	// 	return "recv " + msg
	// })

	// server.OnEvent("/", "bye", func(s socketio.Conn) string {
	// 	last := s.Context().(string)
	// 	s.Emit("bye", last)
	// 	s.Close()
	// 	return last
	// })

	server.OnError("/", func(s socketio.Conn, e error) {
		fmt.Println("meet error:", e)
	})

	server.OnDisconnect("/", func(s socketio.Conn, reason string) {
		fmt.Println("closed", reason)

	})

	go server.Serve()
	defer server.Close()

	// register
	socketPath := os.Getenv("SOCKET_PATH") + "/*any"
	router.GET(socketPath, gin.WrapH(server))
	router.POST(socketPath, gin.WrapH(server))
	router.StaticFS("/public", http.Dir("./asset"))

	router.Run()
}
