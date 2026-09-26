package constants

const (
	MessageOK             = "ok"
	MessageUnauthorized   = "登录已失效"
	MessageForbidden      = "无权限执行此操作"
	MessageValidation     = "请求参数不合法"
	MessageNotFound       = "资源不存在"
	MessagePaymentSuccess = "支付宝沙箱支付成功"
	MessageRepairCreated  = "报修工单已提交"

	MessageRepairSlotInvalid         = "预约时段只能是上午或下午"
	MessageRepairDateInvalid         = "预约日期格式应为 YYYY-MM-DD"
	MessageRepairDateOutOfWindow     = "预约日期须在未来七天内"
	MessageRepairScheduleConflict    = "该师傅当天同一时段已有未结束工单，请更换师傅或时段"
	MessageRepairRescheduleClosed    = "工单完成或关闭后不能修改预约"
	MessageRepairRescheduleForbidden = "只有报修人可以修改预约"
)
