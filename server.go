package main

import (
	"context"
	"errors"
	"net/http"
	"sync"

	"github.com/tenntenn/natureremo"
	"google.golang.org/appengine"
	"google.golang.org/appengine/log"
	"google.golang.org/appengine/urlfetch"
)

type Server struct {
	accessToken string
	appliance   string
	signalNames []string
	signals     map[string]*natureremo.Signal

	mu      sync.RWMutex
	current int
}

func NewServer(accessToken, appliance string, signalNames []string) (*Server, error) {
	s := &Server{
		accessToken: accessToken,
		appliance:   appliance,
		signalNames: signalNames,
	}

	return s, nil
}

func (s *Server) getSignal(ctx context.Context, cli *natureremo.Client, n int) (*natureremo.Signal, error) {
	a, err := s.getAppliance(ctx, cli, s.appliance)
	if err != nil {
		return nil, err
	}

	for _, sig := range a.Signals {
		if sig.Name == s.signalNames[n] {
			return sig, nil
		}
	}

	return nil, errors.New("signal not found")
}

func (s *Server) getAppliance(ctx context.Context, cli *natureremo.Client, name string) (*natureremo.Appliance, error) {
	as, err := cli.ApplianceService.GetAll(ctx)
	if err != nil {
		return nil, err
	}

	for _, a := range as {
		if a.Nickname == name {
			return a, nil
		}
	}

	return nil, errors.New("appliance not found")
}

func (s *Server) sendSignal(ctx context.Context, cli *natureremo.Client, name string) error {

	var n int
	for i := range s.signalNames {
		if name == s.signalNames[i] {
			n = i
			break
		}
	}

	sig, err := s.getSignal(ctx, cli, n)
	if err != nil {
		return err
	}

	if err := cli.SignalService.Send(ctx, sig); err != nil {
		return err
	}

	s.mu.Lock()
	s.current = n
	s.mu.Unlock()

	return nil
}

func (s *Server) sendNext(ctx context.Context, cli *natureremo.Client) error {
	s.mu.RLock()
	n := (s.current + 1) % len(s.signalNames)
	s.mu.RUnlock()

	sig, err := s.getSignal(ctx, cli, n)
	if err != nil {
		return err
	}

	if err := cli.SignalService.Send(ctx, sig); err != nil {
		return err
	}

	s.mu.Lock()
	s.current++
	s.mu.Unlock()

	return nil
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctx := appengine.NewContext(r)
	cli := natureremo.NewClient(s.accessToken)
	cli.HTTPClient = urlfetch.Client(ctx)

	if name := r.FormValue("s"); name != "" {
		if err := s.sendSignal(ctx, cli, name); err != nil {
			code := http.StatusInternalServerError
			log.Errorf(ctx, "%s", err)
			http.Error(w, http.StatusText(code), code)
		}
		return
	}

	if err := s.sendNext(ctx, cli); err != nil {
		code := http.StatusInternalServerError
		log.Errorf(ctx, "%s", err)
		http.Error(w, http.StatusText(code), code)
		return
	}
}
