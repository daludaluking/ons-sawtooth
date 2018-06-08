package main

import (
	"sync"
	"log"
	"strings"
	"net/url"
	"crypto/sha512"
	"encoding/hex"
	"encoding/json"

	"github.com/gorilla/websocket"
)

const (
	maxMessageSize = 2048
)
var familyname string = "ons"
var namespace = Hexdigest(familyname)[:6]

func Hexdigest(str string) string {
	hash := sha512.New()
	hash.Write([]byte(str))
	hashBytes := hash.Sum(nil)
	return strings.ToLower(hex.EncodeToString(hashBytes))
}
/*
'action': 'subscribe',
'address_prefixes': ['5b7349']

'action': 'get_block_deltas',
'block_id': 'd4b46c1c...',
'address_prefixes': ['5b7349']
*/
type subscribingMessage struct {
    Action string    `json:"action"`
    Address_prefixes  []string `json:"address_prefixes"`
}

type unsubscribingMessage struct {
	Action string    `json:"action"`
}

type getBlockDeltasMessage struct {
	Action string    `json:"action"`
	BlockId string   `json:"block_id"`
    Address_prefixes  []string `json:"address_prefixes"`
}

// Hub maintains the set of active clients and broadcasts messages to the
// clients.
type ONSEventHandler struct {
	initialized bool
	subscirbed bool
	subscribing chan bool
	exit_sub chan bool
	exit_rcv chan bool
	block_id chan string
	wg *sync.WaitGroup
	conn *websocket.Conn
}

func NewONSEventHandler(addr string, path string) (*ONSEventHandler, error) {

	u := url.URL{Scheme: "ws", Host: addr, Path: path}
	log.Printf("connecting to %s", u.String())

	conn, _, err := websocket.DefaultDialer.Dial(u.String(), nil)
	if err != nil {
		log.Printf("Websocket dial error: %v", err)
		return nil, err
	}

	onsEvHandler := &ONSEventHandler{
		subscirbed: false,
		subscribing:  make(chan bool),
		block_id: make(chan string),
		exit_sub: make(chan bool),
		exit_rcv: make(chan bool),
		wg: &sync.WaitGroup{},
		conn: conn,
	}
	onsEvHandler.AddWaitGroup(2)
	onsEvHandler.initialized = true
	return onsEvHandler, nil
}

func (h *ONSEventHandler) Run() bool {
	if h.initialized != true {
		log.Printf("ONSEventHandler isn't intialized")
		return false
	}
	go h.runSubscriber()
	go h.runReceiveEvents()
	return true
}

func (h *ONSEventHandler) Terminate(waiting bool) {
	if h.subscirbed == true {
		h.Subscribe(false)
	}
	//waiting needed?

	h.exit_rcv <- true
	h.exit_sub <- true

	if waiting == true {
		h.Wait()
	}

	h.conn.Close()
}

func (h *ONSEventHandler) AddWaitGroup(wait_group_count int) {
	h.wg.Add(wait_group_count)
}

func (h *ONSEventHandler) Wait() {
	h.wg.Wait()
}

func (h *ONSEventHandler) Subscribe(subscribing bool) {
	h.subscribing <- subscribing
}

func (h *ONSEventHandler) GetBlockDeltas(block_id string) {
	h.block_id <- block_id
}

func (h *ONSEventHandler) subscribe(subscribing bool) error {
	var data []byte

	if subscribing == true {
		data, _ = json.Marshal(&subscribingMessage{
			Action: "subscribe",
			Address_prefixes: []string{namespace},
		})
	}else{
		data, _ = json.Marshal(&unsubscribingMessage{
			Action: "unsubscribe",
		})
	}

	err := h.conn.WriteMessage(websocket.TextMessage, data)

	if err != nil {
		log.Printf("Failed to sendSubscribeMessage : %v", err)
	}

	return err
}

func (h *ONSEventHandler) getBlockDelteas(block_id string) error {
	data, _:= json.Marshal(&getBlockDeltasMessage{
			Action: "subscribe",
			BlockId: block_id,
			Address_prefixes: []string{namespace},
		})

	err := h.conn.WriteMessage(websocket.TextMessage, data)

	if err != nil {
		log.Printf("Failed to sendSubscribeMessage : %v", err)
	}

	return err
}

func (h *ONSEventHandler) runSubscriber() {
	defer h.wg.Done()
	for {
		select {
		case subscribing := <- h.subscribing:
			if h.subscirbed != subscribing {
				h.subscirbed = subscribing
				h.subscribe(subscribing)
			}
		case block_id := <- h.block_id:
			h.getBlockDelteas(block_id)
		case _ = <- h.exit_sub:
			return
		}
	}
}

func (h *ONSEventHandler) runReceiveEvents() {
	defer h.wg.Done()
	defer h.conn.Close()
	for {
		msg_type, message, err := h.conn.ReadMessage()
		if err != nil {
			log.Printf("Failed to read from websocket: %v", err)
			break
		}else{
			go func() {
				log.Printf("message type : %v", msg_type)
				//unmarshaling is needed...
				log.Printf("message : %v", string(message))
			}()
		}
	}
}