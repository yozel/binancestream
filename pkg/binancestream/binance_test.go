package binancestream

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSubscribeUnsubscribe(t *testing.T) {
	var err error
	bs, err := New(nil)
	assert.Nil(t, err)

	err = bs.Subscribe("btcusdt@ticker", func(CombinedStream) {})
	assert.NoError(t, err)
	err = bs.Subscribe("ethusdt@ticker", func(CombinedStream) {})
	assert.NoError(t, err)

	err = bs.Unsubscribe("btcusdt@ticker")
	assert.NoError(t, err)
	err = bs.Unsubscribe("ethusdt@ticker")
	assert.NoError(t, err)
}
