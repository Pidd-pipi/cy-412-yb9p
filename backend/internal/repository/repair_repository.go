package repository

import (
	"errors"
	"github.com/smartestate/smartestate/internal/model"
	"gorm.io/gorm"
)

type RepairRepository struct{ DB *gorm.DB }

func NewRepairRepository(db *gorm.DB) *RepairRepository  { return &RepairRepository{db} }
func (r *RepairRepository) Create(v *model.Repair) error { return r.DB.Create(v).Error }
func (r *RepairRepository) List(status string) (out []model.Repair, e error) {
	q := r.DB.Preload("User").Preload("Handler").Order("created_at desc")
	if status != "" {
		q = q.Where("status = ?", status)
	}
	e = q.Find(&out).Error
	return
}
func (r *RepairRepository) ByID(id uint) (v model.Repair, e error) {
	e = r.DB.Preload("User").Preload("Handler").First(&v, id).Error
	if errors.Is(e, gorm.ErrRecordNotFound) {
		e = ErrNotFound
	}
	return
}
func (r *RepairRepository) Update(v *model.Repair) error { return r.DB.Save(v).Error }
func (r *RepairRepository) CountOpen() (int64, error) {
	var n int64
	e := r.DB.Model(&model.Repair{}).Where("status NOT IN ?", []string{"done", "closed"}).Count(&n).Error
	return n, e
}

// CountHandlerSlotConflict 统计该师傅同一天同一时段仍未结束的其他工单数；excludeID 用于排除当前工单自身。
func (r *RepairRepository) CountHandlerSlotConflict(handlerID uint, date, slot string, excludeID uint) (int64, error) {
	var n int64
	q := r.DB.Model(&model.Repair{}).
		Where("handler_id = ?", handlerID).
		Where("appointment_date = ?", date).
		Where("appointment_slot = ?", slot).
		Where("status NOT IN ?", []string{"done", "closed"})
	if excludeID > 0 {
		q = q.Where("id <> ?", excludeID)
	}
	e := q.Count(&n).Error
	return n, e
}
