package constants

import "time"

const (
	RepairStatusPending    = "pending"
	RepairStatusAssigned   = "assigned"
	RepairStatusProcessing = "processing"
	RepairStatusDone       = "done"
	RepairStatusClosed     = "closed"
)

var ValidRepairStatuses = map[string]bool{RepairStatusPending: true, RepairStatusAssigned: true, RepairStatusProcessing: true, RepairStatusDone: true, RepairStatusClosed: true}

// 已完成或已关闭的工单不再占用师傅的预约时段。
var RepairFinalStatuses = map[string]bool{RepairStatusDone: true, RepairStatusClosed: true}

const (
	RepairSlotMorning   = "morning"
	RepairSlotAfternoon = "afternoon"
)

var ValidRepairSlots = map[string]bool{RepairSlotMorning: true, RepairSlotAfternoon: true}

// 预约窗口：从今天起共 7 天（今天 + 未来 6 天）。
const AppointmentWindowDays = 7

// AppointmentBounds 按本地时区返回允许预约的日期区间（含首尾）。
func AppointmentBounds(now time.Time) (time.Time, time.Time) {
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.Local)
	return today, today.AddDate(0, 0, AppointmentWindowDays-1)
}
