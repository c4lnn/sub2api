-- V1 展示分组独立于旧 group_name；存量保持 NULL 归属，不自动发布。
CREATE TABLE IF NOT EXISTS channel_monitor_display_groups (
    id BIGSERIAL PRIMARY KEY,
    name VARCHAR(100) NOT NULL UNIQUE,
    sort_order INTEGER NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT channel_monitor_display_groups_sort_order_check CHECK (sort_order >= 0)
);

ALTER TABLE channel_monitors
    ADD COLUMN IF NOT EXISTS display_group_id BIGINT,
    ADD COLUMN IF NOT EXISTS sort_order INTEGER NOT NULL DEFAULT 0;

DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conrelid = 'channel_monitors'::regclass AND conname = 'channel_monitors_display_group_id_fkey') THEN
        ALTER TABLE channel_monitors ADD CONSTRAINT channel_monitors_display_group_id_fkey
            FOREIGN KEY (display_group_id) REFERENCES channel_monitor_display_groups(id) ON DELETE RESTRICT;
    END IF;
    IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conrelid = 'channel_monitors'::regclass AND conname = 'channel_monitors_sort_order_check') THEN
        ALTER TABLE channel_monitors ADD CONSTRAINT channel_monitors_sort_order_check CHECK (sort_order >= 0);
    END IF;
END $$;

CREATE INDEX IF NOT EXISTS idx_channel_monitors_display_group_sort
    ON channel_monitors (display_group_id, sort_order, id);
