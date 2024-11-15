package main

import (
	"github.com/kickback-space/litmus"
	. "github.com/blitz-frost/log"
	"os"
	"os/signal"
)

func main() {

	defer Close()

	go sigint()

	path := ""
	go func() {
		Log(Info, "Litmus server online.")
		err := litmus.Listen(8000, path)
		if err != nil {
			Err(Critical, "network litmus server listen", err)
		}
	}()

	select {}
}

func sigint() {
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)

	<-c

	Close()
	os.Exit(1)
}
