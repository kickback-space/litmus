package litmus

import (
	"net/http"
	"strconv"
	"sync"

	. "github.com/blitz-frost/log"
)

type Server struct {
	port        uint
	pathBase    string
	connections sync.Map
}

func NewServer(port uint) *Server {
	return &Server{
		port: port,
	}
}

// RegisterHandlers registers litmus-specific HTTP handlers to the provided ServeMux.
func (s *Server) RegisterHandlers(mux *http.ServeMux, pathBase string) {
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

	mux.HandleFunc(path, handle)
	mux.HandleFunc(healthPath, healthHandle)
}


// Optionally, retain the Listen function for standalone usage
func (s *Server) ListenStandalone(pathBase string) error {
	addr := ":" + strconv.FormatUint(uint64(s.port), 10)
	mux := http.NewServeMux()
	s.RegisterHandlers(mux, pathBase)
	return http.ListenAndServe(addr, mux)
}
