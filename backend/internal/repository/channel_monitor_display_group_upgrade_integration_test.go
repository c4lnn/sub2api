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

	snapshot := func(query string) string {
		t.Helper()
		var value string
		require.NoError(t, db.QueryRowContext(ctx, query).Scan(&value))
		return value
	}
	queries := []string{
		`SELECT jsonb_agg(to_jsonb(m) - 'display_group_id' - 'sort_order' ORDER BY id)::text FROM channel_monitors m`,
		`SELECT jsonb_agg(to_jsonb(h) ORDER BY id)::text FROM channel_monitor_histories h`,
		`SELECT jsonb_agg(to_jsonb(r) ORDER BY id)::text FROM channel_monitor_daily_rollups r`,
		`SELECT jsonb_agg(to_jsonb(a) ORDER BY id)::text FROM accounts a`,
		`SELECT jsonb_agg(to_jsonb(t) ORDER BY id)::text FROM channel_monitor_request_templates t`,
	}
	before := make([]string, len(queries))
	for i, query := range queries {
		before[i] = snapshot(query)
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
	for i, query := range queries {
		require.JSONEq(t, before[i], snapshot(query))
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
		for i, query := range queries {
			require.JSONEq(t, before[i], snapshot(query))
		}
	}
}
