package test

import (
	"context"
	"net/http"

	"github.com/foomo/gotssse"
)

const AudienceAll gotssse.Audience = "all"
const EventUpdateState gotssse.Event = "updateState"

type State struct {
	Foo string `json:"foo"`
	Bar string `json:"bar"`
}

type Service interface {
	SetFoo(w http.ResponseWriter, r *http.Request, foo string) State
	SetBar(w http.ResponseWriter, r *http.Request, bar string) State
}

type service struct {
	state State
	pf    gotssse.PublisherFunc
}

func NewService(pf gotssse.PublisherFunc) Service {
	return &service{
		state: State{
			Foo: "",
			Bar: "",
		},
		pf: pf,
	}
}

func (s *service) SetFoo(w http.ResponseWriter, r *http.Request, foo string) State {
	s.state.Foo = foo
	s.publishUpdateState(r.Context())
	return s.state
}

func (s *service) SetBar(w http.ResponseWriter, r *http.Request, bar string) State {
	s.state.Bar = bar
	s.publishUpdateState(r.Context())
	return s.state
}

func (s *service) publishUpdateState(ctx context.Context) {
	s.pf(ctx, AudienceAll, EventUpdateState, s.state)
}
