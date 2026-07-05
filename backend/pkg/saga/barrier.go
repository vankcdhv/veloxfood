package saga

import (
	"context"
	"database/sql"

	"github.com/dtm-labs/client/dtmgrpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// RunWithBarrier executes fn inside a database transaction, guarded by the
// DTM sub-transaction barrier when ctx carries DTM branch metadata (i.e. the
// call came from the DTM server). The barrier inserts an idempotency row
// keyed by (gid, branch_id, op) into dtm_barrier within the SAME transaction
// as fn, so at the database level:
//   - a duplicate branch call becomes a no-op,
//   - a compensation for a never-executed action ("null compensation")
//     becomes a no-op, and the late-arriving action is suppressed too.
//
// Calls without DTM metadata — the inline saga engine and Kafka consumers —
// run fn in a plain transaction, so handlers stay engine-agnostic.
func RunWithBarrier(ctx context.Context, db *gorm.DB, fn func(tx *gorm.DB) error) error {
	bb, err := dtmgrpc.BarrierFromGrpc(ctx)
	if err != nil || bb == nil || bb.Gid == "" {
		return db.WithContext(ctx).Transaction(fn)
	}
	sqlDB, err := db.DB()
	if err != nil {
		return err
	}
	return bb.CallWithDB(sqlDB, func(tx *sql.Tx) error {
		gtx, err := gormOverTx(ctx, tx)
		if err != nil {
			return err
		}
		return fn(gtx)
	})
}

// gormOverTx wraps an already-open *sql.Tx in a gorm handle so repository
// code written against *gorm.DB can join the barrier's transaction. The
// handle must not Begin/Commit — the barrier owns the transaction lifecycle.
func gormOverTx(ctx context.Context, tx *sql.Tx) (*gorm.DB, error) {
	g, err := gorm.Open(postgres.New(postgres.Config{Conn: tx}), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		return nil, err
	}
	return g.WithContext(ctx), nil
}

// Abort marks a gRPC branch response as a business failure: DTM stops
// retrying the branch and rolls the whole saga back through the registered
// compensations. Any other error return means "retry me later".
func Abort(msg string) error {
	return status.Error(codes.Aborted, msg)
}
