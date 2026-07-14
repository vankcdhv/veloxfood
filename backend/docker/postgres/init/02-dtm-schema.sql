-- DTM server transaction storage (from dtm-labs/dtm sqls/dtmsvr.storage.postgres.sql,
-- minus the destructive DROPs). Applied to the dedicated dtm_db.
--
-- Mounted into docker-entrypoint-initdb.d as 02-dtm-schema.sql, which psql runs
-- against POSTGRES_DB — hence the explicit \c. Without these tables the DTM
-- server answers every Submit with `relation "trans_global" does not exist`, and
-- every place-order saga fails.
\c dtm_db
CREATE SEQUENCE if not EXISTS trans_global_seq;
CREATE TABLE if not EXISTS trans_global (
  id bigint NOT NULL DEFAULT NEXTVAL ('trans_global_seq'),
  gid varchar(128) NOT NULL,
  trans_type varchar(45) not null,
  status varchar(45) NOT NULL,
  query_prepared varchar(1024) NOT NULL,
  protocol varchar(45) not null,
  create_time timestamp(0) with time zone DEFAULT NULL,
  update_time timestamp(0) with time zone DEFAULT NULL,
  finish_time timestamp(0) with time zone DEFAULT NULL,
  rollback_time timestamp(0) with time zone DEFAULT NULL,
  options varchar(1024) DEFAULT '',
  custom_data varchar(1024) DEFAULT '',
  next_cron_interval int default null,
  next_cron_time timestamp(0) with time zone default null,
  owner varchar(128) not null default '',
  ext_data text,
  result varchar(1024) DEFAULT '',
  rollback_reason varchar(1024) DEFAULT '',
  PRIMARY KEY (id),
  CONSTRAINT gid UNIQUE (gid)
);
create index if not EXISTS owner on trans_global(owner);
create index if not EXISTS status_next_cron_time on trans_global (status, next_cron_time);
CREATE SEQUENCE if not EXISTS trans_branch_op_seq;
CREATE TABLE IF NOT EXISTS trans_branch_op (
  id bigint NOT NULL DEFAULT NEXTVAL ('trans_branch_op_seq'),
  gid varchar(128) NOT NULL,
  url varchar(1024) NOT NULL,
  data TEXT,
  bin_data bytea,
  branch_id VARCHAR(128) NOT NULL,
  op varchar(45) NOT NULL,
  status varchar(45) NOT NULL,
  finish_time timestamp(0) with time zone DEFAULT NULL,
  rollback_time timestamp(0) with time zone DEFAULT NULL,
  create_time timestamp(0) with time zone DEFAULT NULL,
  update_time timestamp(0) with time zone DEFAULT NULL,
  PRIMARY KEY (id),
  CONSTRAINT gid_branch_uniq UNIQUE (gid, branch_id, op)
);
CREATE SEQUENCE if not EXISTS kv_seq;
CREATE TABLE IF NOT EXISTS kv (
    id bigint NOT NULL DEFAULT NEXTVAL ('kv_seq'),
    cat varchar(45) NOT NULL,
    k varchar(128) NOT NULL,
    v TEXT,
    version bigint default 1,
    create_time timestamp(0) with time zone DEFAULT NULL,
    update_time timestamp(0) with time zone DEFAULT  NULL,
    PRIMARY KEY (id),
    CONSTRAINT uniq_k UNIQUE(cat, k)
);
