package spinner

import (
  "github.com/armadanet/captain/dockercntrl"
  "github.com/armadanet/comms"
  "github.com/google/uuid"
  "log"
)

type Request struct {
  Success  chan bool
  Task     *Task
}

type Task struct {
  Config  *dockercntrl.Config
  From    *uuid.UUID
}

type Handler struct {
  clients         *comms.Messenger
  clientMetaData  map[uuid.UUID]int
  Requester       *comms.Messenger
  Register        chan *comms.Instance
  Unregister      chan *comms.Instance
  Request         chan *Request
}

func NewHandler() *Handler {
  h := &Handler{
    clients: comms.NewMessenger(),
    clientMetaData: make(map[uuid.UUID]int),
    Requester: comms.NewMessenger(),
    Register: make(chan *comms.Instance),
    Unregister: make(chan *comms.Instance),
    Request: make(chan *Request),
  }
  h.clients.Start()
  h.Requester.Start()
  return h
}

func (h *Handler) run() {
  defer func() {
    log.Println("Handler Complete")
  }()
  for {
    log.Println("Handler Action")
    select {
    case client := <- h.Register:
      h.clientMetaData[*client.Id] = 0
      h.clients.Register <- client
    case client := <- h.Unregister:
      delete(h.clientMetaData, *client.Id)
      h.clients.Unregister <- client
    case request := <- h.Request:
      // Round-Robin, extract away to Schedule type
      log.Printf("Round Robin Scheduling\n")
      minimum := -1
      var chosen uuid.UUID
      for k,v := range h.clientMetaData {
        log.Printf("%v - %v (min:%v)\n", k, v, minimum)
        if (minimum == -1) || (v < minimum) {
          chosen = k; minimum = v
        }
      }
      if minimum == -1 {request.Success <- false; break}
      h.clientMetaData[chosen]++
      log.Printf("Chosen: %+v\n", chosen)
      h.clients.Message <- &comms.Message{
        Success: request.Success,
        Reciever: &chosen,
        Data: request.Task,
      }
    }
  }
}

func (h *Handler) Start() {go h.run()}

func (h *Handler) SendTask(from *comms.Instance, task *dockercntrl.Config) bool {
  response := make(chan bool)
  req := &Request{
    Success: response,
    Task: &Task{
      From: from.Id,
      Config: task,
    },
  }
  h.Request <- req
  status := <- response
  return status
}
