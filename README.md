# etcd-postgresql-syncer

A simple tool to sync your etcd cluster to PostgreSQL in realtime. It sets up a watcher on etcd and commits all changes to PostgreSQL.

I use it to have a realtime backup of my etcd data. My PostgreSQL database is already replicated offsite, so I can leverage those benefits for free.

Note that this syncer is asynchronous, so you might lose a few transactions when your cluster dies and you use PostgreSQL to restore your data from. (But that's fewer transactions than using a nightly backup.) If you want a synchronous replica, set up another etcd node offsite (which comes with a latency cost, of course).

## Setup

```
postgres=# CREATE DATABASE etcd;
postgres=# CREATE USER etcd_syncer PASSWORD 'hackme';
postgres=# GRANT ALL PRIVILEGES ON DATABASE etcd TO etcd_syncer;

psql -U etcd_syncer etcd -f database/schema.sql

etcdctl role add postgres_syncer
etcdctl role grant-permission postgres_syncer read "" --prefix
etcdctl user add postgres_syncer
> Enter password hackme2
etcdctl user grant-role postgres_syncer postgres_syncer

ETCD_ENDPOINTS=https://127.0.0.1:2379 ETCD_USER=postgres_syncer ETCD_PASSWORD=hackme2 DATABASE_DSN="user=etcd_syncer password=hackme host=127.0.0.1 port=5432 dbname=etcd" etcd-postgresql-sync
```

## Parameters

All configuration is passed in through environment variables. It takes these settings:

- ETCD_ENDPOINTS is where to find your etcd cluster
- ETCD_USER and ETCD_PASSWORD are used to connect to etcd. No authentication is used if you leave them unset/empty.
- DATABASE_DSN specifies how to connect to PostgreSQL.
- SYNCER_DEBUG can be set to "true" to make it log all queries sent to PostgreSQL.

See the Setup section for example values.

## Future improvements

Currently, when the syncer starts it loads all data from etcd in memory and then starts pushing it to PostgreSQL. We could keep track of the revision we've synced up to and start watching again from that point, and only need to do a full copy when that revision has been compacted.
