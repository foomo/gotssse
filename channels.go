package gotssse

import (
	"context"
	"encoding/json"
	"math"
	"net/http"
	"strconv"

	"go.uber.org/zap"
)

type message struct {
	audience Audience
	event    Event
	data     interface{}
}

type chanMessage chan message

func GetChannelHandlers(ctx context.Context, l *zap.Logger, connectFunc AudienceConnectorFunc) (pf PublisherFunc, hf SSEHandlerFunc, err error) {
	chanPublishState := make(chanMessage)
	sseHandler := newSSEHandler(ctx, l, chanPublishState, connectFunc)
	return func(ctx context.Context, audience Audience, event Event, data interface{}) error {
			go func() {
				chanPublishState <- message{
					audience: audience,
					event:    event,
					data:     data,
				}
			}()
			return nil
		},
		sseHandler.ServeHTTP,
		nil
}

type connection struct {
	audience Audience
	events   map[Event]struct{}
	chanMsg  chanMessage
}

type channelSSE struct {
	chanConnect   chan connection
	l             *zap.Logger
	connectorFunc AudienceConnectorFunc
	serverCtx     context.Context
}

func newSSEHandler(
	ctx context.Context,
	l *zap.Logger,
	chanCollaborationState chanMessage,
	connectorFunc AudienceConnectorFunc,
) http.Handler {
	sse := &channelSSE{
		chanConnect:   make(chan connection, 10),
		l:             l,
		connectorFunc: connectorFunc,
		serverCtx:     ctx,
	}
	go sse.run(ctx, l, chanCollaborationState)
	return sse
}

func (sse *channelSSE) run(
	ctx context.Context,
	l *zap.Logger,
	chanPublish chanMessage,
) {
	connections := make(map[chanMessage]connection)
	for {
		select {
		case newConnection := <-sse.chanConnect:
			_, exists := connections[newConnection.chanMsg]
			if exists {
				delete(connections, newConnection.chanMsg)
			} else {
				connections[newConnection.chanMsg] = newConnection
			}
		case collabStateToBroadcast := <-chanPublish:
			for _, conn := range connections {
				if conn.audience == collabStateToBroadcast.audience {
					_, ok := conn.events[collabStateToBroadcast.event]
					if ok {
						go func(chanMsg chanMessage) { chanMsg <- collabStateToBroadcast }(conn.chanMsg)
					}
				}
			}
		case <-ctx.Done():
			l.Info("context is done")
			sse.chanConnect = nil
			return
		}
	}
}

func (sse *channelSSE) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	sseHandle(
		sse.serverCtx, sse.l, w, r, sse.connectorFunc,
		func(audience Audience, events map[Event]struct{}) (conn connection, err error) {
			conn = connection{
				audience: audience,
				events:   events,
				chanMsg:  make(chanMessage),
			}
			sse.chanConnect <- conn
			return conn, nil
		},
		func(conn connection) {
			chanConnect := sse.chanConnect
			if chanConnect != nil {
				chanConnect <- conn
			}
		},
	)
}

func (sse *channelSSE) _ServeHTTP(w http.ResponseWriter, r *http.Request) {

	l := sse.l.With(zap.String("handler", "ssebus"))
	l.Info("incoming request")
	w.Header().Set("Content-Type", "text/event-stream")

	audience, events, err := sse.connectorFunc(w, r)
	if err != nil {
		http.Error(w, "connection error", http.StatusInternalServerError)
		return
	}

	flusher, ok := w.(http.Flusher)
	if !ok {
		l.Error("could not get flusher")
		return
	}
	chanMsg := make(chanMessage)
	conn := connection{
		chanMsg:  chanMsg,
		audience: audience,
		events:   events,
	}
	sse.chanConnect <- conn

	close := func() {
		chanConnect := sse.chanConnect
		if chanConnect != nil {
			chanConnect <- conn
		}
	}

	i := uint64(0)
	for {
		select {
		case evt := <-chanMsg:
			i++
			if i == math.MaxUint64 {
				i = 0
			}
			eventID := strconv.FormatUint(i, 10)
			jsonDataBytes, err := json.Marshal(evt.data)
			if err != nil {
				l.Error("could not marshal client state", zap.Error(err))
				close()
				return
			}

			_, err = w.Write([]byte(
				"event: " + string(evt.event) + "\n" +
					"id: " + eventID + "\n" +
					"data: " + string(jsonDataBytes) + "\n\n",
			))

			if err != nil {
				l.Error("connection pbly closed", zap.Error(err))
				close()
				return
			}
			flusher.Flush()
		case <-sse.serverCtx.Done():
			l.Info("server context is done")
			close()
			return
		case <-r.Context().Done():
			l.Info("request context is done")
			close()
			return
		}
	}

}
