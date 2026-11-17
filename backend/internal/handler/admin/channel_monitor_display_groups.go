package admin

import (
	"bytes"
	"encoding/json"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
)

func (h *ChannelMonitorHandler) ListDisplayGroups(c *gin.Context) {
	groups, err := h.monitorService.ListDisplayGroups(c.Request.Context())
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, gin.H{"items": groups})
}

func (h *ChannelMonitorHandler) CreateDisplayGroup(c *gin.Context) {
	var req struct {
		Name      string           `json:"name"`
		SortOrder monitorSortOrder `json:"sort_order"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ErrorFrom(c, infraerrors.BadRequest("VALIDATION_ERROR", err.Error()))
		return
	}
	group, err := h.monitorService.CreateDisplayGroup(c.Request.Context(), req.Name, int(req.SortOrder))
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Created(c, group)
}

func (h *ChannelMonitorHandler) UpdateDisplayGroup(c *gin.Context) {
	id, ok := ParseChannelMonitorID(c)
	if !ok {
		return
	}
	var req struct {
		Name      json.RawMessage `json:"name"`
		SortOrder json.RawMessage `json:"sort_order"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ErrorFrom(c, infraerrors.BadRequest("VALIDATION_ERROR", err.Error()))
		return
	}
	p := service.ChannelMonitorDisplayGroupUpdateParams{}
	if len(req.Name) > 0 {
		var name string
		if err := json.Unmarshal(req.Name, &name); err != nil {
			response.ErrorFrom(c, service.ErrMonitorDisplayGroupName)
			return
		}
		p.Name = &name
	}
	if len(req.SortOrder) > 0 {
		var order monitorSortOrder
		if err := json.Unmarshal(req.SortOrder, &order); err != nil {
			response.ErrorFrom(c, service.ErrMonitorDisplaySortOrder)
			return
		}
		value := int(order)
		p.SortOrder = &value
	}
	group, err := h.monitorService.UpdateDisplayGroup(c.Request.Context(), id, p)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, group)
}

func (h *ChannelMonitorHandler) DeleteDisplayGroup(c *gin.Context) {
	id, ok := ParseChannelMonitorID(c)
	if !ok {
		return
	}
	if err := h.monitorService.DeleteDisplayGroup(c.Request.Context(), id); err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, gin.H{"message": "display group deleted"})
}

func (h *ChannelMonitorHandler) UpdateDisplay(c *gin.Context) {
	id, ok := ParseChannelMonitorID(c)
	if !ok {
		return
	}
	var req struct {
		DisplayGroupID json.RawMessage `json:"display_group_id"`
		SortOrder      json.RawMessage `json:"sort_order"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ErrorFrom(c, infraerrors.BadRequest("VALIDATION_ERROR", err.Error()))
		return
	}
	p := service.ChannelMonitorDisplayUpdateParams{SetDisplayGroup: len(req.DisplayGroupID) > 0}
	if p.SetDisplayGroup && !bytes.Equal(bytes.TrimSpace(req.DisplayGroupID), []byte("null")) {
		var groupID int64
		if err := json.Unmarshal(req.DisplayGroupID, &groupID); err != nil {
			response.ErrorFrom(c, service.ErrMonitorDisplayGroupID)
			return
		}
		p.DisplayGroupID = &groupID
	}
	if len(req.SortOrder) > 0 {
		var sortOrder int
		if bytes.Equal(bytes.TrimSpace(req.SortOrder), []byte("null")) || json.Unmarshal(req.SortOrder, &sortOrder) != nil {
			response.ErrorFrom(c, service.ErrMonitorDisplaySortOrder)
			return
		}
		p.SortOrder = &sortOrder
	}
	m, err := h.monitorService.UpdateDisplay(c.Request.Context(), id, p)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, channelMonitorToResponse(m))
}

// monitorSortOrder rejects JSON null, fractions and out-of-range values while
// allowing an omitted create field to retain its zero default.
type monitorSortOrder int

func (order *monitorSortOrder) UnmarshalJSON(raw []byte) error {
	var value int64
	if bytes.Equal(bytes.TrimSpace(raw), []byte("null")) || json.Unmarshal(raw, &value) != nil || value < 0 || value > 2147483647 {
		return service.ErrMonitorDisplaySortOrder
	}
	*order = monitorSortOrder(value)
	return nil
}
