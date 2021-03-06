// Code generated by sqlc. DO NOT EDIT.
// source: copyfrom.go

package gendb

import (
	"context"
)

// iteratorForAddKeys implements pgx.CopyFromSource.
type iteratorForAddKeys struct {
	rows                 []AddKeysParams
	skippedFirstNextCall bool
}

func (r *iteratorForAddKeys) Next() bool {
	if len(r.rows) == 0 {
		return false
	}
	if !r.skippedFirstNextCall {
		r.skippedFirstNextCall = true
		return true
	}
	r.rows = r.rows[1:]
	return len(r.rows) > 0
}

func (r iteratorForAddKeys) Values() ([]interface{}, error) {
	return []interface{}{
		r.rows[0].Key,
		r.rows[0].Value,
	}, nil
}

func (r iteratorForAddKeys) Err() error {
	return nil
}

func (q *Queries) AddKeys(ctx context.Context, arg []AddKeysParams) (int64, error) {
	return q.db.CopyFrom(ctx, []string{"etcd", "kv"}, []string{"key", "value"}, &iteratorForAddKeys{rows: arg})
}
