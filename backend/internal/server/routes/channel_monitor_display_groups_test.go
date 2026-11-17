package routes

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/handler"
	"github.com/Wei-Shaw/sub2api/internal/handler/admin"
	"github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

type monitorDisplayRouteRepo struct {
	service.ChannelMonitorRepository
}

func (*monitorDisplayRouteRepo) ListDisplayGroups(context.Context) ([]*service.ChannelMonitorDisplayGroup, error) {
	return []*service.ChannelMonitorDisplayGroup{{ID: 1, Name: "formal", SortOrder: 10}}, nil
}
func monitorDisplayRouteHandlers() *handler.Handlers {
	return &handler.Handlers{Admin: &handler.AdminHandlers{
		ChannelMonitor:         admin.NewChannelMonitorHandler(service.NewChannelMonitorService(&monitorDisplayRouteRepo{}, nil)),
		ChannelMonitorTemplate: admin.NewChannelMonitorRequestTemplateHandler(nil),
	}}
}
func TestMonitorDisplayGroupRoutesRequireAdminAuthentication(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	group := router.Group("/api/v1/admin")
	group.Use(gin.HandlerFunc(middleware.NewAdminAuthMiddleware(nil, nil, nil, nil)))
	registerChannelMonitorRoutes(group, monitorDisplayRouteHandlers(), newChannelMonitorRouteSettings(true))
	for _, tc := range []struct{ method, path string }{
		{http.MethodGet, "/channel-monitor-display-groups"},
		{http.MethodPost, "/channel-monitor-display-groups"},
		{http.MethodPut, "/channel-monitor-display-groups/1"},
		{http.MethodDelete, "/channel-monitor-display-groups/1"},
		{http.MethodPut, "/channel-monitors/1/display"},
	} {
		w := httptest.NewRecorder()
		router.ServeHTTP(w, httptest.NewRequest(tc.method, "/api/v1/admin"+tc.path, strings.NewReader(`{}`)))
		require.Equal(t, http.StatusUnauthorized, w.Code, tc.path)
	}
}
func TestMonitorDisplayGroupRoutesFeatureGateAndModeIndependence(t *testing.T) {
	for _, enabled := range []bool{false, true} {
		for _, mode := range []string{"v1", "v2", "v3"} {
			router := gin.New()
			registerChannelMonitorRoutes(router.Group("/api/v1/admin"), monitorDisplayRouteHandlers(), newChannelMonitorModeSettings(enabled, mode))
			w := httptest.NewRecorder()
			router.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/api/v1/admin/channel-monitor-display-groups", nil))
			if enabled {
				require.Equal(t, http.StatusOK, w.Code, mode)
				require.Contains(t, w.Body.String(), `"items":[{"id":1,"name":"formal"`)
			} else {
				require.Equal(t, http.StatusForbidden, w.Code, mode)
				require.Contains(t, w.Body.String(), "CHANNEL_MONITOR_DISABLED")
			}
		}
	}
}
