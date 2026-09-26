-- 002_repair_appointment.sql
-- 工单预约时段：业主提交时选择未来七天内的日期与上午/下午时段。
-- 本地 SQLite 与演示环境由 GORM AutoMigrate 自动补列；此文件记录 MySQL 的版本化 schema 变更。
ALTER TABLE repairs ADD COLUMN appointment_date VARCHAR(10) NOT NULL DEFAULT '';
ALTER TABLE repairs ADD COLUMN appointment_slot VARCHAR(10) NOT NULL DEFAULT '';
CREATE INDEX idx_repairs_handler_slot ON repairs (handler_id, appointment_date, appointment_slot);
