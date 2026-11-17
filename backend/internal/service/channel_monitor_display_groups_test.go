//go:build unit

package service

import (
	"context"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

type monitorDisplayRepoStub struct {
	ChannelMonitorRepository
	groups        map[int64]*ChannelMonitorDisplayGroup
	monitor       *ChannelMonitor
	published     []*ChannelMonitor
	created       *ChannelMonitor
	aggregatedIDs []int64
}

func (r *monitorDisplayRepoStub) GetDisplayGroup(_ context.Context, id int64) (*ChannelMonitorDisplayGroup, error) {
	if group := r.groups[id]; group != nil {
		return group, nil
	}
	return nil, ErrMonitorDisplayGroupNotFound
}

func (r *monitorDisplayRepoStub) CreateDisplayGroup(_ context.Context, group *ChannelMonitorDisplayGroup) error {
	group.ID = 1
	return nil
}

func (r *monitorDisplayRepoStub) UpdateDisplayGroup(_ context.Context, id int64, p ChannelMonitorDisplayGroupUpdateParams) (*ChannelMonitorDisplayGroup, error) {
	group := *r.groups[id]
	if p.Name != nil {
		group.Name = *p.Name
	}
	if p.SortOrder != nil {
		group.SortOrder = *p.SortOrder
	}
	return &group, nil
}

func (r *monitorDisplayRepoStub) UpdateDisplay(_ context.Context, id int64, p ChannelMonitorDisplayUpdateParams) (*ChannelMonitor, error) {
	if r.monitor == nil || r.monitor.ID != id {
		return nil, ErrChannelMonitorNotFound
	}
	if p.SetDisplayGroup {
		r.monitor.DisplayGroupID = cloneInt64Pointer(p.DisplayGroupID)
	}
	if p.SortOrder != nil {
		r.monitor.SortOrder = *p.SortOrder
	}
	clone := *r.monitor
	return &clone, nil
}

func (r *monitorDisplayRepoStub) Create(_ context.Context, m *ChannelMonitor) error {
	clone := *m
	r.created = &clone
	m.ID = 1
	return nil
}

func (r *monitorDisplayRepoStub) ListPublished(context.Context) ([]*ChannelMonitor, error) {
	return r.published, nil
}
func (r *monitorDisplayRepoStub) GetPublishedByID(_ context.Context, id int64) (*ChannelMonitor, error) {
	for _, m := range r.published {
		if m.ID == id {
			return m, nil
		}
	}
	return nil, ErrChannelMonitorNotFound
}
func (r *monitorDisplayRepoStub) ListEnabled(context.Context) ([]*ChannelMonitor, error) {
	return []*ChannelMonitor{r.monitor}, nil
}
func (r *monitorDisplayRepoStub) ListLatestForMonitorIDs(_ context.Context, ids []int64) (map[int64][]*ChannelMonitorLatest, error) {
	r.aggregatedIDs = append([]int64(nil), ids...)
	return map[int64][]*ChannelMonitorLatest{}, nil
}
func (r *monitorDisplayRepoStub) ComputeAvailabilityForMonitors(context.Context, []int64, int) (map[int64][]*ChannelMonitorAvailability, error) {
	return map[int64][]*ChannelMonitorAvailability{}, nil
}
func (r *monitorDisplayRepoStub) ListRecentHistoryForMonitors(context.Context, []int64, map[int64]string, int) (map[int64][]*ChannelMonitorHistoryEntry, error) {
	return map[int64][]*ChannelMonitorHistoryEntry{}, nil
}
func (r *monitorDisplayRepoStub) ListLatestPerModel(context.Context, int64) ([]*ChannelMonitorLatest, error) {
	return nil, nil
}
func (r *monitorDisplayRepoStub) ComputeAvailability(context.Context, int64, int) ([]*ChannelMonitorAvailability, error) {
	return nil, nil
}

type monitorDisplayScheduler struct{ calls int }

func (s *monitorDisplayScheduler) Schedule(*ChannelMonitor) { s.calls++ }
func (s *monitorDisplayScheduler) Unschedule(int64)         { s.calls++ }

func TestMonitorDisplayGroupNameAndSortValidation(t *testing.T) {
	repo := &monitorDisplayRepoStub{groups: map[int64]*ChannelMonitorDisplayGroup{1: {ID: 1, Name: "old", SortOrder: 10}}}
	svc := NewChannelMonitorService(repo, nil)
	ctx := context.Background()
	for _, name := range []string{"", " \n\t ", strings.Repeat("界", 101)} {
		_, err := svc.CreateDisplayGroup(ctx, name, 0)
		require.ErrorIs(t, err, ErrMonitorDisplayGroupName)
	}
	for _, order := range []int{-1, 2147483648} {
		_, err := svc.CreateDisplayGroup(ctx, "valid", order)
		require.ErrorIs(t, err, ErrMonitorDisplaySortOrder)
		_, err = svc.UpdateDisplayGroup(ctx, 1, ChannelMonitorDisplayGroupUpdateParams{SortOrder: &order})
		require.ErrorIs(t, err, ErrMonitorDisplaySortOrder)
	}
	group, err := svc.CreateDisplayGroup(ctx, "  "+strings.Repeat("界", 100)+"  ", 2147483647)
	require.NoError(t, err)
	require.Equal(t, strings.Repeat("界", 100), group.Name)
	name := "  renamed  "
	group, err = svc.UpdateDisplayGroup(ctx, 1, ChannelMonitorDisplayGroupUpdateParams{Name: &name})
	require.NoError(t, err)
	require.Equal(t, "renamed", group.Name)
	require.Equal(t, 10, group.SortOrder)
	_, err = svc.UpdateDisplayGroup(ctx, 1, ChannelMonitorDisplayGroupUpdateParams{})
	require.ErrorIs(t, err, ErrMonitorDisplayEmptyUpdate)
}

func TestChannelMonitorDisplayUpdatePreservesSecretsAndScheduling(t *testing.T) {
	groupID := int64(9)
	repo := &monitorDisplayRepoStub{
		groups:  map[int64]*ChannelMonitorDisplayGroup{9: {ID: 9}},
		monitor: &ChannelMonitor{ID: 1, GroupName: "legacy", APIKey: "OLD:secret", Enabled: true, IntervalSeconds: 60, SortOrder: 20},
	}
	svc := NewChannelMonitorService(repo, &duplicateChannelMonitorEncryptor{})
	scheduler := &monitorDisplayScheduler{}
	svc.SetScheduler(scheduler)
	ctx := context.Background()
	_, err := svc.UpdateDisplay(ctx, 1, ChannelMonitorDisplayUpdateParams{})
	require.ErrorIs(t, err, ErrMonitorDisplayEmptyUpdate)
	for _, id := range []int64{0, -1, 123} {
		_, err = svc.UpdateDisplay(ctx, 1, ChannelMonitorDisplayUpdateParams{SetDisplayGroup: true, DisplayGroupID: &id})
		require.Error(t, err)
		require.Nil(t, repo.monitor.DisplayGroupID)
	}
	m, err := svc.UpdateDisplay(ctx, 1, ChannelMonitorDisplayUpdateParams{SetDisplayGroup: true, DisplayGroupID: &groupID})
	require.NoError(t, err)
	require.Equal(t, &groupID, m.DisplayGroupID)
	require.Equal(t, 20, m.SortOrder)
	order := 30
	m, err = svc.UpdateDisplay(ctx, 1, ChannelMonitorDisplayUpdateParams{SortOrder: &order})
	require.NoError(t, err)
	require.Equal(t, &groupID, m.DisplayGroupID)
	m, err = svc.UpdateDisplay(ctx, 1, ChannelMonitorDisplayUpdateParams{SetDisplayGroup: true})
	require.NoError(t, err)
	require.Nil(t, m.DisplayGroupID)
	require.Equal(t, 30, m.SortOrder)
	require.Equal(t, "legacy", m.GroupName)
	require.Equal(t, "secret", m.APIKey)
	require.Equal(t, "OLD:secret", repo.monitor.APIKey)
	require.True(t, m.Enabled)
	require.Equal(t, 60, m.IntervalSeconds)
	require.Zero(t, scheduler.calls)
}

func TestChannelMonitorCreateDisplayContextAndInvalidGroup(t *testing.T) {
	groupID := int64(9)
	repo := &monitorDisplayRepoStub{groups: map[int64]*ChannelMonitorDisplayGroup{9: {ID: 9}}}
	svc := NewChannelMonitorService(repo, &duplicateChannelMonitorEncryptor{})
	p := ChannelMonitorCreateParams{Name: "monitor", Provider: MonitorProviderOpenAI, Endpoint: "https://8.8.8.8", APIKey: "secret", PrimaryModel: "model", IntervalSeconds: 60, GroupName: "legacy", DisplayGroupID: &groupID, SortOrder: 12}
	m, err := svc.Create(context.Background(), p)
	require.NoError(t, err)
	require.Equal(t, &groupID, m.DisplayGroupID)
	require.Equal(t, 12, repo.created.SortOrder)
	require.Equal(t, "legacy", repo.created.GroupName)
	p.DisplayGroupID = nil
	m, err = svc.Create(context.Background(), p)
	require.NoError(t, err)
	require.Nil(t, m.DisplayGroupID)
	invalidID := int64(999)
	p.DisplayGroupID = &invalidID
	_, err = svc.Create(context.Background(), p)
	require.ErrorIs(t, err, ErrMonitorDisplayGroupNotFound)
}

func TestChannelMonitorPublishedVisibilityAndUnpublishedRunner(t *testing.T) {
	groupID := int64(9)
	repo := &monitorDisplayRepoStub{
		monitor:   &ChannelMonitor{ID: 99, GroupName: "formal", Enabled: true, APIKey: "OLD:secret"},
		published: []*ChannelMonitor{{ID: 1, GroupName: "wrong legacy", DisplayGroupID: &groupID, DisplayGroupName: "formal", DisplayGroupSortOrder: 5, SortOrder: 10, PrimaryModel: "model", Enabled: true}},
	}
	svc := NewChannelMonitorService(repo, &duplicateChannelMonitorEncryptor{})
	views, err := svc.ListUserView(context.Background())
	require.NoError(t, err)
	require.Len(t, views, 1)
	require.Equal(t, []int64{1}, repo.aggregatedIDs)
	require.Equal(t, "formal", views[0].DisplayGroupName)
	require.Equal(t, int64(9), views[0].DisplayGroupID)
	detail, err := svc.GetUserDetail(context.Background(), 1)
	require.NoError(t, err)
	require.Equal(t, "formal", detail.DisplayGroupName)
	for _, id := range []int64{2, 99, 999} {
		_, err = svc.GetUserDetail(context.Background(), id)
		require.ErrorIs(t, err, ErrChannelMonitorNotFound)
	}
	enabled, err := svc.ListEnabledMonitors(context.Background())
	require.NoError(t, err)
	require.Len(t, enabled, 1)
	require.Equal(t, int64(99), enabled[0].ID)
	require.Nil(t, enabled[0].DisplayGroupID)
	repo.published = nil
	views, err = svc.ListUserView(context.Background())
	require.NoError(t, err)
	require.NotNil(t, views)
	require.Empty(t, views)
}
