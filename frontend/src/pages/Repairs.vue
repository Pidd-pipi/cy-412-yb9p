<template>
  <section>
    <header class="page-head">
      <div>
        <p class="eyebrow">REPAIR CENTER</p>
        <h2>报修管理</h2>
        <p>提交工单时选择未来七天内的上门日期与上午 / 下午时段，物业按师傅排班分派，同一师傅同一时段不重复派单。</p>
      </div>
      <el-button type="primary" @click="openCreate">提交报修</el-button>
    </header>

    <div class="toolbar">
      <el-select v-model="status" placeholder="全部状态" clearable style="width:160px" @change="load">
        <el-option v-for="(t,k) in repairStatusText" :key="k" :label="t" :value="k"/>
      </el-select>
    </div>

    <div class="repair-list">
      <RepairCard v-for="v in items" :key="v.id" :repair="v" @open="openDetail"/>
      <EmptyState v-if="!items.length"/>
    </div>

    <!-- 提交报修 -->
    <el-dialog v-model="createDialog" title="提交报修工单" width="520px">
      <el-form :model="form" label-width="84px">
        <el-form-item label="标题">
          <el-input v-model="form.title" placeholder="报修标题"/>
        </el-form-item>
        <el-form-item label="问题描述">
          <el-input v-model="form.description" type="textarea" placeholder="请描述故障现象与位置"/>
        </el-form-item>
        <el-form-item label="报修类型">
          <el-select v-model="form.type">
            <el-option label="水电" value="水电"/>
            <el-option label="家具" value="家具"/>
            <el-option label="公共设施" value="公共设施"/>
            <el-option label="其他" value="其他"/>
          </el-select>
        </el-form-item>
        <el-form-item label="上门日期" required>
          <el-date-picker
            v-model="form.appointment_date"
            type="date"
            value-format="YYYY-MM-DD"
            :disabled-date="disabledWindowDate"
            :clearable="false"
            placeholder="请选择未来七天内日期"
          />
        </el-form-item>
        <el-form-item label="上门时段" required>
          <el-radio-group v-model="form.appointment_slot">
            <el-radio value="morning">上午</el-radio>
            <el-radio value="afternoon">下午</el-radio>
          </el-radio-group>
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="createDialog=false">取消</el-button>
        <el-button type="primary" @click="submit">提交</el-button>
      </template>
    </el-dialog>

    <!-- 工单详情 -->
    <el-dialog v-model="detailDialog" title="工单详情" width="600px">
      <template v-if="detail">
        <div class="detail-head">
          <h3>{{detail.title}}</h3>
          <RepairStatusBadge :status="detail.status"/>
        </div>
        <el-descriptions :column="2" border>
          <el-descriptions-item label="类型">{{detail.type}}</el-descriptions-item>
          <el-descriptions-item label="报修人">{{detail.user?.nickname||'业主'}}</el-descriptions-item>
          <el-descriptions-item label="预约日期">{{detail.appointment_date||'—'}}</el-descriptions-item>
          <el-descriptions-item label="预约时段">{{repairSlotText[detail.appointment_slot]||'—'}}</el-descriptions-item>
          <el-descriptions-item label="处理人">{{detail.handler?.nickname||'暂未分派'}}</el-descriptions-item>
          <el-descriptions-item label="提交时间">{{new Date(detail.created_at).toLocaleString()}}</el-descriptions-item>
          <el-descriptions-item label="问题描述" :span="2">{{detail.description}}</el-descriptions-item>
        </el-descriptions>

        <!-- 物业分派：同师傅同一天同一时段存在未结束工单时后端拒绝，原分派保持不变 -->
        <div v-if="canManage" class="detail-action">
          <h4>分派处理人</h4>
          <div class="action-row">
            <el-select v-model="assignHandlerId" placeholder="选择师傅" filterable style="width:220px">
              <el-option v-for="s in staff" :key="s.id" :label="`${s.nickname}（${roleText[s.role]}）`" :value="s.id"/>
            </el-select>
            <el-button type="primary" :loading="saving" @click="doAssign">分派</el-button>
          </div>
        </div>

        <!-- 物业更新进度 -->
        <div v-if="canManage" class="detail-action">
          <h4>更新进度</h4>
          <div class="action-row">
            <el-select v-model="nextStatus" style="width:160px">
              <el-option v-for="(t,k) in repairStatusText" :key="k" :label="t" :value="k"/>
            </el-select>
            <el-select v-if="nextStatus==='done'" v-model="nextRating" placeholder="评价星级" style="width:120px">
              <el-option v-for="n in 5" :key="n" :label="`${n} 星`" :value="n"/>
            </el-select>
            <el-button :loading="saving" @click="doStatus">更新状态</el-button>
          </div>
        </div>

        <!-- 报修人改预约：完成或关闭前可用，与处理人其他未结束工单冲突时拒绝 -->
        <div v-if="canReschedule" class="detail-action">
          <h4>修改预约</h4>
          <div class="action-row">
            <el-date-picker
              v-model="rescheduleDate"
              type="date"
              value-format="YYYY-MM-DD"
              :disabled-date="disabledWindowDate"
              :clearable="false"
            />
            <el-radio-group v-model="rescheduleSlot">
              <el-radio value="morning">上午</el-radio>
              <el-radio value="afternoon">下午</el-radio>
            </el-radio-group>
            <el-button type="primary" :loading="saving" @click="doReschedule">保存预约</el-button>
          </div>
        </div>
        <el-alert
          v-else-if="isOwner && detailFinal"
          class="detail-action"
          type="info"
          :closable="false"
          title="工单已完成或关闭，预约时段已释放，不能再修改预约。"
        />
      </template>
    </el-dialog>
  </section>
</template>

<script setup lang="ts">
import {computed, onMounted, ref} from 'vue';
import {ElMessage} from 'element-plus';
import {listRepairs, createRepair, assignRepair, rescheduleRepair, updateRepairStatus} from '../api/repair';
import {getStaff} from '../api/user';
import {repairStatusText, repairSlotText} from '../constants/repair';
import {roleText} from '../utils/roleText';
import {usePermission} from '../hooks/usePermission';
import {authStore} from '../stores/authStore';
import type {Repair, RepairSlot, RepairStatus, User} from '../types';
import RepairCard from '../components/common/RepairCard.vue';
import EmptyState from '../components/common/EmptyState.vue';
import RepairStatusBadge from '../components/common/RepairStatusBadge.vue';

const items = ref<Repair[]>([]);
const status = ref('');

const createDialog = ref(false);
const form = ref({title: '', description: '', type: '水电', images: '', appointment_date: '', appointment_slot: 'morning' as RepairSlot});

const detailDialog = ref(false);
const detail = ref<Repair | null>(null);
const saving = ref(false);

const canManage = usePermission('repair:manage');
const staff = ref<User[]>([]);
const assignHandlerId = ref<number>();
const nextStatus = ref<RepairStatus>('processing');
const nextRating = ref<number>();

const rescheduleDate = ref('');
const rescheduleSlot = ref<RepairSlot>('morning');

const isOwner = computed(() => !!detail.value && detail.value.user_id === authStore.user?.id);
const detailFinal = computed(() => !!detail.value && ['done', 'closed'].includes(detail.value.status));
const canReschedule = computed(() => isOwner.value && !detailFinal.value);

function toISODate(d: Date): string {
  const m = `${d.getMonth() + 1}`.padStart(2, '0');
  const day = `${d.getDate()}`.padStart(2, '0');
  return `${d.getFullYear()}-${m}-${day}`;
}
// 仅允许选择今天起未来七天（含今天）。
function disabledWindowDate(d: Date): boolean {
  const today = new Date();
  today.setHours(0, 0, 0, 0);
  const last = new Date(today);
  last.setDate(last.getDate() + 6);
  return d.getTime() < today.getTime() || d.getTime() > last.getTime();
}

async function load() {
  items.value = await listRepairs(status.value);
}

function resetForm() {
  form.value = {title: '', description: '', type: '水电', images: '', appointment_date: toISODate(new Date()), appointment_slot: 'morning'};
}
function openCreate() {
  resetForm();
  createDialog.value = true;
}
async function submit() {
  if (!form.value.appointment_date) {
    ElMessage.warning('请选择上门日期');
    return;
  }
  try {
    await createRepair(form.value);
    ElMessage.success('报修工单已提交');
    createDialog.value = false;
    await load();
  } catch (e) {
    ElMessage.error((e as Error).message);
  }
}

async function openDetail(id: number) {
  const v = items.value.find(x => x.id === id);
  detail.value = v ?? null;
  if (!v) return;
  detailDialog.value = true;
  assignHandlerId.value = v.handler_id;
  nextStatus.value = v.status;
  nextRating.value = v.rating || undefined;
  rescheduleDate.value = v.appointment_date || toISODate(new Date());
  rescheduleSlot.value = v.appointment_slot || 'morning';
  if (canManage.value && !staff.value.length) {
    try {
      staff.value = await getStaff();
    } catch (e) {
      ElMessage.error((e as Error).message);
    }
  }
}

async function doAssign() {
  if (!detail.value || !assignHandlerId.value) {
    ElMessage.warning('请选择师傅');
    return;
  }
  saving.value = true;
  try {
    detail.value = await assignRepair(detail.value.id, assignHandlerId.value);
    ElMessage.success('分派成功');
    await load();
  } catch (e) {
    // 冲突等拒绝原因由后端文案提示，原分派保持不变。
    ElMessage.error((e as Error).message);
    assignHandlerId.value = detail.value.handler_id;
  } finally {
    saving.value = false;
  }
}

async function doStatus() {
  if (!detail.value) return;
  saving.value = true;
  try {
    detail.value = await updateRepairStatus(detail.value.id, nextStatus.value, nextRating.value);
    ElMessage.success('工单状态已更新');
    await load();
  } catch (e) {
    ElMessage.error((e as Error).message);
  } finally {
    saving.value = false;
  }
}

async function doReschedule() {
  if (!detail.value || !rescheduleDate.value) {
    ElMessage.warning('请选择预约日期');
    return;
  }
  saving.value = true;
  try {
    detail.value = await rescheduleRepair(detail.value.id, {
      appointment_date: rescheduleDate.value,
      appointment_slot: rescheduleSlot.value,
    });
    ElMessage.success('预约已更新');
    await load();
  } catch (e) {
    ElMessage.error((e as Error).message);
  } finally {
    saving.value = false;
  }
}

onMounted(load);
</script>

<style scoped>
.detail-head {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 14px;
}
.detail-head h3 {
  margin: 0;
}
.detail-action {
  margin-top: 18px;
}
.detail-action h4 {
  margin: 0 0 10px;
  font-size: 14px;
  color: #3d4a61;
}
.action-row {
  display: flex;
  flex-wrap: wrap;
  gap: 10px;
  align-items: center;
}
.repair-appointment {
  display: flex;
  align-items: center;
  gap: 8px;
  margin-top: 8px;
  color: #5b6980;
}
</style>
