package gotssse

import (
	"context"
	"net/http"
)

type (
	Event                 string
	Audience              string
	PublisherFunc         func(ctx context.Context, audience Audience, event Event, data interface{}) error
	AudienceConnectorFunc func(w http.ResponseWriter, r *http.Request) (audience Audience, events map[Event]struct{}, err error)
	SSEHandlerFunc        http.HandlerFunc
)
