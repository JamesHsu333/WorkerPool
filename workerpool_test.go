package main

import (
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
)

func TestPayloadHandler(t *testing.T) {
	s := httptest.NewServer(http.HandlerFunc(PayloadHandler))
	defer s.Close()

	var wg sync.WaitGroup
	n := 10000
	wg.Add(n)
	for i := 1; i <= n; i++ {
		go func() {
			_, err := http.Get(s.URL)
			if err != nil {
				t.Error("Error during requests", err.Error())
			}
			wg.Done()
		}()
	}
	wg.Wait()
}

func TestQueuePayloadHandler(t *testing.T) {
	s := httptest.NewServer(http.HandlerFunc(QueuePayloadHandler))
	defer s.Close()
	go StartProcessor()

	var wg sync.WaitGroup
	n := 15000
	wg.Add(n)
	for i := 1; i <= n; i++ {
		go func() {
			_, err := http.Get(s.URL)
			if err != nil {
				t.Error("Error during requests", err.Error())
			}
			wg.Done()
		}()
	}
	wg.Wait()
}

func TestWorkerPayloadHandler(t *testing.T) {
	d := NewDispatcher(MaxWorker)
	d.Run()
	s := httptest.NewServer(http.HandlerFunc(WorkerPayloadHandler))
	defer s.Close()
	var wg sync.WaitGroup
	n := 10000
	wg.Add(n)
	for i := 1; i <= n; i++ {
		go func() {
			_, err := http.Get(s.URL)
			if err != nil {
				t.Error("Error during requests", err.Error())
			}
			wg.Done()
		}()
	}
	wg.Wait()
}
