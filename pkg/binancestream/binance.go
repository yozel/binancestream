package binancestream

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"sync"
	"time"
)

type StreamSubscriptionHandler func(CombinedStream)

type CombinedStream struct {
	Name string          `json:"stream"`
	Data json.RawMessage `json:"data"`
}

type BinanceStream struct {
	wsm *binanceWs

	defaultHandler StreamSubscriptionHandler

	subscriptions   map[string]StreamSubscriptionHandler
	subscriptionsMu sync.RWMutex
}

func New(defaultHandler StreamSubscriptionHandler) (*BinanceStream, error) {
	bs := &BinanceStream{
		subscriptions:  make(map[string]StreamSubscriptionHandler),
		defaultHandler: defaultHandler,
		wsm:            newBinanceWs(),
	}
	return bs, bs.run()
}

func (b *BinanceStream) run() error {
	go func() {
		for {
			msg, err := b.wsm.readMessage()
			if err != nil {
				if err == ErrClosed {
					return
				} else {
					log.Printf("failed to read message: %s", err)
				}
			}
			go b.messageDispatcher(msg)
		}
	}()
	err := b.wsm.connect()
	if err != nil {
		return fmt.Errorf("failed to connect: %s", err)
	}
	go func() {
		for {
			err = b.wsm.pollMessages()
			switch err {
			case ErrClosed:
				return
			case ErrWSDisconnected:
				for {
					err := b.wsm.connect()
					if err != nil {
						log.Printf("failed to connect: %s", err)
						time.Sleep(time.Second * 5)
						continue
					}
					break
				}
				b.subscriptionsMu.RLock()
				for sub := range b.subscriptions {
					b.subscribe(sub)
				}
				b.subscriptionsMu.RUnlock()
			default:
				panic(fmt.Errorf("unexpected error value: %s", err))
			}

		}
	}()
	return nil
}

func (b *BinanceStream) Wait() {
	<-b.wsm.closeCh
}

func (b *BinanceStream) Close() {
	b.wsm.close()
}

func (b *BinanceStream) Subscribe(stream string, handler StreamSubscriptionHandler) error {
	if _, ok := b.subscriptions[stream]; ok {
		return fmt.Errorf("subscription already exists: %s", stream)
	}
	b.subscriptionsMu.Lock()
	defer b.subscriptionsMu.Unlock()
	b.subscriptions[stream] = handler
	err := b.subscribe(stream)
	if err != nil {
		delete(b.subscriptions, stream)
		return err
	}
	return nil
}

func (b *BinanceStream) subscribe(stream string) error {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()
	_, err := b.wsm.request(ctx, "SUBSCRIBE", stream)
	if err != nil {
		return fmt.Errorf("failed to subscribe: %s", err)
	}
	subs, err := b.listSubscriptions()
	if err != nil {
		return fmt.Errorf("failed to subscribe: %s", err)
	}
	if !contains(subs, stream) {
		return fmt.Errorf("failed to subscribe: %s not in %s", stream, subs)
	}
	return nil
}

func (b *BinanceStream) Unsubscribe(stream string) error {
	if _, ok := b.subscriptions[stream]; !ok {
		return fmt.Errorf("subscription does not exist: %s", stream)
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()
	_, err := b.wsm.request(ctx, "UNSUBSCRIBE", stream)
	if err != nil {
		return fmt.Errorf("failed to unsubscribe: %s", err)
	}
	subs, err := b.listSubscriptions()
	if err != nil {
		return fmt.Errorf("failed to unsubscribe: %s", err)
	}
	if contains(subs, stream) {
		return fmt.Errorf("failed to unsubscribe: %s not in %s", stream, subs)
	}
	b.subscriptionsMu.Lock()
	defer b.subscriptionsMu.Unlock()
	delete(b.subscriptions, stream)
	return nil
}

func (b *BinanceStream) listSubscriptions() ([]string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()
	r, err := b.wsm.request(ctx, "LIST_SUBSCRIPTIONS")
	if err != nil {
		return nil, fmt.Errorf("failed to list subscriptions: %s", err)
	}
	var subs []string
	err = json.Unmarshal(r.Result, &subs)
	if err != nil {
		return nil, fmt.Errorf("failed to list subscriptions: %s", err)
	}
	return subs, nil
}

func (b *BinanceStream) messageDispatcher(message []byte) {
	var c CombinedStream
	err := json.Unmarshal(message, &c)
	if err != nil {
		log.Printf("failed to unmarshal message: %s", err)
	} else {
		if handler, ok := b.subscriptions[c.Name]; ok {
			handler(c)
			return
		} else {
			if b.defaultHandler != nil {
				b.defaultHandler(c)
			}
		}
	}
}
