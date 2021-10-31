package binancestream

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func init() {
	EnableDebugLogger()
}

func Test_SubscribeRequest(t *testing.T) {
	var ctx context.Context
	var cancel context.CancelFunc
	var err error

	m := newBinanceWs()
	err = m.connect()
	assert.Nil(t, err)
	go m.readPump()
	defer m.close()

	ctx, cancel = context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	r, err := m.request(ctx, "SUBSCRIBE", "btcusdt@kline_1m")
	assert.NoError(t, err)
	assert.NotNil(t, r)
	assert.EqualValues(t, r.Result, []byte("null"))

	ctx, cancel = context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	r, err = m.request(ctx, "SUBSCRIBEX", "btcusdtsss@kline_1m")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "Invalid request: unknown variant `SUBSCRIBEX`")
	assert.Nil(t, r)
}
