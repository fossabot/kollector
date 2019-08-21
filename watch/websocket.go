package watch

import (
	"log"
	"net/url"
	"time"

	"github.com/gorilla/websocket"
)

type ReqType int

const (
	PING    ReqType = 0
	MESSAGE ReqType = 1
	EXIT    ReqType = 2
)

type DataSocket struct {
	message string
	RType   ReqType
}

type WebSocketHandler struct {
	data chan DataSocket
}

func CreateWebSocketHandler() *WebSocketHandler {
	return &WebSocketHandler{data: make(chan DataSocket)}
}

func (wsh *WebSocketHandler) StartWebSokcetClient(urlWS string, path string, cluster_name string, customer_guid string) {

	u := url.URL{Scheme: "wss", Host: urlWS, Path: path, ForceQuery: true}
	q := u.Query()
	q.Add("customerGUID", customer_guid)
	q.Add("clusterName", cluster_name)
	u.RawQuery = q.Encode()
	log.Printf("connecting to %s", u.String())

	conn, _, err := websocket.DefaultDialer.Dial(u.String(), nil)
	if err != nil {
		log.Fatal("dial:", err)
	}
	//defer conn.Close()

	go func(conn *websocket.Conn) {
		for {
			data := <-wsh.data
			switch data.RType {
			case MESSAGE:
				log.Println("Sending: message.")
				err := conn.WriteMessage(websocket.TextMessage, []byte(data.message))
				if err != nil {
					log.Println("WriteMessage to websocket:", err)
					i := 0
					for i < 3 {
						log.Printf("reconnect try number %d of 3\n", i+1)
						conn, _, err = websocket.DefaultDialer.Dial(u.String(), nil)
						if err != nil {
							log.Fatal("dial:", err)
						}
						err := conn.WriteMessage(websocket.TextMessage, []byte(data.message))
						if err != nil {
							log.Println("WriteMessage to websocket:", err)
						} else {
							log.Println("resending: message.")
							break
						}
						i++
					}
					if i < 3 {
						continue
					} else {
						log.Printf("reconnection failed fo 3 times, exiting\n")
					}
				}
			case EXIT:
				log.Printf("web socket client got exit with message: %s\n", data.message)
				return
			}
		}
	}(conn)

	go func(conn *websocket.Conn) {
		for {
			log.Println("Sending: ping.")
			err = conn.WriteMessage(websocket.PingMessage, []byte{})
			if err != nil {
				log.Println("Write Error: ", err)
				break
			}

			msgType, _, err := conn.ReadMessage()
			if err != nil {
				log.Println("WebSocket closed.")
				data := DataSocket{message: "WebSocket closed", RType: EXIT}
				wsh.data <- data
				i := 0
				for i < 3 {
					log.Printf("reconnect try number %d of 3", i+1)
					conn, _, err = websocket.DefaultDialer.Dial(u.String(), nil)
					if err != nil {
						log.Fatal("dial:", err)
					}
					i++
				}
				if i < 3 {
					continue
				} else {
					log.Printf("reconnection failed fo 3 times, exiting\n")
				}
				return
			}
			// We don't recognize any message that is not "pong".
			if msgType != websocket.PongMessage {
				log.Println("Unrecognized message received.")
				time.Sleep(40 * time.Second)
				continue
			} else {
				log.Println("Received: pong.")
			}
			time.Sleep(40 * time.Second)
		}
	}(conn)
}

//SendMessageToWebSocket -
func (wh *WatchHandler) SendMessageToWebSocket(jsonData []byte) {
	data := DataSocket{message: string(jsonData), RType: MESSAGE}

	wh.WebSocketHandle.data <- data
}
