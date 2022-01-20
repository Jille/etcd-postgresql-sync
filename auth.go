package main

import (
	"context"
	"log"
	"runtime"
	"strings"
	"time"

	authdump "github.com/Jille/etcd-auth-dump"
	"github.com/Jille/etcd-postgresql-sync/database"
	"github.com/Jille/etcd-postgresql-sync/database/gendb"
	"go.etcd.io/etcd/api/v3/v3rpc/rpctypes"
	"google.golang.org/grpc/codes"
)

var (
	lastSyncedAuthRevision uint64
)

func authSyncLoop(ctx context.Context) {
	for {
		if err := syncAuth(ctx); err != nil {
			log.Printf("Error syncing authentication config: %v", err)
		}
		time.Sleep(10 * time.Minute)
	}
}

func syncAuth(ctx context.Context) error {
	commands, rev, err := authdump.Dump(ctx, c, lastSyncedAuthRevision)
	if err != nil {
		if err == authdump.ErrUnchanged {
			return nil
		}
		if ee, ok := err.(rpctypes.EtcdError); ok && ee.Code() == codes.PermissionDenied {
			log.Printf("Fatal error syncing authentication (does the syncer have role root?): %v [disabling authentication syncing]", err)
			runtime.Goexit()
		}
		return err
	}
	if err := database.RunTransaction(ctx, func(q *gendb.Queries) error {
		if err := q.DeleteAuth(ctx); err != nil {
			return err
		}
		return q.SetAuth(ctx, strings.Join(commands, "\n"))
	}); err != nil {
		return err
	}
	lastSyncedAuthRevision = rev
	return nil
}
