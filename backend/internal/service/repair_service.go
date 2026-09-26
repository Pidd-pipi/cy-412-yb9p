package service

import (
	"fmt"
	"github.com/smartestate/smartestate/internal/constants"
	"github.com/smartestate/smartestate/internal/model"
	"github.com/smartestate/smartestate/internal/repository"
	"log/slog"
	"time"
)

type RepairService struct {
	repo   *repository.RepairRepository
	users  *repository.UserRepository
	logger *slog.Logger
}

func NewRepairService(r *repository.RepairRepository, u *repository.UserRepository, l *slog.Logger) *RepairService {
	return &RepairService{r, u, l}
}
func validAppointment(visitDate, timeSlot string) bool {
	if !constants.ValidTimeSlots[timeSlot] {
		return false
	}
	d, e := time.ParseInLocation("2006-01-02", visitDate, time.Local)
	if e != nil {
		return false
	}
	now := time.Now()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.Local)
	return !d.Before(today) && !d.After(today.AddDate(0, 0, 6))
}
func (s *RepairService) Create(uid uint, title, desc, typ, images, visitDate, timeSlot, role string) (model.Repair, error) {
	if !validAppointment(visitDate, timeSlot) {
		return model.Repair{}, fmt.Errorf("Repair[user_id=%d] create failed: visit_date=%s time_slot=%s not within next 7 days, current role=%s", uid, visitDate, timeSlot, role)
	}
	v := model.Repair{UserID: uid, Title: title, Description: desc, Type: typ, Images: images, Status: constants.RepairStatusPending, VisitDate: visitDate, TimeSlot: timeSlot}
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
	if v.VisitDate != "" && v.TimeSlot != "" {
		n, ce := s.repo.CountOpenByHandlerSlot(handlerID, v.VisitDate, v.TimeSlot, v.ID)
		if ce != nil {
			return v, fmt.Errorf("Repair[id=%d] assign failed: slot conflict check failed, current role=%s: %w", id, role, ce)
		}
		if n > 0 {
			return v, fmt.Errorf("Repair[id=%d] assign failed: handler %d at %s %s: %s, current role=%s", id, handlerID, v.VisitDate, v.TimeSlot, constants.MessageSlotConflict, role)
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
	v.Status = status
	if rating > 0 {
		v.Rating = rating
	}
	if e = s.repo.Update(&v); e != nil {
		return v, fmt.Errorf("Repair[id=%d] status failed: %w", id, e)
	}
	return s.repo.ByID(id)
}
func (s *RepairService) Reschedule(id, uid uint, visitDate, timeSlot, role string) (model.Repair, error) {
	if !validAppointment(visitDate, timeSlot) {
		return model.Repair{}, fmt.Errorf("Repair[id=%d] reschedule failed: visit_date=%s time_slot=%s not within next 7 days, current role=%s", id, visitDate, timeSlot, role)
	}
	v, e := s.repo.ByID(id)
	if e != nil {
		return v, e
	}
	if v.UserID != uid {
		return v, fmt.Errorf("Repair[id=%d] reschedule failed: only the reporter can change the appointment, current role=%s", id, role)
	}
	if v.Status == constants.RepairStatusDone || v.Status == constants.RepairStatusClosed {
		return v, fmt.Errorf("Repair[id=%d] reschedule failed: status=%s, %s, current role=%s", id, v.Status, constants.MessageRepairFinished, role)
	}
	if v.HandlerID != nil {
		n, ce := s.repo.CountOpenByHandlerSlot(*v.HandlerID, visitDate, timeSlot, v.ID)
		if ce != nil {
			return v, fmt.Errorf("Repair[id=%d] reschedule failed: slot conflict check failed, current role=%s: %w", id, role, ce)
		}
		if n > 0 {
			return v, fmt.Errorf("Repair[id=%d] reschedule failed: handler %d at %s %s: %s, current role=%s", id, *v.HandlerID, visitDate, timeSlot, constants.MessageSlotConflict, role)
		}
	}
	v.VisitDate = visitDate
	v.TimeSlot = timeSlot
	v.Handler = nil
	v.User = model.User{}
	if e = s.repo.Update(&v); e != nil {
		return v, fmt.Errorf("Repair[id=%d] reschedule failed: %w", id, e)
	}
	return s.repo.ByID(id)
}
func (s *RepairService) OpenCount() (int64, error) { return s.repo.CountOpen() }
