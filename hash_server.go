package main

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"
)

const (
	hashDelayIntervalSeconds = 5
	gracefulShutdownTimeout  = 30
	httpBadRequest           = 400
	defaultServerListenAddr  = ":8080"
)

type hashStore struct {
	hashedDataMutex   sync.Mutex
	hashedDataCounter int
	hashedData        map[int]string

	hashRequestProcessingDurationsMutex sync.Mutex
	hashRequestProcessingDurations      []int64
}

var gracefulShutdownRequestChan = make(chan bool, 1)

func main() {
	var listenAddr string
	flag.StringVar(&listenAddr, "listen-addr", defaultServerListenAddr, "server listen address")
	flag.Parse()

	logger := log.New(os.Stdout, "http: ", log.LstdFlags)

	serverShutdownComplete := make(chan bool, 1)

	hashStore := hashStore{
		hashedDataCounter:              0,
		hashedData:                     make(map[int]string),
		hashRequestProcessingDurations: make([]int64, 0, 100),
	}
	server := initHashServer(logger, &hashStore, listenAddr)
	go gracefulShutdown(server, logger, gracefulShutdownRequestChan, serverShutdownComplete)

	logger.Println("Server is ready to handle requests at", listenAddr)
	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		logger.Fatalf("Could not listen on %s: %v\n", listenAddr, err)
	}

	<-serverShutdownComplete
	logger.Println("Server stopped")
}

func initHashServer(logger *log.Logger, store *hashStore, listenAddr string) *http.Server {
	router := http.NewServeMux()

	router.HandleFunc("/hash", store.hash)
	router.HandleFunc("/hash/", store.hash)
	router.HandleFunc("/stats", store.stats)
	router.HandleFunc("/shutdown", shutdown)

	return &http.Server{
		Addr:     listenAddr,
		Handler:  router,
		ErrorLog: logger,
	}
}

func (hs *hashStore) hash(w http.ResponseWriter, r *http.Request) {
	if r.Method == "GET" {
		idStr := strings.TrimPrefix(r.URL.Path, "/hash/")
		if idStr != "" {
			id, err := strconv.Atoi(idStr)
			if err != nil {
				http.Error(w, "Invalid hash id.", httpBadRequest)
			} else {
				hs.hashedDataMutex.Lock()
				if id <= hs.hashedDataCounter && id >= 1 {
					if val, ok := hs.hashedData[id]; ok {
						fmt.Fprintf(w, val)
					} else {
						http.Error(w, "Hash not generated yet.", httpBadRequest)
					}
				} else {
					http.Error(w, "Index out of range.", httpBadRequest)
				}
				hs.hashedDataMutex.Unlock()
			}

			return
		} else {
			http.Error(w, "Missing hash id parameter.", httpBadRequest)
			return
		}
	}

	defer hs.storeHashRequestProcessingDuration(time.Now())

	err := r.ParseForm()
	if err != nil {
		log.Printf("unable to parse form: %v", err)
		return
	}

	password := []byte(r.Form.Get("password"))
	hs.hashedDataMutex.Lock()
	hs.hashedDataCounter += 1
	hashId := hs.hashedDataCounter
	hashFunc := hs.hashAndEncode(password, hashId)
	hs.hashedDataMutex.Unlock()
	time.AfterFunc(hashDelayIntervalSeconds*time.Second, hashFunc)

	fmt.Fprintf(w, "%v", hashId)
}

func (hs *hashStore) hashAndEncode(data []byte, hashId int) func() {
	return func() {
		h := sha256.New()
		h.Write(data)
		hash := h.Sum(nil)

		hs.hashedDataMutex.Lock()
		hs.hashedData[hashId] = base64.StdEncoding.EncodeToString(hash)
		hs.hashedDataMutex.Unlock()
	}
}

func (hs *hashStore) storeHashRequestProcessingDuration(start time.Time) {
	hs.hashRequestProcessingDurationsMutex.Lock()
	duration := time.Since(start).Microseconds()
	hs.hashRequestProcessingDurations = append(hs.hashRequestProcessingDurations, duration)
	hs.hashRequestProcessingDurationsMutex.Unlock()
}

func (hs *hashStore) stats(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	stats := make(map[string]int64)

	hs.hashRequestProcessingDurationsMutex.Lock()
	numRequests := int64(len(hs.hashRequestProcessingDurations))
	stats["total"] = numRequests
	var totalProcessingTime int64
	for i := 0; i < int(numRequests); i++ {
		totalProcessingTime += hs.hashRequestProcessingDurations[i]
	}
	var average int64 = 0
	if numRequests != 0 {
		average = totalProcessingTime / numRequests
	}
	stats["average"] = average
	hs.hashRequestProcessingDurationsMutex.Unlock()

	err := json.NewEncoder(w).Encode(stats)
	if err != nil {
		log.Printf("failed to send json: %v", err)
	}
}

func gracefulShutdown(server *http.Server, logger *log.Logger, gracefulShutdownRequestChan <-chan bool, serverShutdownComplete chan<- bool) {
	<-gracefulShutdownRequestChan
	logger.Println("Server is shutting down...")

	ctx, cancel := context.WithTimeout(context.Background(), gracefulShutdownTimeout*time.Second)
	defer cancel()

	server.SetKeepAlivesEnabled(false)
	if err := server.Shutdown(ctx); err != nil {
		logger.Fatalf("Could not gracefully shutdown the server: %v\n", err)
	}
	close(serverShutdownComplete)
}

func shutdown(w http.ResponseWriter, r *http.Request) {
	close(gracefulShutdownRequestChan)
}
