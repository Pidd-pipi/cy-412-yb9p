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
func (r *RepairRepository) CountOpenByHandlerSlot(handlerID uint, date, slot string, excludeID uint) (int64, error) {
	var n int64
	e := r.DB.Model(&model.Repair{}).Where("handler_id = ? AND visit_date = ? AND time_slot = ? AND status NOT IN ? AND id <> ?", handlerID, date, slot, []string{"done", "closed"}, excludeID).Count(&n).Error
	return n, e
}
