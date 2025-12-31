package db

import "database/sql"

// sqlc's [Queries.WithTx] doesn't return the [Querier] interface so we have to wrap it.

type QuerierWrapper struct {
	*Queries
}

func NewQuerierWrapper(q *Queries) *QuerierWrapper {
	return &QuerierWrapper{Queries: q}
}

func (w *QuerierWrapper) WithTx(tx *sql.Tx) QuerierWithTxSupport {
	return NewQuerierWrapper(w.Queries.WithTx(tx))
}

type QuerierWithTxSupport interface {
	WithTx(tx *sql.Tx) QuerierWithTxSupport
	Querier
}
