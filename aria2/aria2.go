package aria2

import (
	"context"
	"get-magnet/internal/model"
	"github.com/siku2/arigo"
	"log"
)

type Aria2 struct {
	client     *arigo.Client
	magnetChan chan model.MagnetItem
	closeChan  chan string
}

func New() *Aria2 {
	aria := &Aria2{
		magnetChan: make(chan model.MagnetItem),
		closeChan:  make(chan string, 2),
	}
	aria.connect()
	return aria
}

func (aria *Aria2) Submit(item model.MagnetItem) {
	aria.magnetChan <- item
}

func (aria *Aria2) Run(ctx context.Context) {
	go aria.waitDisconnection()

	for {
		select {
		case <-ctx.Done():
			log.Println("aria2 client close...")
			err := aria.client.Close()
			if err != nil {
				log.Printf("aria2 client close err: %s \n", err.Error())
			}
			return
		case item := <-aria.magnetChan:
			magnetUri := item.OptimalLink
			g, err := aria.client.AddURI(arigo.URIs(magnetUri), nil)
			if err != nil {
				log.Printf("add uri (%s) to aria2 err: %s \n", magnetUri, err.Error())
			}
			g.Subscribe(arigo.StartEvent, func(event *arigo.DownloadEvent) {
				log.Printf("StartEvent#%s \n", g.GID)
			})
		}
	}
}

// connect connect jsonrpc server
func (aria *Aria2) connect() {
	client, err := arigo.Dial("wss://aria2.sakuraio.com/jsonrpc", "nekoimi")
	if err != nil {
		panic(err)
	}
	aria.client = &client
}

// waitDisconnection wait disconnection and reconnect
func (aria *Aria2) waitDisconnection() {
	for {
		closeMsg := <-aria.closeChan
		log.Printf("aria2 client close: %s \n", closeMsg)
		// TODO aria.connect()
	}
}
