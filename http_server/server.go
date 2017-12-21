package http_server

import (
	"net/http"
	"log"
	"os"
	"time"
	"encoding/json"
	"sync"
	"golang.org/x/net/context"
	"errors"
	"fmt"
	"github.com/dropbox/godropbox/sync2"
)

// stored items
type ResultItem struct {
	Key uint32    `json:"key,omitempty"`
	Value string `json:"value,omitempty"`
}

// service implementation
type Server struct {
	logger *log.Logger
	srvMux *http.ServeMux

	id sync2.AtomicUint32
	strs map[uint32]string
	mux sync.RWMutex
}

// service factory
func New() *Server {
	s := &Server{
		logger: log.New(os.Stdout, "", 0),
		srvMux: http.NewServeMux(),
		strs:   make(map[uint32]string, 0),
	}

	s.srvMux.HandleFunc("/api/strings", s.Handle)

	return s
}

// serve function
func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.logger.Printf("request received. method:%s", r.Method)
	s.srvMux.ServeHTTP(w, r)
}

// handler function
func (s *Server) Handle(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 1 * time.Second)
	defer cancel()

	var (
		result interface{}
		err error
	)

	switch r.Method {
	default:
		err = errors.New("unsupported method")
	case "GET":
		result = s.get(ctx)
	case "POST":
		result, err = s.add(r.WithContext(ctx))
	case "PUT":
		result, err = s.update(r.WithContext(ctx))
	case "DELETE":
		err = s.delete(r.WithContext(ctx))
		result = "OK"
	}

	if r.Method == "GET" {
		result = s.get(ctx)
	}

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")

	resStr, err := json.Marshal(result)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Write(resStr)
}

func (s *Server) get(ctx context.Context) []ResultItem {
	s.mux.RLock()
	defer s.mux.RUnlock()

	result := make([]ResultItem, 0, len(s.strs))
	for id, val := range s.strs {
		result = append(result, ResultItem{
			Key: id,
			Value: val,
		})
	}

	return result
}

func (s *Server) decode(req *http.Request) (ResultItem, error) {
	item := ResultItem{}
	decoder := json.NewDecoder(req.Body)

	err := decoder.Decode(&item)
	if err != nil {
		return ResultItem{}, err
	}

	return item, nil
}

func (s *Server) add(req *http.Request) (ResultItem, error) {
	item, err := s.decode(req)
	if err != nil {
		return item, err
	}

	id := s.id.Add(1)

	s.mux.Lock()
	defer s.mux.Unlock()

	item.Key = id
	s.strs[id] = item.Value

	return item, nil
}

func (s *Server) update(req *http.Request) (ResultItem, error) {
	item, err := s.decode(req)
	if err != nil {
		return item, err
	}

	s.mux.Lock()
	defer s.mux.Unlock()

	if _, ok := s.strs[item.Key]; !ok {
		return ResultItem{}, errors.New(fmt.Sprintf("item with id %d not found", item.Key))
	}

	s.strs[item.Key] = item.Value

	return item, nil
}

func (s *Server) delete(req *http.Request) error {
	item, err := s.decode(req)
	if err != nil {
		return err
	}

	s.mux.Lock()
	defer s.mux.Unlock()

	if _, ok := s.strs[item.Key]; !ok {
		return errors.New(fmt.Sprintf("item with id %d not found", item.Key))
	}

	delete(s.strs, item.Key)

	return nil
}


