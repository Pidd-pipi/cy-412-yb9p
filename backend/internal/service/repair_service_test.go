package service

import (
	"errors"
	"io"
	"log/slog"
	"testing"
	"time"

	"github.com/smartestate/smartestate/internal/constants"
	"github.com/smartestate/smartestate/internal/model"
	"github.com/smartestate/smartestate/internal/repository"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func newRepairServiceDB(t *testing.T) (*RepairService, *gorm.DB, model.User, model.User) {
	t.Helper()
	db, e := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if e != nil {
		t.Fatal(e)
	}
	if e = db.AutoMigrate(&model.User{}, &model.Repair{}); e != nil {
		t.Fatal(e)
	}
	owner := model.User{Phone: "100", Nickname: "业主", Role: constants.UserRoleResident}
	staff := model.User{Phone: "200", Nickname: "王师傅", Role: constants.UserRoleStaff}
	if e = db.Create(&owner).Error; e != nil {
		t.Fatal(e)
	}
	if e = db.Create(&staff).Error; e != nil {
		t.Fatal(e)
	}
	s := NewRepairService(repository.NewRepairRepository(db), repository.NewUserRepository(db), slog.New(slog.NewTextHandler(io.Discard, nil)))
	return s, db, owner, staff
}

func TestRepairCreateAppointmentTable(t *testing.T) {
	s, _, owner, _ := newRepairServiceDB(t)
	tomorrow := time.Now().AddDate(0, 0, 1).Format("2006-01-02")
	yesterday := time.Now().AddDate(0, 0, -1).Format("2006-01-02")
	farFuture := time.Now().AddDate(0, 0, 8).Format("2006-01-02")
	for _, tt := range []struct {
		name string
		date string
		slot string
		want error
	}{
		{"tomorrow morning", tomorrow, constants.RepairSlotMorning, nil},
		{"today afternoon", time.Now().Format("2006-01-02"), constants.RepairSlotAfternoon, nil},
		{"yesterday rejected", yesterday, constants.RepairSlotMorning, ErrRepairDateOutOfWindow},
		{"day 8 rejected", farFuture, constants.RepairSlotMorning, ErrRepairDateOutOfWindow},
		{"bad slot rejected", tomorrow, "evening", nil}, // 非法时段不返回哨兵错误，但仍必须失败
	} {
		_, e := s.Create(owner.ID, "灯具损坏", "客厅灯具不亮需要检修", "水电", "", tt.date, tt.slot)
		if tt.name == "bad slot rejected" {
			if e == nil {
				t.Fatalf("%s: expected validation error, got nil", tt.name)
			}
			continue
		}
		if !errors.Is(e, tt.want) {
			t.Fatalf("%s: err=%v want=%v", tt.name, e, tt.want)
		}
	}
}

func TestRepairAssignSlotConflict(t *testing.T) {
	s, db, owner, staff := newRepairServiceDB(t)
	day := time.Now().AddDate(0, 0, 2).Format("2006-01-02")

	first, e := s.Create(owner.ID, "工单一", "水管漏水需要上门", "水电", "", day, constants.RepairSlotMorning)
	if e != nil {
		t.Fatal(e)
	}
	second, e := s.Create(owner.ID, "工单二", "插座无电需要上门", "水电", "", day, constants.RepairSlotMorning)
	if e != nil {
		t.Fatal(e)
	}

	// 第一张分派成功。
	if _, e = s.Assign(first.ID, staff.ID, "staff"); e != nil {
		t.Fatalf("first assign: %v", e)
	}
	// 同师傅同一天同一时段另有未结束工单：拒绝分派，原分派（未分派）保持不变。
	if _, e = s.Assign(second.ID, staff.ID, "staff"); !errors.Is(e, ErrRepairScheduleConflict) {
		t.Fatalf("second assign: err=%v want conflict", e)
	}
	var fresh model.Repair
	if e = db.First(&fresh, second.ID).Error; e != nil {
		t.Fatal(e)
	}
	if fresh.HandlerID != nil || fresh.Status != constants.RepairStatusPending {
		t.Fatalf("original assignment must stay unchanged, got handler=%v status=%s", fresh.HandlerID, fresh.Status)
	}

	// 第一张完成后时段释放，第二张可分派给同一师傅。
	if _, e = s.UpdateStatus(first.ID, constants.RepairStatusDone, 0, "staff"); e != nil {
		t.Fatal(e)
	}
	if _, e = s.Assign(second.ID, staff.ID, "staff"); e != nil {
		t.Fatalf("assign after release: %v", e)
	}
}

func TestRepairRescheduleRules(t *testing.T) {
	s, db, owner, staff := newRepairServiceDB(t)
	other := model.User{Phone: "101", Nickname: "其他业主", Role: constants.UserRoleResident}
	if e := db.Create(&other).Error; e != nil {
		t.Fatal(e)
	}
	day1 := time.Now().AddDate(0, 0, 1).Format("2006-01-02")
	day2 := time.Now().AddDate(0, 0, 3).Format("2006-01-02")

	open, e := s.Create(owner.ID, "占用工单", "上午时段占用", "家具", "", day1, constants.RepairSlotMorning)
	if e != nil {
		t.Fatal(e)
	}
	if _, e = s.Assign(open.ID, staff.ID, "staff"); e != nil {
		t.Fatal(e)
	}

	mine, e := s.Create(owner.ID, "我的工单", "需要改约", "家具", "", day2, constants.RepairSlotAfternoon)
	if e != nil {
		t.Fatal(e)
	}
	if _, e = s.Assign(mine.ID, staff.ID, "staff"); e != nil {
		t.Fatal(e)
	}

	// 非报修人不能改预约。
	if _, e = s.Reschedule(mine.ID, other.ID, day2, constants.RepairSlotMorning, "resident"); !errors.Is(e, ErrRepairRescheduleForbidden) {
		t.Fatalf("non-owner reschedule: err=%v want forbidden", e)
	}
	// 改到师傅已有未结束工单的同日同时段：拒绝。
	if _, e = s.Reschedule(mine.ID, owner.ID, day1, constants.RepairSlotMorning, "resident"); !errors.Is(e, ErrRepairScheduleConflict) {
		t.Fatalf("conflict reschedule: err=%v want conflict", e)
	}
	// 改到空闲时段：成功。
	if _, e = s.Reschedule(mine.ID, owner.ID, day2, constants.RepairSlotMorning, "resident"); e != nil {
		t.Fatalf("free reschedule: %v", e)
	}

	// 完成或关闭后不可再改约。
	if _, e = s.UpdateStatus(mine.ID, constants.RepairStatusClosed, 0, "staff"); e != nil {
		t.Fatal(e)
	}
	if _, e = s.Reschedule(mine.ID, owner.ID, day2, constants.RepairSlotAfternoon, "resident"); !errors.Is(e, ErrRepairRescheduleClosed) {
		t.Fatalf("closed reschedule: err=%v want closed", e)
	}
}
