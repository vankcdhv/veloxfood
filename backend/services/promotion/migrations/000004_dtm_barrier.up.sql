-- DTM sub-transaction barrier table (schema per dtm-labs client): each saga
-- branch call inserts (gid, branch_id, op, barrier_id) in the same transaction
-- as the business change, making duplicate calls and null-compensations no-ops.
-- The constraint MUST be named uniq_barrier — the client's Postgres
-- insert-ignore template references it by name.
CREATE TABLE IF NOT EXISTS dtm_barrier (
    id          BIGSERIAL PRIMARY KEY,
    trans_type  VARCHAR(45)  NOT NULL DEFAULT '',
    gid         VARCHAR(128) NOT NULL DEFAULT '',
    branch_id   VARCHAR(128) NOT NULL DEFAULT '',
    op          VARCHAR(45)  NOT NULL DEFAULT '',
    barrier_id  VARCHAR(45)  NOT NULL DEFAULT '',
    reason      VARCHAR(45)  NOT NULL DEFAULT '',
    create_time TIMESTAMPTZ  NOT NULL DEFAULT now(),
    update_time TIMESTAMPTZ  NOT NULL DEFAULT now(),
    CONSTRAINT uniq_barrier UNIQUE (gid, branch_id, op, barrier_id)
);
