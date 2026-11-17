package repository

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	dbent "github.com/Wei-Shaw/sub2api/ent"
	"github.com/Wei-Shaw/sub2api/ent/channelmonitordisplaygroup"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/lib/pq"
)

func translateMonitorDisplayWriteError(err error) error {
	var pg *pq.Error
	if errors.As(err, &pg) && pg.Code == "23503" {
		return service.ErrMonitorDisplayConflict.WithCause(err)
	}
	return translatePersistenceError(err, service.ErrChannelMonitorNotFound, nil)
}

func displayGroupModel(row *dbent.ChannelMonitorDisplayGroup) *service.ChannelMonitorDisplayGroup {
	return &service.ChannelMonitorDisplayGroup{
		ID: row.ID, Name: row.Name, SortOrder: row.SortOrder,
		CreatedAt: row.CreatedAt, UpdatedAt: row.UpdatedAt,
	}
}

func (r *channelMonitorRepository) ListDisplayGroups(ctx context.Context) ([]*service.ChannelMonitorDisplayGroup, error) {
	rows, err := clientFromContext(ctx, r.client).ChannelMonitorDisplayGroup.Query().
		Order(dbent.Asc(channelmonitordisplaygroup.FieldSortOrder), dbent.Asc(channelmonitordisplaygroup.FieldID)).All(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]*service.ChannelMonitorDisplayGroup, 0, len(rows))
	for _, row := range rows {
		out = append(out, displayGroupModel(row))
	}
	return out, nil
}

func (r *channelMonitorRepository) GetDisplayGroup(ctx context.Context, id int64) (*service.ChannelMonitorDisplayGroup, error) {
	row, err := clientFromContext(ctx, r.client).ChannelMonitorDisplayGroup.Get(ctx, id)
	if err != nil {
		return nil, translatePersistenceError(err, service.ErrMonitorDisplayGroupNotFound, nil)
	}
	return displayGroupModel(row), nil
}

func (r *channelMonitorRepository) CreateDisplayGroup(ctx context.Context, group *service.ChannelMonitorDisplayGroup) error {
	row, err := clientFromContext(ctx, r.client).ChannelMonitorDisplayGroup.Create().
		SetName(group.Name).SetSortOrder(group.SortOrder).Save(ctx)
	if err != nil {
		return translatePersistenceError(err, nil, service.ErrMonitorDisplayGroupConflict)
	}
	*group = *displayGroupModel(row)
	return nil
}

func (r *channelMonitorRepository) UpdateDisplayGroup(ctx context.Context, id int64, p service.ChannelMonitorDisplayGroupUpdateParams) (*service.ChannelMonitorDisplayGroup, error) {
	update := clientFromContext(ctx, r.client).ChannelMonitorDisplayGroup.UpdateOneID(id)
	if p.Name != nil {
		update.SetName(*p.Name)
	}
	if p.SortOrder != nil {
		update.SetSortOrder(*p.SortOrder)
	}
	row, err := update.Save(ctx)
	if err != nil {
		return nil, translatePersistenceError(err, service.ErrMonitorDisplayGroupNotFound, service.ErrMonitorDisplayGroupConflict)
	}
	return displayGroupModel(row), nil
}

func (r *channelMonitorRepository) DeleteDisplayGroup(ctx context.Context, id int64) error {
	// RESTRICT is the atomic emptiness check, including disabled monitors and
	// concurrent creates/moves. No application-side check-then-delete race.
	err := clientFromContext(ctx, r.client).ChannelMonitorDisplayGroup.DeleteOneID(id).Exec(ctx)
	var pg *pq.Error
	// PostgreSQL 的 ON DELETE RESTRICT 返回 restrict_violation (23001)。
	if errors.As(err, &pg) && (pg.Code == "23001" || pg.Code == "23503") {
		return service.ErrMonitorDisplayGroupConflict.WithCause(err)
	}
	return translatePersistenceError(err, service.ErrMonitorDisplayGroupNotFound, nil)
}

func (r *channelMonitorRepository) UpdateDisplay(ctx context.Context, id int64, p service.ChannelMonitorDisplayUpdateParams) (*service.ChannelMonitor, error) {
	update := clientFromContext(ctx, r.client).ChannelMonitor.UpdateOneID(id)
	if p.SetDisplayGroup {
		if p.DisplayGroupID == nil {
			update.ClearDisplayGroupID()
		} else {
			update.SetDisplayGroupID(*p.DisplayGroupID)
		}
	}
	if p.SortOrder != nil {
		update.SetSortOrder(*p.SortOrder)
	}
	row, err := update.Save(ctx)
	if err != nil {
		return nil, translateMonitorDisplayWriteError(err)
	}
	return entToServiceMonitor(row), nil
}

const publishedMonitorQuery = `SELECT m.id, m.name, m.provider, m.group_name,
 m.primary_model, m.extra_models, m.display_group_id, m.sort_order, g.name, g.sort_order
 FROM channel_monitors m JOIN channel_monitor_display_groups g ON g.id = m.display_group_id
 WHERE m.enabled = TRUE`

func (r *channelMonitorRepository) queryPublished(ctx context.Context, id *int64) ([]*service.ChannelMonitor, error) {
	query := publishedMonitorQuery
	var args []any
	if id != nil {
		query += " AND m.id = $1"
		args = []any{*id}
	}
	query += " ORDER BY g.sort_order ASC, g.id ASC, m.sort_order ASC, m.id ASC"
	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("query published monitors: %w", err)
	}
	defer rows.Close()
	out := make([]*service.ChannelMonitor, 0)
	for rows.Next() {
		m := &service.ChannelMonitor{Enabled: true}
		var groupID int64
		var extras []byte
		if err := rows.Scan(&m.ID, &m.Name, &m.Provider, &m.GroupName, &m.PrimaryModel,
			&extras, &groupID, &m.SortOrder, &m.DisplayGroupName, &m.DisplayGroupSortOrder); err != nil {
			return nil, err
		}
		m.DisplayGroupID = &groupID
		if len(extras) > 0 {
			if err := json.Unmarshal(extras, &m.ExtraModels); err != nil {
				return nil, fmt.Errorf("decode published monitor models: %w", err)
			}
		}
		out = append(out, m)
	}
	return out, rows.Err()
}

func (r *channelMonitorRepository) ListPublished(ctx context.Context) ([]*service.ChannelMonitor, error) {
	return r.queryPublished(ctx, nil)
}

func (r *channelMonitorRepository) GetPublishedByID(ctx context.Context, id int64) (*service.ChannelMonitor, error) {
	items, err := r.queryPublished(ctx, &id)
	if err != nil {
		return nil, err
	}
	if len(items) == 0 {
		return nil, service.ErrChannelMonitorNotFound
	}
	return items[0], nil
}
