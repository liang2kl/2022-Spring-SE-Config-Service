package push

import (
	"service/internal/utils"
	"sync"

	"github.com/gorilla/websocket"
)

type client struct {
	// channel to send notification
	ch       chan ConfigUpdateNotification
	configId string
}

type ConfigUpdateNotification struct {
	UpdateTime int64  `json:"update_time"`
	ConfigID   string `json:"config_id"`
}

type pushService struct {
	upgrader websocket.Upgrader
	clients  map[string][]*client
	closeCh  chan int
	sync.RWMutex
}

// need to acquire lock before calling
func getClients(configId string) []*client {
	clients, exist := service.clients[configId]
	if !exist {
		clients = []*client{}
	}
	return clients
}

func (service *pushService) addClient(c *client) {
	service.Lock()
	defer service.Unlock()

	configId := c.configId

	clients := getClients(configId)
	service.clients[configId] = append(clients, c)
}

func (service *pushService) removeClient(c *client) {
	service.Lock()
	defer service.Unlock()

	configId := c.configId

	clients := getClients(configId)

	index := utils.Find(clients, c)
	if index < 0 {
		return
	}

	service.clients[configId] = utils.Remove(clients, index)
}
