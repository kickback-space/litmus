package litmus

import (
	"net/http"
	"strconv"
	"sync"

	. "github.com/blitz-frost/log"
)

type Server struct {
	port     uint
	pathBase string
	connections sync.Map
}

func NewServer(port uint) *Server {
	return &Server{
		port: port,
	}
}

func (s *Server) Listen(port uint, pathBase string) error {
	addr := ":" + strconv.FormatUint(uint64(port), 10)

	path := "/litmus"
	if pathBase != "" {
		path = "/" + pathBase + path
	}

	handle := func(w http.ResponseWriter, r *http.Request) {
		Log(Info, "litmus connection attempt")
		var err error
		defer func() {
			if err != nil {
				Err(Error, "litmus connection failed", err)
			}
		}()

		err = s.handleConnection(w, r)
	}

	healthPath := path + "/health"
	healthHandle := func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("Litmus OK"))
	}

	mux := http.NewServeMux()
	mux.HandleFunc(path, handle)
	mux.HandleFunc(healthPath, healthHandle)

	return http.ListenAndServe(addr, mux)
}

func Listen(port uint, pathBase string) error {
	server := NewServer(port)
	return server.Listen(port, pathBase)
}