package service

import (
	"context"
	"strings"
	"time"
	"unicode/utf8"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
)

var (
	ErrMonitorDisplayGroupNotFound = infraerrors.NotFound("MONITOR_DISPLAY_GROUP_NOT_FOUND", "monitor display group not found")
	ErrMonitorDisplayGroupConflict = infraerrors.Conflict("MONITOR_DISPLAY_GROUP_CONFLICT", "display group name already exists or group is in use")
	ErrMonitorDisplayConflict      = infraerrors.Conflict("MONITOR_DISPLAY_CONFLICT", "display group changed concurrently; reload and retry")
	ErrMonitorDisplayGroupName     = infraerrors.BadRequest("MONITOR_DISPLAY_GROUP_INVALID_NAME", "name must contain 1 to 100 characters after trimming")
	ErrMonitorDisplayGroupID       = infraerrors.BadRequest("MONITOR_DISPLAY_GROUP_INVALID_ID", "display_group_id must be a positive integer or null")
	ErrMonitorDisplaySortOrder     = infraerrors.BadRequest("MONITOR_DISPLAY_INVALID_SORT_ORDER", "sort_order must be an integer in [0, 2147483647]")
	ErrMonitorDisplayEmptyUpdate   = infraerrors.BadRequest("MONITOR_DISPLAY_EMPTY_UPDATE", "provide at least one display field")
)

type ChannelMonitorDisplayGroup struct {
	ID        int64     `json:"id"`
	Name      string    `json:"name"`
	SortOrder int       `json:"sort_order"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type ChannelMonitorDisplayGroupUpdateParams struct {
	Name      *string
	SortOrder *int
}

// SetDisplayGroup distinguishes an omitted group from an explicit null.
type ChannelMonitorDisplayUpdateParams struct {
	SetDisplayGroup bool
	DisplayGroupID  *int64
	SortOrder       *int
}

func validateMonitorSortOrder(value int) error {
	if value < 0 || int64(value) > 2147483647 {
		return ErrMonitorDisplaySortOrder
	}
	return nil
}

func normalizeMonitorDisplayGroupName(value string) (string, error) {
	value = strings.TrimSpace(value)
	if !utf8.ValidString(value) || utf8.RuneCountInString(value) < 1 || utf8.RuneCountInString(value) > 100 {
		return "", ErrMonitorDisplayGroupName
	}
	return value, nil
}

func (s *ChannelMonitorService) validateDisplayGroup(ctx context.Context, id *int64) error {
	if id == nil {
		return nil
	}
	if *id <= 0 {
		return ErrMonitorDisplayGroupID
	}
	_, err := s.repo.GetDisplayGroup(ctx, *id)
	return err
}

func (s *ChannelMonitorService) ListDisplayGroups(ctx context.Context) ([]*ChannelMonitorDisplayGroup, error) {
	return s.repo.ListDisplayGroups(ctx)
}

func (s *ChannelMonitorService) CreateDisplayGroup(ctx context.Context, name string, sortOrder int) (*ChannelMonitorDisplayGroup, error) {
	name, err := normalizeMonitorDisplayGroupName(name)
	if err != nil {
		return nil, err
	}
	if err := validateMonitorSortOrder(sortOrder); err != nil {
		return nil, err
	}
	group := &ChannelMonitorDisplayGroup{Name: name, SortOrder: sortOrder}
	if err := s.repo.CreateDisplayGroup(ctx, group); err != nil {
		return nil, err
	}
	return group, nil
}

func (s *ChannelMonitorService) UpdateDisplayGroup(ctx context.Context, id int64, p ChannelMonitorDisplayGroupUpdateParams) (*ChannelMonitorDisplayGroup, error) {
	if p.Name == nil && p.SortOrder == nil {
		return nil, ErrMonitorDisplayEmptyUpdate
	}
	if p.Name != nil {
		name, err := normalizeMonitorDisplayGroupName(*p.Name)
		if err != nil {
			return nil, err
		}
		p.Name = &name
	}
	if p.SortOrder != nil {
		if err := validateMonitorSortOrder(*p.SortOrder); err != nil {
			return nil, err
		}
	}
	return s.repo.UpdateDisplayGroup(ctx, id, p)
}

func (s *ChannelMonitorService) DeleteDisplayGroup(ctx context.Context, id int64) error {
	return s.repo.DeleteDisplayGroup(ctx, id)
}

// UpdateDisplay does not load or rewrite credentials and never reschedules probes.
func (s *ChannelMonitorService) UpdateDisplay(ctx context.Context, id int64, p ChannelMonitorDisplayUpdateParams) (*ChannelMonitor, error) {
	if !p.SetDisplayGroup && p.SortOrder == nil {
		return nil, ErrMonitorDisplayEmptyUpdate
	}
	if p.SetDisplayGroup {
		if err := s.validateDisplayGroup(ctx, p.DisplayGroupID); err != nil {
			return nil, err
		}
	}
	if p.SortOrder != nil {
		if err := validateMonitorSortOrder(*p.SortOrder); err != nil {
			return nil, err
		}
	}
	m, err := s.repo.UpdateDisplay(ctx, id, p)
	if err != nil {
		return nil, err
	}
	s.decryptInPlace(m)
	return m, nil
}
