package binancestream

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net"
	"sync"

	"github.com/gorilla/websocket"
)

var (
	debugLog *log.Logger = log.New(io.Discard, "", log.LstdFlags)

	ErrWSDisconnected = fmt.Errorf("websocket disconnected")
	ErrClosed         = fmt.Errorf("websocket manager is closed")
)

func EnableDebugLogger() {
	debugLog = log.Default()
}

type wsRequest struct {
	ID     uint     `json:"id"`
	Method string   `json:"method"`
	Params []string `json:"params"`
}

type Response struct {
	ID            *uint           `json:"id"`
	Result        json.RawMessage `json:"result"`
	ResponseError *struct {
		Code *uint   `json:"code"`
		Msg  *string `json:"msg"`
	} `json:"error"`
}

func (e *Response) Error() error {
	if e.ResponseError == nil {
		return nil
	}
	return fmt.Errorf("%d: %s", *e.ResponseError.Code, *e.ResponseError.Msg)
}

type binanceWs struct {
	mu sync.RWMutex
	ws *websocket.Conn

	requestIDCounter uint

	messages chan []byte
	response map[uint]chan Response

	closeCh chan struct{}
}

func newBinanceWs() *binanceWs {
	return &binanceWs{
		messages: make(chan []byte, 100),
		response: make(map[uint]chan Response),
		closeCh:  make(chan struct{}),
	}
}

func (m *binanceWs) connect() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.closed() {
		return ErrClosed
	}
	var err error
	m.ws, _, err = websocket.DefaultDialer.Dial("wss://stream.binance.com:9443/stream", nil)
	if err != nil {
		return fmt.Errorf("connect error: %w", err)
	}
	return nil
}

func (m *binanceWs) pollMessages() error {
	for {
		if m.closed() {
			return ErrClosed
		}
		mtype, message, err := m.ws.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err) {
				return ErrWSDisconnected
			}
			if v, ok := err.(*net.OpError); ok && v.Err.Error() == "use of closed network connection" {
				return ErrWSDisconnected
			}
			log.Printf("read error: %v", err)
			continue
		}
		debugLog.Println("message:", string(message)) // TODO: remove
		if mtype != websocket.TextMessage {
			log.Printf("unexpected message type mtype: %d, message: %s", mtype, message)
			continue
		}

		// Handle the message
		var r Response
		err = json.Unmarshal(message, &r)
		if err != nil {
			log.Printf("unmarshal error: %v, message: %s", err, message)
			continue
		}

		if r.ID != nil {
			if ch, ok := m.response[*r.ID]; ok {
				ch <- r
			} else {
				log.Printf("unexpected response id: %d, msg: %s", *r.ID, message)
			}
		} else {
			m.messages <- message
		}
	}
}

func (m *binanceWs) request(ctx context.Context, method string, params ...string) (*Response, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if m.closed() {
		return nil, ErrClosed
	}
	m.requestIDCounter++
	requestID := m.requestIDCounter
	req := wsRequest{
		Method: method,
		Params: params,
		ID:     requestID,
	}
	debugLog.Printf("request: %+v", req)
	err := m.ws.WriteJSON(req)
	if err != nil {
		return nil, fmt.Errorf("request error: %w", err)
	}
	m.response[requestID] = make(chan Response, 1)
	select {
	case res := <-m.response[requestID]:
		if res.Error() != nil {
			return nil, res.Error()
		}
		return &res, nil
	case <-ctx.Done():
		return nil, fmt.Errorf("request timeout")
	}
}

func (m *binanceWs) readMessage() ([]byte, error) {
	select {
	case msg := <-m.messages:
		return msg, nil
	case <-m.closeCh:
		return nil, ErrClosed
	}
}

func (m *binanceWs) close() {
	m.mu.Lock()
	defer m.mu.Unlock()
	close(m.closeCh)
	err := m.ws.Close()
	if err != nil {
		log.Printf("error when closing websocket: %s", err)
	}
}

func (m *binanceWs) closed() bool {
	select {
	case <-m.closeCh:
		return true
	default:
		return false
	}
}
