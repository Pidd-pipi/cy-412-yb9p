package util

import (
	"fmt"
	"github.com/smartestate/smartestate/internal/constants"
	"time"
)

func Date(v time.Time) string { return v.Format("2006-01-02 15:04") }
func AppointmentDay(v string) string {
	if t, e := time.ParseInLocation("2006-01-02", v, time.Local); e == nil {
		return t.Format("2006年01月02日")
	}
	return v
}
func Money(v float64) string { return fmt.Sprintf("¥%.2f", v) }
func SlotText(v string) string {
	m := map[string]string{constants.RepairSlotMorning: "上午", constants.RepairSlotAfternoon: "下午"}
	if t, ok := m[v]; ok {
		return t
	}
	return "未预约"
}
func StatusText(v string) string {
	m := map[string]string{constants.RepairStatusPending: "待受理", constants.RepairStatusAssigned: "已分派", constants.RepairStatusProcessing: "处理中", constants.RepairStatusDone: "已完成", constants.RepairStatusClosed: "已关闭"}
	return m[v]
}
func RoleText(v string) string {
	m := map[string]string{constants.UserRoleResident: "业主", constants.UserRoleStaff: "物业人员", constants.UserRoleAdmin: "管理员"}
	return m[v]
}
