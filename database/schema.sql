CREATE SCHEMA etcd;

CREATE TABLE etcd.kv (
	key TEXT NOT NULL PRIMARY KEY,
	value TEXT NOT NULL
);

-- This table will have exactly one row.
CREATE TABLE etcd.auth (
	commands TEXT NOT NULL
);
