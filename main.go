package main

import (
	"net/http"
	"os"
	"strings"
)

func init() {

	accessToken := os.Getenv("NATUREREMO_TOKEN")
	applianceName := os.Getenv("APPLIANCE")
	signalNames := strings.Split(os.Getenv("SIGNALS"), ",")

	s, err := NewServer(accessToken, applianceName, signalNames)
	if err != nil {
		panic(err)
	}

	http.Handle("/change", s)
	http.Handle("/", http.FileServer(http.Dir("static")))
}
