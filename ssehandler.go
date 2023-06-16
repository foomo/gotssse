package gotssse

import (
	"context"
	"encoding/json"
	"math"
	"net/http"
	"strconv"

	"go.uber.org/zap"
)

func sseHandle(
	serverCtx context.Context,
	l *zap.Logger,
	w http.ResponseWriter,
	r *http.Request,
	cf AudienceConnectorFunc,
	connect func(audience Audience, events map[Event]struct{}) (conn connection, err error),
	disconnect func(conn connection),
) {
	l = l.With(zap.String("handler", "sse"))
	l.Info("incoming request")
	w.Header().Set("Content-Type", "text/event-stream")

	audience, events, err := cf(w, r)
	if err != nil {
		http.Error(w, "connection error", http.StatusInternalServerError)
		return
	}

	flusher, ok := w.(http.Flusher)
	if !ok {
		l.Error("could not get flusher")
		return
	}

	conn, err := connect(audience, events)
	if err != nil {
		l.Error("could not connect", zap.Error(err))
		http.Error(w, "conn err", http.StatusInternalServerError)
		return
	}
	close := func() {
		disconnect(conn)
	}
	i := uint64(0)
	for {
		select {
		case evt := <-conn.chanMsg:
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
		case <-serverCtx.Done():
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
