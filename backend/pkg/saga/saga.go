// Package saga wraps the DTM client so services share one way of opening
// distributed sagas and guarding branch handlers with a sub-transaction
// barrier. The place-order and cancel-refund flows can run on either engine:
// "inline" (hand-rolled orchestration inside the order service) or "dtm"
// (DTM server coordinates branches and drives compensations automatically).
package saga

import (
	"fmt"

	"github.com/dtm-labs/client/dtmcli"
	"github.com/dtm-labs/client/dtmcli/dtmimp"
	"github.com/dtm-labs/client/dtmgrpc"
)

// Saga engine selector values for the `saga.engine` config key.
const (
	EngineInline = "inline"
	EngineDTM    = "dtm"
)

// BarrierTable is where the DTM branch barrier writes its idempotency rows.
// Each saga-participant database ships a migration creating this table.
const BarrierTable = "public.dtm_barrier"

// Setup configures the DTM client globals for Postgres and this repo's
// barrier table. Call once at service start, before any barrier use.
func Setup() {
	dtmcli.SetCurrentDBType(dtmimp.DBTypePostgres)
	dtmcli.SetBarrierTableName(BarrierTable)
}

// ResolveEngine normalises the configured engine value; anything other than
// an explicit "inline" runs through DTM (the default engine).
func ResolveEngine(configured string) string {
	if configured == EngineInline {
		return EngineInline
	}
	return EngineDTM
}

// NewSaga opens a synchronous gRPC saga against the DTM server at dtmAddr.
// WaitResult makes Submit block until the saga reaches a terminal state, so
// the caller can report success/rollback to the end user in-request.
// Branch action/compensate URLs use DTM's gRPC form:
//
//	"host:port/package.Service/Method"
func NewSaga(dtmAddr string) (*dtmgrpc.SagaGrpc, error) {
	gid, err := genGid(dtmAddr)
	if err != nil {
		return nil, err
	}
	sg := dtmgrpc.NewSagaGrpc(dtmAddr, gid)
	sg.WaitResult = true
	return sg, nil
}

// genGid asks the DTM server for a fresh global transaction id. The client
// lib only offers a panicking variant, so recover into a plain error — a dead
// DTM server must fail the request, not crash the service.
func genGid(dtmAddr string) (gid string, err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("saga: dtm server unreachable at %s: %v", dtmAddr, r)
		}
	}()
	return dtmgrpc.MustGenGid(dtmAddr), nil
}
