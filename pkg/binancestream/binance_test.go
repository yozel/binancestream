package binancestream

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSubscribeUnsubscribe(t *testing.T) {
	bs := New(nil)
	assert.NotNil(t, bs)
	// defer bs.Close()

	var err error

	err = bs.Subscribe("btcusdt@ticker", func(CombinedStream) {})
	assert.NoError(t, err)
	err = bs.Subscribe("ethusdt@ticker", func(CombinedStream) {})
	assert.NoError(t, err)

	err = bs.Unsubscribe("btcusdt@ticker")
	assert.NoError(t, err)
	err = bs.Unsubscribe("ethusdt@ticker")
	assert.NoError(t, err)
}
