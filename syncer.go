package main

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/Jille/etcd-postgresql-sync/database"
	"github.com/Jille/etcd-postgresql-sync/database/gendb"
	clientv3 "go.etcd.io/etcd/client/v3"
)

var (
	c *clientv3.Client
)

func main() {
	ctx := context.Background()
	database.Init()
	log.Printf("Connecting to etcd...")
	var err error
	c, err = clientv3.New(clientv3.Config{
		Endpoints:        []string{"https://etcd_etcd_1:2379"},
		DialTimeout:      15 * time.Second,
		AutoSyncInterval: 5 * time.Minute,
		TLS: &tls.Config{
			InsecureSkipVerify: true,
		},
		Username: os.Getenv("ETCD_USER"),
		Password: os.Getenv("ETCD_PASSWORD"),
	})
	if err != nil {
		log.Fatalf("Failed to connect to etcd: %v", err)
	}
	defer c.Close()
	log.Printf("Connected.")
	go c.Sync(ctx)

	for {
		if err := syncLoop(ctx); err != nil {
			log.Print(err)
		}
		time.Sleep(10 * time.Second)
	}
}

func syncLoop(ctx context.Context) error {
	resp, err := c.Get(ctx, "", clientv3.WithPrefix())
	if err != nil {
		return fmt.Errorf("Failed to retrieve initial keys: %v", err)
	}
	rows := make([]gendb.AddKeysParams, len(resp.Kvs))
	for i, kv := range resp.Kvs {
		rows[i].Key = string(kv.Key)
		rows[i].Value = string(kv.Value)
	}
	resp.Kvs = nil // Allow garbage collection.
	if err := database.RunTransaction(ctx, func(q *gendb.Queries) error {
		if err := q.DeleteAll(ctx); err != nil {
			return err
		}
		_, err := q.AddKeys(ctx, rows)
		return err
	}); err != nil {
		return err
	}
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	wc := c.Watch(ctx, "", clientv3.WithPrefix(), clientv3.WithRev(resp.Header.Revision))
	for wr := range wc {
		if err := wr.Err(); err != nil {
			return err
		}
		if wr.Header.Revision == resp.Header.Revision {
			// When we subscribe on revision X, we get an immediate notification if X was a revision that changed our watched files.
			// We just fetched that, so we should ignore this to avoid reapplying the same change.
			continue
		}
		if err := database.RunTransaction(ctx, func(q *gendb.Queries) error {
			for _, e := range wr.Events {
				switch e.Type {
				case clientv3.EventTypePut:
					if e.Kv.CreateRevision == e.Kv.ModRevision {
						if err := q.AddKey(ctx, gendb.AddKeyParams{Key: string(e.Kv.Key), Value: string(e.Kv.Value)}); err != nil {
							return err
						}
					} else {
						if err := q.UpdateKey(ctx, gendb.UpdateKeyParams{Key: string(e.Kv.Key), Value: string(e.Kv.Value)}); err != nil {
							return err
						}
					}
				case clientv3.EventTypeDelete:
					if err := q.DeleteKey(ctx, string(e.Kv.Key)); err != nil {
						return err
					}
				}
			}
			return nil
		}); err != nil {
			return err
		}
	}
	return errors.New("watch channel was closed unexpectedly")
}
