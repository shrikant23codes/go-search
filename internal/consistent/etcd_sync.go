package consistent

import (
	"context"
	"encoding/json"
	"log"
	"time"

	clientV3 "go.etcd.io/etcd/client/v3"
)

const etcdPrefix = "/gosearch/nodes/"

type EtcdWatcher struct {
	client *clientV3.Client
	ring   *Ring
}

func NewEtcdWatcher(endpoints []string, ring *Ring) (*EtcdWatcher, error) {
	cli, err := clientV3.New(clientV3.Config{
		Endpoints:   endpoints,
		DialTimeout: 5 * time.Second,
	})

	if err != nil {
		return nil, err
	}

	return &EtcdWatcher{
		client: cli,
		ring:   ring,
	}, nil
}

func (w *EtcdWatcher) LoadExistingNodes(ctx context.Context) error {
	resp, err := w.client.Get(ctx, etcdPrefix, clientV3.WithPrefix())
	if err != nil {
		return err
	}

	for _, kv := range resp.Kvs {
		var node Node
		if err := json.Unmarshal(kv.Value, &node); err != nil {
			continue
		}
		w.ring.Add(node)
		log.Printf("etcd: loaded node %s at %s", node.ID, node.Address)
	}
	return nil
}

func (w *EtcdWatcher) Watch(ctx context.Context) {
	ch := w.client.Watch(ctx, etcdPrefix, clientV3.WithPrefix())
	for resp := range ch {
		for _, ev := range resp.Events {
			switch ev.Type {
			case clientV3.EventTypePut:
				var node Node
				if err := json.Unmarshal(ev.Kv.Value, &node); err != nil {
					log.Printf("etcd watch: bad node value: %v", err)
					continue
				}
				w.ring.Add(node)
				log.Printf("etcd watch: new node %s joined at %s", node.ID, node.Address)
			case clientV3.EventTypeDelete:
				nodeID := string(ev.Kv.Key)[len(etcdPrefix):] // remove the prefix to get node ID
				w.ring.Remove(nodeID)
				log.Printf("etcd: node left %s", nodeID)
			}
		}
	}
}

func (w *EtcdWatcher) Close() {
	w.client.Close()
}
