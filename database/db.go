package database

import (
	"context"
	"errors"
	"log"
	"os"
	"time"

	"github.com/Jille/errchain"
	"github.com/Jille/etcd-postgresql-sync/database/gendb"
	"github.com/jackc/pgconn"
	"github.com/jackc/pgx/v4"
	"github.com/jackc/pgx/v4/log/logrusadapter"
	"github.com/jackc/pgx/v4/pgxpool"
	"github.com/sirupsen/logrus"
)

var (
	db      *pgxpool.Pool
	queries *gendb.Queries
)

type TransactionRunner func(*gendb.Queries) error

func Init() {
	cc, err := pgxpool.ParseConfig(os.Getenv("DATABASE_DSN"))
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	cc.MaxConns = 1
	cc.MinConns = 0
	if cc.ConnConfig.RuntimeParams["application_name"] == "" {
		cc.ConnConfig.RuntimeParams["application_name"] = "etcd-postgres-syncer"
	}
	if os.Getenv("SYNCER_DEBUG") != "" {
		cc.ConnConfig.Logger = logrusadapter.NewLogger(&logrus.Logger{
			Out:          os.Stderr,
			Formatter:    new(logrus.JSONFormatter),
			Hooks:        make(logrus.LevelHooks),
			Level:        logrus.InfoLevel,
			ExitFunc:     os.Exit,
			ReportCaller: false,
		})
	}

	db, err = pgxpool.ConnectConfig(context.Background(), cc)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	queries = gendb.New(db)
}

func RunTransaction(ctx context.Context, runner TransactionRunner) error {
	txo := pgx.TxOptions{
		IsoLevel:   pgx.ReadCommitted,
		AccessMode: pgx.ReadWrite,
	}
	for attempt := 0; ; attempt++ {
		err := runTransactionOnce(ctx, txo, runner)
		if attempt >= 3 {
			return err
		}
		switch ToSQLState(err) {
		case "40001", "40P01":
			// TODO: Exponential backoff
			time.Sleep(500 * time.Millisecond)
			continue
		}
		return err
	}
}

func runTransactionOnce(ctx context.Context, txo pgx.TxOptions, runner TransactionRunner) error {
	tx, err := db.BeginTx(ctx, txo)
	if err != nil {
		return err
	}
	q := queries.WithTx(tx)
	if err := runner(q); err != nil {
		return errchain.Chain(err, tx.Rollback(ctx))
	}
	return tx.Commit(ctx)
}

func ToSQLState(err error) string {
	var pge *pgconn.PgError
	if errors.As(err, &pge) {
		return pge.SQLState()
	}
	return ""
}
