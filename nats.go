package gotssse

// import (
// 	"context"
// 	"encoding/json"

// 	"github.com/nats-io/nats.go"
// 	"go.uber.org/zap"
// )

// type natsPublisher struct {
// 	l *zap.Logger
// }

// func getNatsSubject(audience Audience, event Event) string {
// 	return string(audience) + "." + string(event)
// }

// func NewNatsPublisherFunc(ctx context.Context, l *zap.Logger, nc *nats.Conn) (f PublisherFunc, err error) {
// 	return func(ctx context.Context, audience Audience, event Event, data interface{}) error {
// 		jsonBytes, err := json.Marshal(data)
// 		if err != nil {
// 			return err
// 		}
// 		return nc.Publish(getNatsSubject(audience, event), jsonBytes)
// 	}, nil
// }

// func NewNatsSSEHandlerFunc(ctx context.Context, l *zap.Logger, nc *nats.Conn) (f SSEHandlerFunc, err error) {
// 	return
// }
