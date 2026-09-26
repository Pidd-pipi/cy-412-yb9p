package handler

import (
	"errors"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/smartestate/smartestate/internal/constants"
	"github.com/smartestate/smartestate/internal/dto"
	"github.com/smartestate/smartestate/internal/service"
)

type RepairHandler struct {
	Handler
	svc *service.RepairService
}

func NewRepairHandler(s *service.RepairService, h *Handler) *RepairHandler {
	return &RepairHandler{Handler: *h, svc: s}
}
func (h *RepairHandler) List(c *gin.Context) {
	v, e := h.svc.List(c.Query("status"))
	if e != nil {
		Fail(c, 500, 50001, e.Error())
		return
	}
	OK(c, v)
}
func (h *RepairHandler) Create(c *gin.Context) {
	var r dto.CreateRepairRequest
	if !Bind(c, &r, h.Validate) {
		return
	}
	v, e := h.svc.Create(c.GetUint("userID"), r.Title, r.Description, r.Type, r.Images, r.AppointmentDate, r.AppointmentSlot)
	if e != nil {
		h.failRepair(c, e)
		return
	}
	OK(c, v)
}
func (h *RepairHandler) Assign(c *gin.Context) {
	var r dto.AssignRepairRequest
	if !Bind(c, &r, h.Validate) {
		return
	}
	id, _ := strconv.Atoi(c.Param("id"))
	v, e := h.svc.Assign(uint(id), r.HandlerID, c.GetString("role"))
	if e != nil {
		h.failRepair(c, e)
		return
	}
	OK(c, v)
}
func (h *RepairHandler) Status(c *gin.Context) {
	var r dto.UpdateRepairStatusRequest
	if !Bind(c, &r, h.Validate) {
		return
	}
	id, _ := strconv.Atoi(c.Param("id"))
	v, e := h.svc.UpdateStatus(uint(id), r.Status, r.Rating, c.GetString("role"))
	if e != nil {
		Fail(c, 400, 40001, e.Error())
		return
	}
	OK(c, v)
}
func (h *RepairHandler) Reschedule(c *gin.Context) {
	var r dto.RescheduleRepairRequest
	if !Bind(c, &r, h.Validate) {
		return
	}
	id, _ := strconv.Atoi(c.Param("id"))
	v, e := h.svc.Reschedule(uint(id), c.GetUint("userID"), r.AppointmentDate, r.AppointmentSlot, c.GetString("role"))
	if e != nil {
		h.failRepair(c, e)
		return
	}
	OK(c, v)
}

// failRepair 把预约业务错误映射成对应的状态码、错误码与中文提示，其余错误沿用 400 透传。
func (h *RepairHandler) failRepair(c *gin.Context, e error) {
	switch {
	case errors.Is(e, service.ErrRepairScheduleConflict):
		Fail(c, 409, constants.CodeConflict, constants.MessageRepairScheduleConflict)
	case errors.Is(e, service.ErrRepairDateOutOfWindow):
		Fail(c, 400, constants.CodeBadRequest, constants.MessageRepairDateOutOfWindow)
	case errors.Is(e, service.ErrRepairRescheduleClosed):
		Fail(c, 400, constants.CodeBadRequest, constants.MessageRepairRescheduleClosed)
	case errors.Is(e, service.ErrRepairRescheduleForbidden):
		Fail(c, 403, constants.CodeForbidden, constants.MessageRepairRescheduleForbidden)
	default:
		Fail(c, 400, 40001, e.Error())
	}
}
