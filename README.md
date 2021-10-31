# binancestream
A go library for consuming Binance Websocket Market Streams

This library handles network failures by automatically reconnecting to the websocket.

## Usage
```go
bs := binancestream.New(nil)
defer bs.Close()

var err error

err = bs.Subscribe("btcusdt@kline_1m", func(cs binancestream.CombinedStream) {
    fmt.Printf("Stream name: %s, Data: %s\n", cs.Name, cs.Data)
})
if err != nil {
    log.Fatal(err)
}

err = bs.Subscribe("ethusdt@kline_1m", func(cs binancestream.CombinedStream) {
    fmt.Printf("Stream name: %s, Data: %s\n", cs.Name, cs.Data)
})
if err != nil {
    log.Fatal(err)
}

bs.Wait()
```

If you want to find stream names to subscribe, you can find them here: https://binance-docs.github.io/apidocs/spot/en/#websocket-market-streams