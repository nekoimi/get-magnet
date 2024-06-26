package aria2

import (
	"get-magnet/internal/model"
	"github.com/siku2/arigo"
	"log"
)

type Aria2 struct {
	client     *arigo.Client
	magnetChan chan *model.MagnetItem
}

func New() *Aria2 {
	client, err := arigo.Dial("wss://aria2.sakuraio.com/jsonrpc", "nekoimi")
	if err != nil {
		panic(err)
	}

	aria := &Aria2{
		client:     client,
		magnetChan: make(chan *model.MagnetItem),
	}

	return aria
}

func (aria *Aria2) Submit(item *model.MagnetItem) {
	aria.magnetChan <- item
}

func (aria *Aria2) Run() {
	for {
		select {
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

func (aria *Aria2) Stop() {
	log.Println("aria2 client close")
	err := aria.client.Close()
	if err != nil {
		log.Printf("aria2 client close err: %s \n", err.Error())
	}
}
