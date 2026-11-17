//go:build unit

package admin

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sort"
	"strings"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

type monitorDisplayHandlerRepo struct {
	service.ChannelMonitorRepository
	groups  map[int64]*service.ChannelMonitorDisplayGroup
	monitor service.ChannelMonitor
	nextID  int64
}

func (r *monitorDisplayHandlerRepo) ListDisplayGroups(context.Context) ([]*service.ChannelMonitorDisplayGroup, error) {
	out := make([]*service.ChannelMonitorDisplayGroup, 0, len(r.groups))
	for _, group := range r.groups {
		out = append(out, group)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].SortOrder != out[j].SortOrder {
			return out[i].SortOrder < out[j].SortOrder
		}
		return out[i].ID < out[j].ID
	})
	return out, nil
}
func (r *monitorDisplayHandlerRepo) GetDisplayGroup(_ context.Context, id int64) (*service.ChannelMonitorDisplayGroup, error) {
	if group := r.groups[id]; group != nil {
		return group, nil
	}
	return nil, service.ErrMonitorDisplayGroupNotFound
}
func (r *monitorDisplayHandlerRepo) CreateDisplayGroup(_ context.Context, group *service.ChannelMonitorDisplayGroup) error {
	for _, old := range r.groups {
		if old.Name == group.Name {
			return service.ErrMonitorDisplayGroupConflict
		}
	}
	r.nextID++
	group.ID = r.nextID
	copy := *group
	r.groups[group.ID] = &copy
	return nil
}
func (r *monitorDisplayHandlerRepo) UpdateDisplayGroup(_ context.Context, id int64, p service.ChannelMonitorDisplayGroupUpdateParams) (*service.ChannelMonitorDisplayGroup, error) {
	group := r.groups[id]
	if group == nil {
		return nil, service.ErrMonitorDisplayGroupNotFound
	}
	if p.Name != nil {
		for otherID, old := range r.groups {
			if otherID != id && old.Name == *p.Name {
				return nil, service.ErrMonitorDisplayGroupConflict
			}
		}
		group.Name = *p.Name
	}
	if p.SortOrder != nil {
		group.SortOrder = *p.SortOrder
	}
	return group, nil
}
func (r *monitorDisplayHandlerRepo) DeleteDisplayGroup(_ context.Context, id int64) error {
	if r.groups[id] == nil {
		return service.ErrMonitorDisplayGroupNotFound
	}
	if r.monitor.DisplayGroupID != nil && *r.monitor.DisplayGroupID == id {
		return service.ErrMonitorDisplayGroupConflict
	}
	delete(r.groups, id)
	return nil
}
func (r *monitorDisplayHandlerRepo) UpdateDisplay(_ context.Context, id int64, p service.ChannelMonitorDisplayUpdateParams) (*service.ChannelMonitor, error) {
	if id != r.monitor.ID {
		return nil, service.ErrChannelMonitorNotFound
	}
	if p.SetDisplayGroup {
		r.monitor.DisplayGroupID = p.DisplayGroupID
	}
	if p.SortOrder != nil {
		r.monitor.SortOrder = *p.SortOrder
	}
	copy := r.monitor
	return &copy, nil
}
func (r *monitorDisplayHandlerRepo) List(_ context.Context, p service.ChannelMonitorListParams) ([]*service.ChannelMonitor, int64, error) {
	if p.DisplayGroupID != nil && ((*p.DisplayGroupID == 0 && r.monitor.DisplayGroupID != nil) || (*p.DisplayGroupID > 0 && (r.monitor.DisplayGroupID == nil || *r.monitor.DisplayGroupID != *p.DisplayGroupID))) {
		return []*service.ChannelMonitor{}, 0, nil
	}
	copy := r.monitor
	return []*service.ChannelMonitor{&copy}, 1, nil
}
func (r *monitorDisplayHandlerRepo) ListLatestForMonitorIDs(context.Context, []int64) (map[int64][]*service.ChannelMonitorLatest, error) {
	return nil, nil
}
func (r *monitorDisplayHandlerRepo) ComputeAvailabilityForMonitors(context.Context, []int64, int) (map[int64][]*service.ChannelMonitorAvailability, error) {
	return nil, nil
}

func monitorDisplayRequest(router *gin.Engine, method, path, body string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	return rec
}
func newMonitorDisplayHandlerRouter() (*gin.Engine, *monitorDisplayHandlerRepo) {
	gin.SetMode(gin.TestMode)
	repo := &monitorDisplayHandlerRepo{groups: map[int64]*service.ChannelMonitorDisplayGroup{}, monitor: service.ChannelMonitor{ID: 42, Name: "monitor", APIKey: "ENC:secret", GroupName: "legacy", SortOrder: 17, Enabled: false}}
	h := NewChannelMonitorHandler(service.NewChannelMonitorService(repo, duplicateChannelMonitorHandlerEncryptor{}))
	r := gin.New()
	r.GET("/groups", h.ListDisplayGroups)
	r.POST("/groups", h.CreateDisplayGroup)
	r.PUT("/groups/:id", h.UpdateDisplayGroup)
	r.DELETE("/groups/:id", h.DeleteDisplayGroup)
	r.PUT("/monitors/:id/display", h.UpdateDisplay)
	r.GET("/monitors", h.List)
	return r, repo
}
func TestMonitorDisplayGroupHandlerCRUDEnvelopeAndErrors(t *testing.T) {
	router, repo := newMonitorDisplayHandlerRouter()
	rec := monitorDisplayRequest(router, http.MethodGet, "/groups", "")
	require.Equal(t, http.StatusOK, rec.Code)
	require.Contains(t, rec.Body.String(), `"items":[]`)
	rec = monitorDisplayRequest(router, http.MethodPost, "/groups", `{"name":"  正式组  ","sort_order":20}`)
	require.Equal(t, http.StatusCreated, rec.Code)
	require.Contains(t, rec.Body.String(), `"name":"正式组"`)
	for _, field := range []string{`"id":1`, `"sort_order":20`, `"created_at":`, `"updated_at":`} {
		require.Contains(t, rec.Body.String(), field)
	}
	rec = monitorDisplayRequest(router, http.MethodPost, "/groups", `{"name":"正式组"}`)
	require.Equal(t, http.StatusConflict, rec.Code)
	rec = monitorDisplayRequest(router, http.MethodPut, "/groups/1", `{"name":"  改名  "}`)
	require.Equal(t, http.StatusOK, rec.Code)
	require.Contains(t, rec.Body.String(), `"name":"改名"`)
	require.Contains(t, rec.Body.String(), `"sort_order":20`)
	id := int64(1)
	repo.monitor.DisplayGroupID = &id // disabled monitors still block deletion
	rec = monitorDisplayRequest(router, http.MethodDelete, "/groups/1", "")
	require.Equal(t, http.StatusConflict, rec.Code)
	repo.monitor.DisplayGroupID = nil
	rec = monitorDisplayRequest(router, http.MethodDelete, "/groups/1", "")
	require.Equal(t, http.StatusOK, rec.Code)
	rec = monitorDisplayRequest(router, http.MethodDelete, "/groups/1", "")
	require.Equal(t, http.StatusNotFound, rec.Code)
	for _, body := range []string{`{"name":" "}`, `{"name":"valid","sort_order":-1}`, `{"name":"valid","sort_order":2147483648}`, `{"name":"valid","sort_order":1.5}`} {
		rec = monitorDisplayRequest(router, http.MethodPost, "/groups", body)
		require.Equal(t, http.StatusBadRequest, rec.Code, body)
	}
}
func TestChannelMonitorDisplayHandlerPartialNullAndValidation(t *testing.T) {
	router, repo := newMonitorDisplayHandlerRouter()
	repo.groups[1] = &service.ChannelMonitorDisplayGroup{ID: 1, Name: "formal"}
	rec := monitorDisplayRequest(router, http.MethodPut, "/monitors/42/display", `{"display_group_id":1}`)
	require.Equal(t, http.StatusOK, rec.Code)
	require.Contains(t, rec.Body.String(), `"display_group_id":1`)
	require.Contains(t, rec.Body.String(), `"sort_order":17`)
	rec = monitorDisplayRequest(router, http.MethodPut, "/monitors/42/display", `{"sort_order":9}`)
	require.Equal(t, http.StatusOK, rec.Code)
	require.Contains(t, rec.Body.String(), `"display_group_id":1`)
	require.Contains(t, rec.Body.String(), `"api_key_masked":"secr***"`)
	require.NotContains(t, rec.Body.String(), "ENC:secret")
	rec = monitorDisplayRequest(router, http.MethodPut, "/monitors/42/display", `{"display_group_id":null}`)
	require.Equal(t, http.StatusOK, rec.Code)
	require.Contains(t, rec.Body.String(), `"display_group_id":null`)
	require.Contains(t, rec.Body.String(), `"sort_order":9`)
	require.Equal(t, "legacy", repo.monitor.GroupName)
	for _, body := range []string{`{}`, `{"display_group_id":0}`, `{"display_group_id":-1}`, `{"display_group_id":1.5}`, `{"sort_order":-1}`, `{"sort_order":2147483648}`, `{"sort_order":1.5}`, `{"sort_order":null}`} {
		rec = monitorDisplayRequest(router, http.MethodPut, "/monitors/42/display", body)
		require.Equal(t, http.StatusBadRequest, rec.Code, body)
		require.Nil(t, repo.monitor.DisplayGroupID)
		require.Equal(t, 9, repo.monitor.SortOrder)
	}
	rec = monitorDisplayRequest(router, http.MethodPut, "/monitors/42/display", `{"display_group_id":99}`)
	require.Equal(t, http.StatusNotFound, rec.Code)
	rec = monitorDisplayRequest(router, http.MethodPut, "/monitors/99/display", `{"sort_order":1}`)
	require.Equal(t, http.StatusNotFound, rec.Code)
}
func TestChannelMonitorDisplayHandlerListGroupFilter(t *testing.T) {
	router, _ := newMonitorDisplayHandlerRouter()
	for _, query := range []string{"", "?display_group_id=0"} {
		rec := monitorDisplayRequest(router, http.MethodGet, "/monitors"+query, "")
		require.Equal(t, http.StatusOK, rec.Code)
		var body struct {
			Data struct {
				Items []json.RawMessage `json:"items"`
			} `json:"data"`
		}
		require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
		require.Len(t, body.Data.Items, 1)
	}
	rec := monitorDisplayRequest(router, http.MethodGet, "/monitors?display_group_id=1", "")
	require.Equal(t, http.StatusOK, rec.Code)
	require.Contains(t, rec.Body.String(), `"items":[]`)
	for _, query := range []string{"-1", "garbage", "1.5", "9223372036854775808"} {
		rec = monitorDisplayRequest(router, http.MethodGet, "/monitors?display_group_id="+query, "")
		require.Equal(t, http.StatusBadRequest, rec.Code)
	}
}
