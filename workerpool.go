package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"time"
)

var mode string

const (
	MaxQueue  = 10000
	MaxWorker = 10000
)

var JobQueue chan Job
var Queue chan Payload

func init() {
	flag.StringVar(&mode, "m", "Default", "Choose mode of ratelimiter")
	Queue = make(chan Payload, MaxQueue)
	JobQueue = make(chan Job, MaxQueue)
}

// Data to Upload
type Payload struct{}

func (p *Payload) Update() error {
	time.Sleep(500 * time.Millisecond)
	return nil
}

/*
 * Default PayloadHandler
 */
func PayloadHandler(w http.ResponseWriter, r *http.Request) {
	var p Payload
	go p.Update()
	w.Write([]byte("Success!!"))
}

/*
 * Buffered Channel PayloadHandler
 */
func StartProcessor() {
	for {
		select {
		case Payload := <-Queue:
			Payload.Update()
		}
	}
}

func QueuePayloadHandler(w http.ResponseWriter, r *http.Request) {
	var p Payload
	Queue <- p
	w.Write([]byte("Success!!"))
}

/*
 * Job Channel & Worker Pool PayloadHandler
 */

type Job struct {
	payload Payload
}

type Worker struct {
	workerPool chan chan Job
	jobChannel chan Job
	quit       chan bool
}

func NewWorker(workerPool chan chan Job) Worker {
	return Worker{
		workerPool: workerPool,
		jobChannel: make(chan Job),
		quit:       make(chan bool),
	}
}

func (w Worker) Start() {
	go func() {
		for {
			w.workerPool <- w.jobChannel
			select {
			case job := <-w.jobChannel:
				time.Sleep(500 * time.Millisecond)
				fmt.Printf("Update Success: %v\n", job)
			case <-w.quit:
				return
			}
		}
	}()
}

func (w Worker) Stop() {
	go func() {
		w.quit <- true
	}()
}

type Dispatcher struct {
	workerPool chan chan Job
}

func NewDispatcher(maxWorkers int) *Dispatcher {
	pool := make(chan chan Job, maxWorkers)
	return &Dispatcher{workerPool: pool}
}

func (d *Dispatcher) Run() {
	for i := 0; i < MaxWorker; i++ {
		worker := NewWorker(d.workerPool)
		worker.Start()
	}
	go d.dispatch()
}

func (d *Dispatcher) dispatch() {
	for {
		select {
		case job := <-JobQueue:
			go func(job Job) {
				jobChannel := <-d.workerPool
				jobChannel <- job
			}(job)
		}
	}
}

func WorkerPayloadHandler(w http.ResponseWriter, r *http.Request) {
	work := Job{payload: Payload{}}
	JobQueue <- work
	w.Write([]byte("Success!!"))
}

func main() {
	flag.Parse()

	switch mode {
	case "BufferedChannel":
		http.HandleFunc("/", QueuePayloadHandler)
		go StartProcessor()
	case "WorkerPool":
		d := NewDispatcher(MaxWorker)
		d.Run()
		http.HandleFunc("/", WorkerPayloadHandler)
	default:
		http.HandleFunc("/", PayloadHandler)
	}

	log.Fatal(http.ListenAndServe(":8080", nil))
}
