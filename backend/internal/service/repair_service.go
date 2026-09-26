package service

import (
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/smartestate/smartestate/internal/constants"
	"github.com/smartestate/smartestate/internal/model"
	"github.com/smartestate/smartestate/internal/repository"
)

// 预约相关业务错误，handler 层据此映射状态码与提示文案。
var (
	ErrRepairDateOutOfWindow     = errors.New("repair appointment date out of window")
	ErrRepairScheduleConflict    = errors.New("repair handler slot conflict")
	ErrRepairRescheduleClosed    = errors.New("repair already finished, cannot reschedule")
	ErrRepairRescheduleForbidden = errors.New("repair reschedule forbidden for this user")
)

type RepairService struct {
	repo   *repository.RepairRepository
	users  *repository.UserRepository
	logger *slog.Logger
}

func NewRepairService(r *repository.RepairRepository, u *repository.UserRepository, l *slog.Logger) *RepairService {
	return &RepairService{r, u, l}
}

// validateAppointment 校验时段合法且日期落在今天起的未来七天内。
func validateAppointment(date, slot string) error {
	if !constants.ValidRepairSlots[slot] {
		return fmt.Errorf("Repair appointment invalid: slot=%s", slot)
	}
	d, e := time.ParseInLocation("2006-01-02", date, time.Local)
	if e != nil {
		return fmt.Errorf("Repair appointment invalid: date=%s: %w", date, e)
	}
	start, end := constants.AppointmentBounds(time.Now())
	if d.Before(start) || d.After(end) {
		return fmt.Errorf("%w: date=%s, window=[%s,%s]", ErrRepairDateOutOfWindow, date, start.Format("2006-01-02"), end.Format("2006-01-02"))
	}
	return nil
}

func (s *RepairService) Create(uid uint, title, desc, typ, images, date, slot string) (model.Repair, error) {
	if e := validateAppointment(date, slot); e != nil {
		return model.Repair{}, fmt.Errorf("Repair[user_id=%d] create failed, current role=resident: %w", uid, e)
	}
	v := model.Repair{UserID: uid, Title: title, Description: desc, Type: typ, Images: images, Status: constants.RepairStatusPending, AppointmentDate: date, AppointmentSlot: slot}
	if e := s.repo.Create(&v); e != nil {
		return v, fmt.Errorf("Repair[user_id=%d] create failed: %w", uid, e)
	}
	return s.repo.ByID(v.ID)
}
func (s *RepairService) List(status string) ([]model.Repair, error) { return s.repo.List(status) }

func (s *RepairService) Assign(id, handlerID uint, role string) (model.Repair, error) {
	handler, e := s.users.ByID(handlerID)
	if e != nil {
		return model.Repair{}, fmt.Errorf("Repair[id=%d] assign failed: staff not found, current role=%s: %w", id, role, e)
	}
	if handler.Role != constants.UserRoleStaff && handler.Role != constants.UserRoleAdmin {
		return model.Repair{}, fmt.Errorf("Repair[id=%d] assign failed: handler %d is not staff/admin, current role=%s", id, handlerID, role)
	}
	v, e := s.repo.ByID(id)
	if e != nil {
		return v, e
	}
	// 该师傅同一天同一时段仍有未结束工单：拒绝分派，原分派保持不变。
	if v.AppointmentDate != "" && v.AppointmentSlot != "" {
		n, ce := s.repo.CountHandlerSlotConflict(handlerID, v.AppointmentDate, v.AppointmentSlot, v.ID)
		if ce != nil {
			return v, fmt.Errorf("Repair[id=%d] assign failed: slot conflict query error: %w", id, ce)
		}
		if n > 0 {
			return v, fmt.Errorf("Repair[id=%d] assign failed: %w: handler=%d date=%s slot=%s, current role=%s", id, ErrRepairScheduleConflict, handlerID, v.AppointmentDate, v.AppointmentSlot, role)
		}
	}
	v.HandlerID = &handlerID
	v.Handler = nil
	v.User = model.User{}
	v.Status = constants.RepairStatusAssigned
	if e = s.repo.Update(&v); e != nil {
		return v, fmt.Errorf("Repair[id=%d] assign failed: %w", id, e)
	}
	return s.repo.ByID(id)
}
func (s *RepairService) UpdateStatus(id uint, status string, rating int, role string) (model.Repair, error) {
	if !constants.ValidRepairStatuses[status] {
		return model.Repair{}, fmt.Errorf("Repair[id=%d] status failed: invalid status, current role=%s", id, role)
	}
	v, e := s.repo.ByID(id)
	if e != nil {
		return v, e
	}
	// 完成或关闭即释放时段：状态进入终态后不再参与同时段冲突统计。
	v.Status = status
	if rating > 0 {
		v.Rating = rating
	}
	if e = s.repo.Update(&v); e != nil {
		return v, fmt.Errorf("Repair[id=%d] status failed: %w", id, e)
	}
	return s.repo.ByID(id)
}

// Reschedule 仅允许报修人在工单完成或关闭前修改预约日期与时段；与处理人既有未结束工单冲突时拒绝。
func (s *RepairService) Reschedule(id, uid uint, date, slot, role string) (model.Repair, error) {
	if e := validateAppointment(date, slot); e != nil {
		return model.Repair{}, fmt.Errorf("Repair[id=%d] reschedule failed, current role=%s: %w", id, role, e)
	}
	v, e := s.repo.ByID(id)
	if e != nil {
		return v, e
	}
	if v.UserID != uid {
		return v, fmt.Errorf("%w: Repair[id=%d] owner=%d operator=%d, current role=%s", ErrRepairRescheduleForbidden, id, v.UserID, uid, role)
	}
	if constants.RepairFinalStatuses[v.Status] {
		return v, fmt.Errorf("%w: Repair[id=%d] status=%s", ErrRepairRescheduleClosed, id, v.Status)
	}
	if v.HandlerID != nil {
		n, ce := s.repo.CountHandlerSlotConflict(*v.HandlerID, date, slot, v.ID)
		if ce != nil {
			return v, fmt.Errorf("Repair[id=%d] reschedule failed: slot conflict query error: %w", id, ce)
		}
		if n > 0 {
			return v, fmt.Errorf("Repair[id=%d] reschedule failed: %w: handler=%d date=%s slot=%s, current role=%s", id, ErrRepairScheduleConflict, *v.HandlerID, date, slot, role)
		}
	}
	v.AppointmentDate = date
	v.AppointmentSlot = slot
	v.Handler = nil
	v.User = model.User{}
	if e = s.repo.Update(&v); e != nil {
		return v, fmt.Errorf("Repair[id=%d] reschedule failed: %w", id, e)
	}
	return s.repo.ByID(id)
}

func (s *RepairService) OpenCount() (int64, error) { return s.repo.CountOpen() }
