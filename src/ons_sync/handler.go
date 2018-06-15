package main

import (
	"sync"
	"log"
	"strings"
	"net/url"
	"crypto/sha512"
	"encoding/hex"
	"encoding/json"
	"encoding/base64"
	"protobuf/ons_pb2"
	"github.com/golang/protobuf/proto"
	"github.com/gorilla/websocket"
)

const (
	maxMessageSize = 2048
)
var familyname string = "ons"
var namespace = Hexdigest(familyname)[:6]
var g_verbose bool = false

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
	rcv_exited chan bool
	block_id chan string
	wg *sync.WaitGroup
	conn *websocket.Conn
}

func NewONSEventHandler(addr string, path string, verbose bool) (*ONSEventHandler, error) {

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
		rcv_exited: make(chan bool),
		wg: &sync.WaitGroup{},
		conn: conn,
	}
	onsEvHandler.AddWaitGroup(1)
	onsEvHandler.initialized = true
	g_verbose = verbose
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

	defer h.conn.Close()

	if h.subscirbed == true {
		h.Subscribe(false)
	}
	//waiting needed?

	//h.exit_rcv <- true
	h.exit_sub <- true
	log.Println("Terminate : called")
	if waiting == true {
		h.Wait()
	}
}

func (h *ONSEventHandler) AddWaitGroup(wait_group_count int) {
	h.wg.Add(wait_group_count)
}

func (h *ONSEventHandler) Wait() {
	h.wg.Wait()
}

func (h *ONSEventHandler) Subscribe(subscribing bool) {
	h.subscribing <- subscribing
	//h.subscirbed = subscribing
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

	log.Printf("Called subscribe : %v", subscribing)

	return err
}

func (h *ONSEventHandler) getBlockDelteas(block_id string) error {
	data, _:= json.Marshal(&getBlockDeltasMessage{
			Action: "get_block_deltas",
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
	defer func() {
		h.wg.Done()
		log.Println("runSubscriber : Exit")
	}()
	for {
		select {
		case subscribing := <- h.subscribing:
			if h.subscirbed != subscribing {
				h.subscirbed = subscribing
				h.subscribe(subscribing)
			}
			log.Printf("runSubscriber : called subscribing : %v", subscribing)
		case block_id := <- h.block_id:
			h.getBlockDelteas(block_id)
		case _ = <- h.exit_sub:
			log.Println("runSubscriber : called exit")
			return
		}
	}
}

type ONSEvent struct {
	BlockNum float64 `json:"block_num,string"`
	BlockId string `json:"block_id"`
	PreviousBlockId string `json:"previous_block_id"`
	StateChanges []map[string]string `json:"state_changes"`
	Type string `json:"type"`
}

type ONSGS1CodeEvent struct {
	ons_pb2.GS1CodeData
	Address string `json:"address"`
	BlockNum float64 `json:"block_num"`
}

type ONSServiceTypeEvent struct {
	ons_pb2.ServiceType
	BlockNum float64 `json:"block_num"`
}

var g_current_latest_ons_event *ONSEvent = nil

func getCurrentLatestONSEvent(onsEvent *ONSEvent) *ONSEvent {
	if g_current_latest_ons_event == nil{
		g_current_latest_ons_event = onsEvent
		err := DBUpdateLatestUpdatedBlockInfo(onsEvent.BlockNum, onsEvent.BlockId, onsEvent.PreviousBlockId)
		if err != nil {
			log.Printf("getCurrentLatestONSEvent : Failed to update latest block info : ", err)
		}
		return g_current_latest_ons_event
	}

	if g_current_latest_ons_event.BlockNum < onsEvent.BlockNum {
		g_current_latest_ons_event = onsEvent
		err := DBUpdateLatestUpdatedBlockInfo(onsEvent.BlockNum, onsEvent.BlockId, onsEvent.PreviousBlockId)
		if err != nil {
			log.Printf("getCurrentLatestONSEvent : Failed to update latest block info : ", err)
		}
	}

	return g_current_latest_ons_event
}

func UpdateOnsEvent(h *ONSEventHandler, onsEvent *ONSEvent, verbose bool) {
	if onsEvent == nil {
		return
	}

	//currentLatestONSEvent := getCurrentLatestONSEvent(onsEvent)
	_ = getCurrentLatestONSEvent(onsEvent)

	latest_updated_block_num,  latest_updated_block_id := DBGetLatestUpdatedBlock()
	if verbose == true {
		log.Printf("latest block num : %v", latest_updated_block_num)
	}

	if onsEvent.PreviousBlockId != latest_updated_block_id {
		log.Printf("Call previous block info\n - current block id : %s\n - previous block id : ", onsEvent.BlockId, onsEvent.PreviousBlockId)
		h.GetBlockDeltas(onsEvent.PreviousBlockId)
	}else{
		//no more delta.
		DBGetLatestUpdatedBlockInfo(verbose)
	}

	for _, state := range onsEvent.StateChanges {
		event_type, ok := state["type"]

		if ok == false {
			log.Printf("event: NONE, block num : %v, block id : %v\n", onsEvent.BlockNum, onsEvent.BlockId)
			continue
		}

		if event_type == "DELETE" {
			log.Printf("event: DELETE, block num : %v, block id : %v\n", onsEvent.BlockNum, onsEvent.BlockId)
			err := DBDeleteAddress(state["address"])
			if err != nil {
				log.Printf("Fail to DBDeleteGS1Code : %v\n", err)
			}
			continue
		}

		log.Printf("event: SET, block num : %v, block id : %v\n", onsEvent.BlockNum, onsEvent.BlockId)

		state_value, err := base64.StdEncoding.DecodeString(state["value"])
		if err != nil {
			log.Printf("Fail to base64 decoding in UpdateOnsEvent : %v\n", err)
		}else {

			table_idx := GetTableIdxByAddress(state["address"])

			if table_idx == GS1_CODE_TABLE {
				log.Printf("Update gs1 code\n")
				gs1_code_event := &ONSGS1CodeEvent{}
				gs1_code_event.Address = state["address"]
				gs1_code_event.BlockNum = onsEvent.BlockNum
				err = proto.Unmarshal(state_value, gs1_code_event)
				if verbose == true {
					log.Printf("unmarshaled state value = %v\n", gs1_code_event)
				}
				DBUpdateOrInsert(GS1_CODE_TABLE, gs1_code_event.Gs1Code, gs1_code_event.BlockNum, gs1_code_event)
			}else if table_idx == SERVICE_TYPE_TABLE {
				log.Printf("Update service type\n")
				service_type_event := &ONSServiceTypeEvent{}
				service_type_event.BlockNum = onsEvent.BlockNum
				err = proto.Unmarshal(state_value, service_type_event)
				if err != nil {
					log.Printf("Fail to unmarshal proto buffer binary data in UpdateOnsEvent : %v\n", err)
					return
				}
				if verbose == true {
					log.Printf("unmarshaled state value = %v\n", service_type_event)
				}
				DBUpdateOrInsert(SERVICE_TYPE_TABLE, service_type_event.Address, service_type_event.BlockNum, service_type_event)
			}
		}
	}

}

func (h *ONSEventHandler) runReceiveEvents() {
	defer func() {
		//h.conn.Close()
		//h.wg.Done()
		//h.rcv_exited <- true
		log.Println("runReceiveEvents : Exit")
	}()
	for {
		_, message, err := h.conn.ReadMessage()
		if err != nil {
			log.Printf("Failed to read from websocket: %v", err)
			break
		}else{
			go func(h *ONSEventHandler) {
				//log.Printf("message type : %v", msg_type)
				//unmarshaling is needed...
				message = append([]byte{'['}, append(message, []byte{']'}...)...)
				//log.Printf("message : %v", string(message))
				var onsEvent []ONSEvent
				err = json.Unmarshal(message, &onsEvent)
				if err != nil {
					log.Printf("marshaling error : %#v", err)
				}
				//log.Printf("json : %#v", onsEvent[0])
				UpdateOnsEvent(h, &onsEvent[0], g_verbose)
			}(h)
		}
	}
}