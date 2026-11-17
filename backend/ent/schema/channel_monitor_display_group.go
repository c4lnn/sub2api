package schema

import (
	"github.com/Wei-Shaw/sub2api/ent/schema/mixins"

	"entgo.io/ent"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
)

// ChannelMonitorDisplayGroup 仅定义 V1 展示归属，不控制监控探测或业务路由。
type ChannelMonitorDisplayGroup struct {
	ent.Schema
}

func (ChannelMonitorDisplayGroup) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Annotation{Table: "channel_monitor_display_groups", Checks: map[string]string{
			"channel_monitor_display_groups_sort_order_check": "sort_order >= 0",
		}},
	}
}

func (ChannelMonitorDisplayGroup) Mixin() []ent.Mixin {
	return []ent.Mixin{mixins.TimeMixin{}}
}

func (ChannelMonitorDisplayGroup) Fields() []ent.Field {
	return []ent.Field{
		field.String("name").NotEmpty().MaxLen(100).Unique(),
		field.Int("sort_order").Default(0).Range(0, 2147483647),
	}
}

func (ChannelMonitorDisplayGroup) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("monitors", ChannelMonitor.Type).Ref("display_group"),
	}
}
