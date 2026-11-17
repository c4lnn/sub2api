//go:build unit

package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/Wei-Shaw/sub2api/internal/domain"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

type channelMonitorPublishedHandlerRepo struct {
	service.ChannelMonitorRepository
	published []*service.ChannelMonitor
	reads     int
}

func (r *channelMonitorPublishedHandlerRepo) ListPublished(context.Context) ([]*service.ChannelMonitor, error) {
	r.reads++
	return r.published, nil
}
func (r *channelMonitorPublishedHandlerRepo) GetPublishedByID(_ context.Context, id int64) (*service.ChannelMonitor, error) {
	r.reads++
	for _, m := range r.published {
		if m.ID == id {
			return m, nil
		}
	}
	return nil, service.ErrChannelMonitorNotFound
}
func (r *channelMonitorPublishedHandlerRepo) ListLatestForMonitorIDs(_ context.Context, ids []int64) (map[int64][]*service.ChannelMonitorLatest, error) {
	out := make(map[int64][]*service.ChannelMonitorLatest, len(ids))
	for _, id := range ids {
		out[id] = []*service.ChannelMonitorLatest{{Model: "model", Status: service.MonitorStatusOperational, Quota: &domain.MonitorQuotaSnapshot{Success: true}}}
	}
	return out, nil
}
func (r *channelMonitorPublishedHandlerRepo) ComputeAvailabilityForMonitors(context.Context, []int64, int) (map[int64][]*service.ChannelMonitorAvailability, error) {
	return nil, nil
}
func (r *channelMonitorPublishedHandlerRepo) ListRecentHistoryForMonitors(context.Context, []int64, map[int64]string, int) (map[int64][]*service.ChannelMonitorHistoryEntry, error) {
	return nil, nil
}
func (r *channelMonitorPublishedHandlerRepo) ListLatestPerModel(context.Context, int64) ([]*service.ChannelMonitorLatest, error) {
	return nil, nil
}
func (r *channelMonitorPublishedHandlerRepo) ComputeAvailability(context.Context, int64, int) ([]*service.ChannelMonitorAvailability, error) {
	return nil, nil
}

func channelMonitorUserTestRouter(repo *channelMonitorPublishedHandlerRepo, enabled, showQuota bool) *gin.Engine {
	settings := map[string]string{service.SettingKeyChannelMonitorEnabled: "false", service.SettingKeyChannelMonitorMode: "v1", service.SettingKeyChannelMonitorShowQuota: "false"}
	if enabled {
		settings[service.SettingKeyChannelMonitorEnabled] = "true"
	}
	if showQuota {
		settings[service.SettingKeyChannelMonitorShowQuota] = "true"
	}
	settingSvc := service.NewSettingService(&settingHandlerPublicRepoStub{values: settings}, &config.Config{})
	h := NewChannelMonitorUserHandler(service.NewChannelMonitorService(repo, nil), settingSvc)
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET("/monitors", h.List)
	r.GET("/monitors/:id/status", h.GetStatus)
	return r
}
func TestChannelMonitorUserPublishedDTOAndQuotaGate(t *testing.T) {
	groupID := int64(8)
	for _, quota := range []bool{false, true} {
		repo := &channelMonitorPublishedHandlerRepo{published: []*service.ChannelMonitor{{ID: 1, Name: "visible", Provider: "openai", GroupName: "wrong legacy", PrimaryModel: "model", Enabled: true, DisplayGroupID: &groupID, DisplayGroupName: "正式", DisplayGroupSortOrder: 20, SortOrder: 7}}}
		r := channelMonitorUserTestRouter(repo, true, quota)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/monitors", nil))
		require.Equal(t, http.StatusOK, w.Code)
		var body struct {
			Data struct {
				Items []map[string]any `json:"items"`
			} `json:"data"`
		}
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
		require.Len(t, body.Data.Items, 1)
		item := body.Data.Items[0]
		require.EqualValues(t, 8, item["display_group_id"])
		require.Equal(t, "正式", item["display_group_name"])
		require.EqualValues(t, 20, item["display_group_sort_order"])
		require.EqualValues(t, 7, item["sort_order"])
		_, hasQuota := item["latest_quota"]
		require.Equal(t, quota, hasQuota)
		require.NotContains(t, w.Body.String(), "api_key")
		w = httptest.NewRecorder()
		r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/monitors/1/status", nil))
		require.Equal(t, http.StatusOK, w.Code)
		require.Contains(t, w.Body.String(), `"display_group_name":"正式"`)
		require.Contains(t, w.Body.String(), `"display_group_id":8`)
	}
}
func TestChannelMonitorUserInvisibleDetailsAndAllUnpublished(t *testing.T) {
	// Repository's published query rejects disabled/unpublished/nonexistent IDs
	// before the service can invoke any of the history aggregation methods.
	repo := &channelMonitorPublishedHandlerRepo{}
	r := channelMonitorUserTestRouter(repo, true, false)
	var first string
	for _, path := range []string{"/monitors/1/status", "/monitors/2/status", "/monitors/999/status"} {
		w := httptest.NewRecorder()
		r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, path, nil))
		require.Equal(t, http.StatusNotFound, w.Code)
		if first == "" {
			first = w.Body.String()
		} else {
			require.JSONEq(t, first, w.Body.String())
		}
		require.NotContains(t, w.Body.String(), "models")
		require.NotContains(t, w.Body.String(), "quota")
	}
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/monitors", nil))
	require.Equal(t, http.StatusOK, w.Code)
	require.Contains(t, w.Body.String(), `"items":[]`)
}
func TestChannelMonitorUserFeatureGatePreventsRepositoryReads(t *testing.T) {
	repo := &channelMonitorPublishedHandlerRepo{}
	r := channelMonitorUserTestRouter(repo, false, true)
	for _, path := range []string{"/monitors", "/monitors/1/status"} {
		w := httptest.NewRecorder()
		r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, path, nil))
		if path == "/monitors" {
			require.Equal(t, http.StatusOK, w.Code)
			require.Contains(t, w.Body.String(), `"items":[]`)
		} else {
			require.Equal(t, http.StatusNotFound, w.Code)
		}
	}
	require.Zero(t, repo.reads)
}
