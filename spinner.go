// Nebula Spinner server to maintain socket connections to Captains.
package spinner

import (
  "github.com/gorilla/mux"
  "github.com/phayes/freeport"
  "github.com/armadanet/comms"
  "github.com/armadanet/captain/dockercntrl"
  "net/http"
  "log"
  "strconv"
  "time"
  "os"
)

// Server for the Nebula Spinner
type Server interface {
  // Given a port of 0, assigns a free port to the server.
  Run(beaconURL string, port int)
}

type server struct {
  router          *mux.Router
  handler         *Handler
  state           *dockercntrl.State
  container_name  string
  overlay_name    string
}

// Produces a new Server interface of struct server
func New(container_name string) (Server, error) {
  router := mux.NewRouter().StrictSlash(true)
  handler := NewHandler()
  state, err := dockercntrl.New()
  if err != nil {return nil, err}
  router.HandleFunc("/join", join(handler)).Name("Join")
  router.HandleFunc("/spin", spin(handler)).Name("Spin")
  handler.Start()
  return &server{
    router: router,
    handler: handler,
    state: state,
    container_name: container_name,
  }, nil
}

type newSpinnerRes struct {
  SwarmToken        string  `json:"SwarmToken"`
  BeaconIp          string  `json:"BeaconIp"`
  BeaconOverlay     string  `json:"BeaconOverlay"`
  BeaconName        string  `json:"BeaconName"`
  SpinnerOverlay    string  `json:"SpinnerOverlay"`
}

// Runs the spinner server.
func (s *server) Run(beaconURL string, port int) {
  // Query beacon
  log.Println("New Spinner query beacon...")
  var res newSpinnerRes
  err := comms.SendPostRequest(beaconURL, map[string]string{
    "SpinnerId":s.container_name,
  }, &res)
  if err!=nil {
    log.Println(err)
    return
  }
  s.overlay_name = res.SpinnerOverlay

  // join beacon swarm and attach self to beacon overlay
  err = s.state.JoinSwarmAndOverlay(res.SwarmToken, res.BeaconIp, s.container_name, res.BeaconOverlay)
  if err != nil {
    log.Println(err)
    return
  }

  // attach self to spinner_overlay
  err = s.state.JoinOverlay(s.container_name, res.SpinnerOverlay)
  if err != nil {
    log.Println(err)
    return
  }

  // go routine periodically ping beacon to notify the alive (wait 1s)
  go s.ping(res.BeaconName)

  // start the server
  go s.startServer(port)

  // spinner notifies parent captain
  selfSpin, err := strconv.ParseBool(os.Getenv("SELFSPIN"))
  if err != nil {
    log.Println(err)
    return
  }
  if selfSpin {
    // captain name for bridge network communication
    captain_url := os.Getenv("CAPTAIN_URL")
    // notify the parent captain spinner overlay_name for it to join
    err = comms.SendPostRequest(captain_url, map[string]string{
      "OverlayName":res.SpinnerOverlay,
    }, nil)
    if err!=nil {
      log.Println(err)
      return
    }
  }

  // TODO: change the exit logic
  select {
  case <- make(chan interface{}):
  }
}

func (s *server) startServer(port int) {
  if port == 0 {
    var err error
    port, err = freeport.GetFreePort()
    if err != nil {log.Println(err); return}
  }
  log.Fatal(http.ListenAndServe(":" + strconv.Itoa(port), s.router))
}

func (s *server) ping(beaconName string) {
  for {
    // err := comms.SendPostRequest("http://localhost:8787/register", map[string]interface{}{
    err := comms.SendPostRequest("http://"+beaconName+":8787/register", map[string]interface{}{
      "Id":s.container_name,
      "OverlayName":s.overlay_name,
      "LastUpdate":time.Now(),
    }, nil)
    if err!=nil {
      panic(err)
      return
    }
    // fmt.Println(beaconName)
    // ping every 3 seconds
    time.Sleep(3 * time.Second)
  }
}
