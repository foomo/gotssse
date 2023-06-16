package test

import (
	"context"
	"encoding/json"
	http "net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/foomo/gotssse"
	"github.com/r3labs/sse/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

// "github.com/nats-io/nats.go"
// "github.com/stretchr/testify/require"
//	func getConn(t *testing.T) *nats.Conn {
//		nc, err := nats.Connect(nats.DefaultURL)
//		require.NoError(t, err)
//		return nc
//	}

func setup(t *testing.T) (ctx context.Context, cancel context.CancelFunc, l *zap.Logger, rcpProxyTestServer, sseTestServer *httptest.Server) {
	ctx = context.Background()
	ctx, cancel = context.WithTimeout(ctx, 10*time.Second)
	l, err := zap.NewDevelopment()
	require.NoError(t, err)
	pf, hf, err := gotssse.GetChannelHandlers(ctx, l, func(w http.ResponseWriter, r *http.Request) (audience gotssse.Audience, events map[gotssse.Event]struct{}, err error) {
		return "all", map[gotssse.Event]struct{}{
			"updateState": {},
		}, nil
	})
	proxy := NewDefaultServiceGoTSRPCProxy(NewService(pf))
	require.NoError(t, err)
	rcpProxyTestServer = httptest.NewServer(proxy)
	sseTestServer = httptest.NewServer(http.HandlerFunc(hf))
	return
}

func Test(t *testing.T) {
	ctx, cancel, l, rcpProxyTestServer, sseTestServer := setup(t)
	sseClient := sse.NewClient(sseTestServer.URL)
	chanState := make(chan State)
	go func() {
		err := sseClient.SubscribeWithContext(ctx, string(EventUpdateState), func(msg *sse.Event) {
			nextState := State{}
			json.Unmarshal(msg.Data, &nextState)
			l.Info(
				"sse client msg",
				zap.String("event", string(msg.Event)),
				zap.String("id", string(msg.ID)),
			)
			chanState <- nextState
		})
		require.NoError(t, err)
	}()
	// wait for sse client connection - I wish there was a nicer way too
	time.Sleep(100 * time.Millisecond)
	client := NewDefaultServiceGoTSRPCClient(rcpProxyTestServer.URL)

	const (
		fooo = "foooooo"
		baar = "baar"
	)

	nextState, err := client.SetFoo(ctx, fooo)
	require.NoError(t, err)
	sseState := <-chanState
	assert.Equal(t, fooo, nextState.Foo)
	assert.Equal(t, nextState, sseState)

	nextState, err = client.SetBar(ctx, baar)
	require.NoError(t, err)
	sseState = <-chanState
	assert.Equal(t, baar, nextState.Bar)
	assert.Equal(t, nextState, sseState)

	cancel()
	rcpProxyTestServer.Close()
	sseTestServer.Close()
	l.Sync()

}
