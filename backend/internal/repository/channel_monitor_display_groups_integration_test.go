//go:build integration

package repository

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/stretchr/testify/require"
)

func createMonitorDisplayGroup(t *testing.T, name string, order int) *service.ChannelMonitorDisplayGroup {
	t.Helper()
	repo := NewChannelMonitorRepository(integrationEntClient, integrationDB)
	group := &service.ChannelMonitorDisplayGroup{Name: fmt.Sprintf("%s-%d", name, time.Now().UnixNano()), SortOrder: order}
	require.NoError(t, repo.CreateDisplayGroup(context.Background(), group))
	t.Cleanup(func() { _ = repo.DeleteDisplayGroup(context.Background(), group.ID) })
	return group
}

func createDisplayMonitor(t *testing.T, groupID *int64, order int, enabled bool) *service.ChannelMonitor {
	t.Helper()
	repo := NewChannelMonitorRepository(integrationEntClient, integrationDB)
	m := &service.ChannelMonitor{
		Name:     fmt.Sprintf("display-monitor-%d", time.Now().UnixNano()),
		Provider: service.MonitorProviderOpenAI, PrimaryModel: "model",
		Endpoint: "https://8.8.8.8", APIKey: "ciphertext-unchanged", GroupName: "legacy-shared",
		ExtraModels: []string{"extra"}, Enabled: enabled, IntervalSeconds: 60, JitterSeconds: 5,
		DisplayGroupID: groupID, SortOrder: order,
		ExtraHeaders:     map[string]string{"X-Custom": "original"},
		BodyOverrideMode: service.MonitorBodyOverrideModeMerge,
		BodyOverride:     map[string]any{"temperature": 0.5},
	}
	require.NoError(t, repo.Create(context.Background(), m))
	t.Cleanup(func() { _ = repo.Delete(context.Background(), m.ID) })
	return m
}

func TestChannelMonitorDisplayGroupCRUDAndExactUniqueness(t *testing.T) {
	ctx := context.Background()
	repo := NewChannelMonitorRepository(integrationEntClient, integrationDB)
	a := createMonitorDisplayGroup(t, "CaseSensitive", 20)
	b := createMonitorDisplayGroup(t, "early", 10)
	c := createMonitorDisplayGroup(t, "same-order", 10)
	groups, err := repo.ListDisplayGroups(ctx)
	require.NoError(t, err)
	positions := map[int64]int{}
	for i, group := range groups {
		positions[group.ID] = i
	}
	require.Less(t, positions[b.ID], positions[c.ID])
	require.Less(t, positions[c.ID], positions[a.ID])
	duplicate := &service.ChannelMonitorDisplayGroup{Name: a.Name}
	require.ErrorIs(t, repo.CreateDisplayGroup(ctx, duplicate), service.ErrMonitorDisplayGroupConflict)
	lower := &service.ChannelMonitorDisplayGroup{Name: strings.ToLower(a.Name)}
	require.NoError(t, repo.CreateDisplayGroup(ctx, lower))
	t.Cleanup(func() { _ = repo.DeleteDisplayGroup(ctx, lower.ID) })
	name := b.Name
	_, err = repo.UpdateDisplayGroup(ctx, a.ID, service.ChannelMonitorDisplayGroupUpdateParams{Name: &name})
	require.ErrorIs(t, err, service.ErrMonitorDisplayGroupConflict)
	order := 2147483647
	updated, err := repo.UpdateDisplayGroup(ctx, a.ID, service.ChannelMonitorDisplayGroupUpdateParams{SortOrder: &order})
	require.NoError(t, err)
	require.Equal(t, a.Name, updated.Name)
	require.Equal(t, order, updated.SortOrder)
	require.NoError(t, repo.DeleteDisplayGroup(ctx, a.ID))
	require.ErrorIs(t, repo.DeleteDisplayGroup(ctx, a.ID), service.ErrMonitorDisplayGroupNotFound)
	_, err = repo.GetDisplayGroup(ctx, a.ID)
	require.ErrorIs(t, err, service.ErrMonitorDisplayGroupNotFound)
}

func TestChannelMonitorDisplayGroupPublishedOrderingAndFilterPagination(t *testing.T) {
	ctx := context.Background()
	repo := NewChannelMonitorRepository(integrationEntClient, integrationDB)
	a := createMonitorDisplayGroup(t, "formal-a", 20)
	b := createMonitorDisplayGroup(t, "formal-b", 10)
	c := createMonitorDisplayGroup(t, "formal-c", 10)
	ma := createDisplayMonitor(t, &a.ID, 0, true)
	mb1 := createDisplayMonitor(t, &b.ID, 30, true)
	mb2 := createDisplayMonitor(t, &b.ID, 10, true)
	mb3 := createDisplayMonitor(t, &b.ID, 10, true)
	mc := createDisplayMonitor(t, &c.ID, 0, true)
	disabled := createDisplayMonitor(t, &b.ID, 0, false)
	unpublished := createDisplayMonitor(t, nil, 0, true)
	published, err := repo.ListPublished(ctx)
	require.NoError(t, err)
	wanted := map[int64]bool{ma.ID: true, mb1.ID: true, mb2.ID: true, mb3.ID: true, mc.ID: true, disabled.ID: true, unpublished.ID: true}
	var ids []int64
	for _, m := range published {
		if wanted[m.ID] {
			ids = append(ids, m.ID)
			require.NotNil(t, m.DisplayGroupID)
			require.NotEmpty(t, m.DisplayGroupName)
			require.Empty(t, m.APIKey, "published query must not load secrets")
		}
	}
	require.Equal(t, []int64{mb2.ID, mb3.ID, mb1.ID, mc.ID, ma.ID}, ids)
	detail, err := repo.GetPublishedByID(ctx, mb2.ID)
	require.NoError(t, err)
	require.Equal(t, b.Name, detail.DisplayGroupName)
	require.Equal(t, 10, detail.DisplayGroupSortOrder)
	for _, id := range []int64{disabled.ID, unpublished.ID, 9223372036854775807} {
		_, err = repo.GetPublishedByID(ctx, id)
		require.ErrorIs(t, err, service.ErrChannelMonitorNotFound)
	}
	enabled := true
	params := service.ChannelMonitorListParams{Page: 1, PageSize: 1, DisplayGroupID: &b.ID, Enabled: &enabled, Provider: service.MonitorProviderOpenAI, Search: "legacy-shared"}
	items, total, err := repo.List(ctx, params)
	require.NoError(t, err)
	require.EqualValues(t, 3, total)
	require.Len(t, items, 1)
	require.Equal(t, mb2.ID, items[0].ID)
	params.Page = 2
	items, total, err = repo.List(ctx, params)
	require.NoError(t, err)
	require.EqualValues(t, 3, total)
	require.Equal(t, mb3.ID, items[0].ID)
	zero := int64(0)
	items, _, err = repo.List(ctx, service.ChannelMonitorListParams{DisplayGroupID: &zero, Search: unpublished.Name})
	require.NoError(t, err)
	require.Len(t, items, 1)
	require.Equal(t, unpublished.ID, items[0].ID)
	items, _, err = repo.List(ctx, service.ChannelMonitorListParams{Search: unpublished.Name})
	require.NoError(t, err)
	require.Len(t, items, 1, "omitted group retains admin all-monitor access")
	allEnabled, err := repo.ListEnabled(ctx)
	require.NoError(t, err)
	found := false
	for _, m := range allEnabled {
		if m.ID == unpublished.ID {
			found = true
		}
	}
	require.True(t, found, "runner still loads unpublished enabled monitors")
}

func TestChannelMonitorDisplayGroupNarrowWriteAndStaleEdit(t *testing.T) {
	ctx := context.Background()
	repo := NewChannelMonitorRepository(integrationEntClient, integrationDB)
	a := createMonitorDisplayGroup(t, "source", 0)
	b := createMonitorDisplayGroup(t, "target", 0)
	m := createDisplayMonitor(t, &a.ID, 10, true)
	checkedAt := time.Now().UTC().Truncate(time.Microsecond)
	require.NoError(t, repo.MarkChecked(ctx, m.ID, checkedAt))
	require.NoError(t, repo.InsertHistoryBatch(ctx, []*service.ChannelMonitorHistoryRow{{MonitorID: m.ID, Model: m.PrimaryModel, Status: service.MonitorStatusOperational, CheckedAt: checkedAt}}))
	before, err := repo.GetByID(ctx, m.ID)
	require.NoError(t, err)
	stale := *before
	order := 99
	moved, err := repo.UpdateDisplay(ctx, m.ID, service.ChannelMonitorDisplayUpdateParams{SetDisplayGroup: true, DisplayGroupID: &b.ID, SortOrder: &order})
	require.NoError(t, err)
	want := *before
	want.DisplayGroupID = &b.ID
	want.SortOrder = order
	want.UpdatedAt = moved.UpdatedAt
	require.Equal(t, &want, moved, "display writes preserve every non-display field")
	stale.Name = "edited without display fields"
	require.NoError(t, repo.Update(ctx, &stale))
	after, err := repo.GetByID(ctx, m.ID)
	require.NoError(t, err)
	require.Equal(t, &b.ID, after.DisplayGroupID)
	require.Equal(t, 99, after.SortOrder)
	require.Equal(t, "legacy-shared", after.GroupName)
	require.Equal(t, before.APIKey, after.APIKey)
	moved, err = repo.UpdateDisplay(ctx, m.ID, service.ChannelMonitorDisplayUpdateParams{SetDisplayGroup: true})
	require.NoError(t, err)
	require.Nil(t, moved.DisplayGroupID)
	require.Equal(t, 99, moved.SortOrder)
	history, err := repo.ListHistory(ctx, m.ID, "", 10)
	require.NoError(t, err)
	require.Len(t, history, 1)
	require.Equal(t, checkedAt, history[0].CheckedAt.UTC())
	invalid := int64(9223372036854775807)
	_, err = repo.UpdateDisplay(ctx, m.ID, service.ChannelMonitorDisplayUpdateParams{SetDisplayGroup: true, DisplayGroupID: &invalid})
	require.ErrorIs(t, err, service.ErrMonitorDisplayConflict)
	m.DisplayGroupID = &invalid
	require.ErrorIs(t, repo.Create(ctx, m), service.ErrMonitorDisplayConflict)
}

func TestChannelMonitorDisplayGroupDeleteDisabledAndConcurrentMove(t *testing.T) {
	ctx := context.Background()
	repo := NewChannelMonitorRepository(integrationEntClient, integrationDB)
	group := createMonitorDisplayGroup(t, "disabled-blocks", 0)
	m := createDisplayMonitor(t, &group.ID, 0, false)
	require.ErrorIs(t, repo.DeleteDisplayGroup(ctx, group.ID), service.ErrMonitorDisplayGroupConflict)
	_, err := repo.GetByID(ctx, m.ID)
	require.NoError(t, err)
	_, err = repo.UpdateDisplay(ctx, m.ID, service.ChannelMonitorDisplayUpdateParams{SetDisplayGroup: true})
	require.NoError(t, err)
	require.NoError(t, repo.DeleteDisplayGroup(ctx, group.ID))
	for range 8 {
		target := createMonitorDisplayGroup(t, "concurrent", 0)
		start := make(chan struct{})
		moveDone := make(chan error, 1)
		deleteDone := make(chan error, 1)
		go func() {
			<-start
			_, err := repo.UpdateDisplay(ctx, m.ID, service.ChannelMonitorDisplayUpdateParams{SetDisplayGroup: true, DisplayGroupID: &target.ID})
			moveDone <- err
		}()
		go func() { <-start; deleteDone <- repo.DeleteDisplayGroup(ctx, target.ID) }()
		close(start)
		moveErr, deleteErr := <-moveDone, <-deleteDone
		current, err := repo.GetByID(ctx, m.ID)
		require.NoError(t, err, "group deletion never deletes the monitor")
		if moveErr == nil {
			require.ErrorIs(t, deleteErr, service.ErrMonitorDisplayGroupConflict)
			require.Equal(t, &target.ID, current.DisplayGroupID)
			_, err = repo.UpdateDisplay(ctx, m.ID, service.ChannelMonitorDisplayUpdateParams{SetDisplayGroup: true})
			require.NoError(t, err)
		} else {
			require.ErrorIs(t, moveErr, service.ErrMonitorDisplayConflict)
			require.NoError(t, deleteErr)
			require.Nil(t, current.DisplayGroupID)
		}
	}
}

func TestChannelMonitorDisplayGroupConcurrentNameConflict(t *testing.T) {
	repo := NewChannelMonitorRepository(integrationEntClient, integrationDB)
	name := fmt.Sprintf("concurrent-name-%d", time.Now().UnixNano())
	groups := []*service.ChannelMonitorDisplayGroup{{Name: name}, {Name: name}}
	results := make(chan error, 2)
	start := make(chan struct{})
	for _, group := range groups {
		go func(g *service.ChannelMonitorDisplayGroup) {
			<-start
			results <- repo.CreateDisplayGroup(context.Background(), g)
		}(group)
	}
	close(start)
	first, second := <-results, <-results
	if first == nil {
		require.ErrorIs(t, second, service.ErrMonitorDisplayGroupConflict)
	} else {
		require.ErrorIs(t, first, service.ErrMonitorDisplayGroupConflict)
		require.NoError(t, second)
	}
	for _, group := range groups {
		if group.ID > 0 {
			require.NoError(t, repo.DeleteDisplayGroup(context.Background(), group.ID))
		}
	}
}

func TestChannelMonitorDisplayGroupConcurrentCreateAndDelete(t *testing.T) {
	ctx := context.Background()
	repo := NewChannelMonitorRepository(integrationEntClient, integrationDB)
	for range 8 {
		group := createMonitorDisplayGroup(t, "create-race", 0)
		m := &service.ChannelMonitor{Name: "concurrent-create", Provider: service.MonitorProviderOpenAI,
			APIKey: "ciphertext", PrimaryModel: "model", IntervalSeconds: 60, DisplayGroupID: &group.ID}
		start := make(chan struct{})
		created := make(chan error, 1)
		deleted := make(chan error, 1)
		go func() { <-start; created <- repo.Create(ctx, m) }()
		go func() { <-start; deleted <- repo.DeleteDisplayGroup(ctx, group.ID) }()
		close(start)
		createErr, deleteErr := <-created, <-deleted
		if createErr == nil {
			require.ErrorIs(t, deleteErr, service.ErrMonitorDisplayGroupConflict)
			loaded, err := repo.GetByID(ctx, m.ID)
			require.NoError(t, err)
			require.Equal(t, &group.ID, loaded.DisplayGroupID)
			require.NoError(t, repo.Delete(ctx, m.ID))
		} else {
			require.ErrorIs(t, createErr, service.ErrMonitorDisplayConflict)
			require.NoError(t, deleteErr)
			require.Zero(t, m.ID)
		}
	}
}

func TestChannelMonitorDisplayGroupDuplicatePersistence(t *testing.T) {
	ctx := context.Background()
	repo := NewChannelMonitorRepository(integrationEntClient, integrationDB)
	group := createMonitorDisplayGroup(t, "duplicate", 0)
	for _, groupID := range []*int64{nil, &group.ID} {
		source := createDisplayMonitor(t, groupID, 33, true)
		copy := *source
		copy.ID = 0
		copy.Enabled = false
		copy.DuplicateOperationID = fmt.Sprintf("display-copy-%d", source.ID)
		require.NoError(t, repo.Create(ctx, &copy))
		t.Cleanup(func() { _ = repo.Delete(ctx, copy.ID) })
		restored, err := repo.FindByDuplicateOperationID(ctx, copy.DuplicateOperationID)
		require.NoError(t, err)
		require.Equal(t, groupID, restored.DisplayGroupID)
		require.Equal(t, 33, restored.SortOrder)
		require.Equal(t, source.GroupName, restored.GroupName)
		require.False(t, restored.Enabled)
		history, err := repo.ListHistory(ctx, restored.ID, "", 10)
		require.NoError(t, err)
		require.Empty(t, history)
	}
}
