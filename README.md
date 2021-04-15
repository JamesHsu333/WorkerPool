# Simple Go Worker Pool
[![Go Reference](https://pkg.go.dev/badge/github.com/JamesHsu333/WorkerPool.svg)](https://pkg.go.dev/github.com/JamesHsu333/WorkerPool)
[![Go Report Card](https://goreportcard.com/badge/github.com/JamesHsu333/WorkerPool)](https://goreportcard.com/report/github.com/JamesHsu333/WorkerPool)

Experiment of Go Worker Pool
## Scenario
Client sends a request, and Server receives the request to process the data
```
Client                  Server                   Database
+------------+          +-------------+          +-------------+
|   Client   |  +---->  |   Server    |  +---->  |  Database   |
|            |          |             |          |             |
+------------+          +-------------+          +-------------+
```
## How to start
```bash
# run Default mode
go run workerpool.go
# run Buffered Channel mode
go run workerpool.go -m "BufferedChannel"
# run Worker Pool mode
go run workerpool.go -m "WorkerPool"

# test Default mode with 10000 request
go test -v -count=-1 -run TestPayloadHandler
# test Buffered Channel mode with 10000 request
go test -v -count=-1 -run TestQueuePayloadHandler
# test Worker Pool mode with 10000 request
go test -v -count=-1 -run TestWorkerPayloadHandler
```
## Default Mode
```go
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

func main() {
	http.HandleFunc("/", PayloadHandler)
	log.Fatal(http.ListenAndServe(":8080", nil))
}
```
### Pros:
* Easy to use
### Cons:
* Not work very well at a large scale

## Bufferd Channel Mode
```go
const MaxQueue  = 10000
var Queue chan Payload

func init() {
	Queue = make(chan Payload, MaxQueue)
}

// Data to Upload
type Payload struct{}

func (p *Payload) Update() error {
	time.Sleep(500 * time.Millisecond)
	return nil
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
}

func main() {
	http.HandleFunc("/", QueuePayloadHandler)
	go StartProcessor()
	log.Fatal(http.ListenAndServe(":8080", nil))
}
```
### Pros:
* Avoid the unlimited goroutines
### Cons:
* The speed of incoming requests under high concurrency will far exceed the processing speed
* Once the channel is full, subsequent requests will be blocked waiting

## Worker Pool Mode
```go
const (
	MaxQueue  = 10000
	MaxWorker = 10000
)
var JobQueue chan Job

func init() {
	JobQueue = make(chan Job, MaxQueue)
}

// Data to Upload
type Payload struct{}

func (p *Payload) Update() error {
	time.Sleep(500 * time.Millisecond)
	return nil
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
	d := NewDispatcher(MaxWorker)
	d.Run()
	http.HandleFunc("/", WorkerPayloadHandler)
	log.Fatal(http.ListenAndServe(":8080", nil))
}
```
### Idea:
Create a 2-tier channel system, one for queuing jobs and another to control how many workers operate on the JobQueue concurrently.\
The idea was to parallelize the update to a sustainable rate, one that would not cripple the machine nor start generating connections errors from database.
### Architecture:
```
Clients                  Server                                                          Database
+------------+                                                                           
|   Client   |  +---     +---------------------------------------------------------+     
|            |     |     |  JobQueue                           WorkerPool          |     
+------------+     |     |  +----------------+  Get Available  +----------------+  |    +----------+
+------------+     |     |  |+Job+ +Job+ ... |     Worker      |   +Worker1+    |+----> |          |
|   Client   |  +----->  |  |+Job+ +Job+ ... | +-------------> |     . . .      |  |    | Database |
|            |     |     |  |+Job+ +Job+ ... |                 |   +Workern+    |+----> |          |
+------------+     |     |  +----------------+                 +----------------+  |    +----------+
+------------+     |     |                                                         |     
|   Client   |  +---     +---------------------------------------------------------+     
|            |                                                   
+------------+                                                   

```

## References
[[1]](http://marcio.io/2015/07/handling-1-million-requests-per-minute-with-golang/) Marcio Castilho. Handling 1 Million Requests per Minute with Go. 2015