//go:build integration

package repository

import (
	"context"
	"database/sql"
	"net/url"
	"strings"
	"testing"
	"testing/fstest"
	"time"

	"github.com/Wei-Shaw/sub2api/migrations"
	"github.com/google/uuid"
	"github.com/lib/pq"
	"github.com/stretchr/testify/require"
)

// 在真实旧结构上应用实际 migration runner，逐行比较旧列而非检查 SQL 文本。
func TestChannelMonitorDisplayGroupUpgradePreservesLegacyData(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()
	dsn, err := url.Parse(integrationPostgresDSN)
	require.NoError(t, err)
	name := "monitor_display_upgrade_" + strings.ReplaceAll(uuid.NewString(), "-", "")
	quoted := pq.QuoteIdentifier(name)
	_, err = integrationDB.ExecContext(ctx, "CREATE DATABASE "+quoted)
	require.NoError(t, err)
	t.Cleanup(func() {
		cleanup, stop := context.WithTimeout(context.Background(), 15*time.Second)
		defer stop()
		_, e := integrationDB.ExecContext(cleanup, "DROP DATABASE "+quoted)
		require.NoError(t, e)
	})
	dsn.Path = "/" + name
	db, err := sql.Open("postgres", dsn.String())
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, db.Close()) })

	legacy := fstest.MapFS{}
	entries, err := migrations.FS.ReadDir(".")
	require.NoError(t, err)
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".sql") || entry.Name() >= "383_" {
			continue
		}
		raw, e := migrations.FS.ReadFile(entry.Name())
		require.NoError(t, e)
		legacy[entry.Name()] = &fstest.MapFile{Data: raw, Mode: 0444}
	}
	require.NoError(t, applyMigrationsFS(ctx, db, legacy))

	var accountID, templateID int64
	require.NoError(t, db.QueryRowContext(ctx, `INSERT INTO accounts(name,platform,type,credentials,extra) VALUES('synthetic monitor account','openai','apikey','{"api_key":"synthetic-not-used"}','{}') RETURNING id`).Scan(&accountID))
	require.NoError(t, db.QueryRowContext(ctx, `INSERT INTO channel_monitor_request_templates(name,provider,extra_headers,body_override_mode,body_override) VALUES('synthetic monitor template','openai','{"X-Synthetic":"legacy"}','merge','{"temperature":0}') RETURNING id`).Scan(&templateID))
	ids := make([]int64, 0, 3)
	for i, mode := range []string{"probe", "quota", "quota_probe"} {
		label := []string{"历史标签 A", "", "历史标签 B"}[i]
		var id int64
		require.NoError(t, db.QueryRowContext(ctx, `INSERT INTO channel_monitors(name,provider,endpoint,api_key_encrypted,primary_model,extra_models,group_name,enabled,interval_seconds,jitter_seconds,created_by,check_mode,account_id,template_id,extra_headers,body_override_mode,body_override,last_checked_at)
			VALUES($1,'openai','https://synthetic.invalid','synthetic-opaque-encrypted-key','synthetic-model','["synthetic-extra"]',$2,$3,300,15,1,$4,$5,$6,'{"X-Synthetic":"snapshot"}','merge','{"temperature":0.2}',NOW()-INTERVAL '2 hours') RETURNING id`, mode, label, i != 1, mode, accountID, templateID).Scan(&id))
		ids = append(ids, id)
		_, err = db.ExecContext(ctx, `INSERT INTO channel_monitor_histories(monitor_id,model,status,latency_ms,ping_latency_ms,message,checked_at,quota) VALUES($1,'synthetic-model','operational',123,12,'synthetic history',NOW()-INTERVAL '1 hour','{"remaining":37}')`, id)
		require.NoError(t, err)
		_, err = db.ExecContext(ctx, `INSERT INTO channel_monitor_daily_rollups(monitor_id,model,bucket_date,total_checks,ok_count,operational_count,sum_latency_ms,count_latency) VALUES($1,'synthetic-model',CURRENT_DATE-10,7,6,6,738,6)`, id)
		require.NoError(t, err)
	}

	snapshot := func(query string, args ...any) string {
		t.Helper()
		var value string
		require.NoError(t, db.QueryRowContext(ctx, query, args...).Scan(&value))
		return value
	}
	// 升级前冻结各表旧列，之后只比较这些列：
	// 后续迁移给表新增列（如 accounts.domestic_route_probe_at）不算改写旧数据。
	legacyRows := map[string]func() string{}
	for _, table := range []string{"channel_monitors", "channel_monitor_histories", "channel_monitor_daily_rollups", "accounts", "channel_monitor_request_templates"} {
		columns := pq.StringArray{}
		require.NoError(t, db.QueryRowContext(ctx, `SELECT array_agg(column_name::text ORDER BY ordinal_position) FROM information_schema.columns WHERE table_schema='public' AND table_name=$1`, table).Scan(&columns))
		legacyRows[table] = func() string {
			t.Helper()
			return snapshot(`SELECT jsonb_agg((SELECT jsonb_object_agg(c,to_jsonb(t)->c) FROM unnest($1::text[]) c) ORDER BY id)::text FROM `+pq.QuoteIdentifier(table)+` t`, columns)
		}
	}
	before := make(map[string]string, len(legacyRows))
	for table, rows := range legacyRows {
		before[table] = rows()
	}
	columnsBefore := snapshot(`SELECT jsonb_agg(jsonb_build_array(column_name,data_type,is_nullable,column_default) ORDER BY ordinal_position)::text FROM information_schema.columns WHERE table_schema='public' AND table_name='channel_monitors'`)
	indexesBefore := snapshot(`SELECT jsonb_agg(indexdef ORDER BY indexname)::text FROM pg_indexes WHERE schemaname='public' AND tablename='channel_monitors'`)
	require.NoError(t, ApplyMigrations(ctx, db))
	var count int
	require.NoError(t, db.QueryRowContext(ctx, `SELECT COUNT(*) FROM channel_monitor_display_groups`).Scan(&count))
	require.Zero(t, count, "不从旧标签自动建正式组")
	require.NoError(t, db.QueryRowContext(ctx, `SELECT COUNT(*) FROM channel_monitors WHERE display_group_id IS NULL AND sort_order=0`).Scan(&count))
	require.Equal(t, len(ids), count)
	require.Equal(t, columnsBefore, snapshot(`SELECT jsonb_agg(jsonb_build_array(column_name,data_type,is_nullable,column_default) ORDER BY ordinal_position)::text FROM information_schema.columns WHERE table_schema='public' AND table_name='channel_monitors' AND column_name NOT IN ('display_group_id','sort_order')`))
	require.Equal(t, indexesBefore, snapshot(`SELECT jsonb_agg(indexdef ORDER BY indexname)::text FROM pg_indexes WHERE schemaname='public' AND tablename='channel_monitors' AND indexname <> 'idx_channel_monitors_display_group_sort'`))
	for table, rows := range legacyRows {
		require.JSONEq(t, before[table], rows(), "迁移改写了 %s 的旧数据", table)
	}
	var groupID int64
	require.NoError(t, db.QueryRowContext(ctx, `INSERT INTO channel_monitor_display_groups(name,sort_order) VALUES('正式组',20) RETURNING id`).Scan(&groupID))
	_, err = db.ExecContext(ctx, `UPDATE channel_monitors SET display_group_id=$1,sort_order=30 WHERE id=$2`, groupID, ids[0])
	require.NoError(t, err)
	for range 2 {
		require.NoError(t, ApplyMigrations(ctx, db))
		var loadedGroup int64
		var order int
		require.NoError(t, db.QueryRowContext(ctx, `SELECT display_group_id,sort_order FROM channel_monitors WHERE id=$1`, ids[0]).Scan(&loadedGroup, &order))
		require.Equal(t, groupID, loadedGroup)
		require.Equal(t, 30, order)
		for table, rows := range legacyRows {
			require.JSONEq(t, before[table], rows(), "迁移改写了 %s 的旧数据", table)
		}
	}
}
