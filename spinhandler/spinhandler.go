package spinhandler

import (
	// "github.com/armadanet/spinner/spincomm"
	"github.com/armadanet/spinner/spinclient"
	"github.com/armadanet/spinner/spincomm"
	"sync"
	// "context"
	// "errors"
)

type handler struct {
	mutex 		*sync.Mutex
	clientmap	*clientmap
}

type Handler interface{
	AddClient(client spinclient.Client) error
	RemoveClient(id string) error
	ChooseClient(ch Chooser, req *spincomm.TaskRequest) (string, string, error)
	ListClientIds() []string
	GetClient(id string) (spinclient.Client, bool)
	// ConnectClient(id string) error
	UpdateClient(status *spincomm.NodeInfo) error
}

func New() Handler {
	return &handler{
		mutex: &sync.Mutex{},
		clientmap: newclientmap(),
	}
}

func (h *handler) AddClient(client spinclient.Client) error {
	err := h.clientmap.add(client)
	return err
}

func (h *handler) RemoveClient(id string) error {
	err := h.clientmap.remove(id)
	return err
}

func (h *handler) ChooseClient(ch Chooser, req *spincomm.TaskRequest) (string, string, error) {
	return ch.F(h.clientmap, req)
}

func (h *handler) ListClientIds() []string {
	return h.clientmap.Keys()
}

func (h *handler) GetClient(id string) (spinclient.Client, bool) {
	return h.clientmap.Get(id)
}

func (h *handler) UpdateClient(status *spincomm.NodeInfo) error {
	return h.clientmap.update(status)
}

