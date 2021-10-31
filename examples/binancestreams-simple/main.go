package main

import (
	"fmt"
	"log"

	"github.com/yozel/binancestream"
)

func main() {
	// binancestream.EnableDebugLogger()

	bs := binancestream.New(nil)
	defer bs.Close()

	err := bs.Subscribe("btcusdt@kline_1m", func(cs binancestream.CombinedStream) {
		fmt.Printf("Stream name: %s, Data: %s\n", cs.Name, cs.Data)
	})
	if err != nil {
		log.Fatal(err)
	}

	bs.Wait()
}
