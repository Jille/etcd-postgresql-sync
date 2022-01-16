-- name: AddKey :exec
INSERT INTO etcd.kv (key, value) VALUES ($1, $2);

-- name: AddKeys :copyfrom
INSERT INTO etcd.kv (key, value) VALUES ($1, $2);

-- name: UpdateKey :exec
UPDATE etcd.kv SET value=$1 WHERE key=$2;

-- name: DeleteKey :exec
DELETE FROM etcd.kv WHERE key = $1;

-- name: DeleteAll :exec
DELETE FROM etcd.kv;
