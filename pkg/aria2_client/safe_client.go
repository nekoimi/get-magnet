package aria2_client

import (
	"errors"
	"github.com/cenkalti/rpc2"
	"github.com/nekoimi/arigo"
	"log"
	"sync"
	"time"
)

type SafeClient struct {
	jsonrpc string
	secret  string

	cmu    *sync.Mutex
	client *arigo.Client
}

func New(jsonrpc string, secret string) *SafeClient {
	client, err := arigo.Dial(jsonrpc, secret)
	if err != nil {
		panic(err)
	}

	return &SafeClient{
		jsonrpc: jsonrpc,
		secret:  secret,

		cmu:    &sync.Mutex{},
		client: client,
	}
}

func (sc *SafeClient) Client() *arigo.Client {
	sc.cmu.Lock()
	defer sc.cmu.Unlock()

	err := sc.ping()
	if err != nil {
		if errors.Is(err, rpc2.ErrShutdown) {
			for {
				err := sc.reconnect()
				if err != nil {
					log.Printf("Check the rpc connection is closed, reconnect... %s\n", err.Error())
					time.Sleep(5 * time.Second)
					continue
				}

				break
			}
		}
	}

	return sc.client
}

func (sc *SafeClient) ping() error {
	_, err := sc.client.GetVersion()
	return err
}

func (sc *SafeClient) reconnect() error {
	client, err := arigo.Dial(sc.jsonrpc, sc.secret)
	if err != nil {
		return err
	}
	sc.client = client
	return nil
}
