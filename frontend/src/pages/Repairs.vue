<template><section><header class="page-head"><div><p class="eyebrow">REPAIR CENTER</p><h2>报修管理</h2></div><el-button type="primary" @click="dialog=true">提交报修</el-button></header><div class="toolbar"><el-select v-model="status" placeholder="全部状态" clearable @change="load"><el-option v-for="(t,k) in repairStatusText" :key="k" :label="t" :value="k"/></el-select></div><div class="repair-list"><RepairCard v-for="v in items" :key="v.id" :repair="v" style="cursor:pointer" @click="openDetail(v)"/><EmptyState v-if="!items.length"/></div>
<el-dialog v-model="dialog" title="提交报修工单"><el-form><el-input v-model="form.title" placeholder="报修标题"/><el-input v-model="form.description" type="textarea" placeholder="问题描述"/><el-select v-model="form.type"><el-option label="水电" value="水电"/><el-option label="家具" value="家具"/><el-option label="公共设施" value="公共设施"/><el-option label="其他" value="其他"/></el-select><el-select v-model="form.visit_date" placeholder="预约日期（未来七天）"><el-option v-for="d in nextDays" :key="d" :label="d" :value="d"/></el-select><el-select v-model="form.time_slot" placeholder="预约时段"><el-option label="上午" value="am"/><el-option label="下午" value="pm"/></el-select></el-form><template #footer><el-button @click="dialog=false">取消</el-button><el-button type="primary" @click="submit">提交</el-button></template></el-dialog>
<el-dialog v-model="detailDialog" title="工单详情"><div v-if="current" class="repair-detail"><h3>{{current.title}} <RepairStatusBadge :status="current.status"/></h3><p>{{current.description}}</p><p>类型：{{current.type}} · 报修人：{{current.user?.nickname||'业主'}}</p><p>预约上门：{{current.visit_date}} {{timeSlotText[current.time_slot]||''}} · 处理人：{{current.handler?.nickname||'待分派'}}</p><p>提交时间：{{new Date(current.created_at).toLocaleString()}}</p>
<template v-if="canReschedule"><el-divider/><h4>修改预约</h4><el-select v-model="rescheduleForm.visit_date" placeholder="预约日期（未来七天）"><el-option v-for="d in nextDays" :key="d" :label="d" :value="d"/></el-select><el-select v-model="rescheduleForm.time_slot" placeholder="预约时段"><el-option label="上午" value="am"/><el-option label="下午" value="pm"/></el-select><el-button type="primary" @click="reschedule">保存预约</el-button></template>
<template v-if="canManage"><el-divider/><h4>物业分派</h4><el-select v-model="assignHandler" placeholder="选择师傅"><el-option v-for="s in staff" :key="s.id" :label="s.nickname" :value="s.id"/></el-select><el-button type="primary" @click="assign">分派</el-button><el-select v-model="nextStatus" placeholder="更新状态"><el-option v-for="(t,k) in repairStatusText" :key="k" :label="t" :value="k"/></el-select><el-button @click="changeStatus">更新状态</el-button></template></div></el-dialog></section></template>
<script setup lang="ts">
import{ref,computed,onMounted}from'vue';
import{ElMessage}from'element-plus';
import{listRepairs,createRepair,assignRepair,rescheduleRepair,updateRepairStatus}from'../api/repair';
import{getStaff}from'../api/user';
import{repairStatusText,timeSlotText}from'../constants/repair';
import{authStore}from'../stores/authStore';
import{usePermission}from'../hooks/usePermission';
import type{Repair,RepairStatus,User}from'../types';
import RepairCard from'../components/common/RepairCard.vue';
import RepairStatusBadge from'../components/common/RepairStatusBadge.vue';
import EmptyState from'../components/common/EmptyState.vue';
const items=ref<Repair[]>([]),status=ref(''),dialog=ref(false);
const blank=()=>({title:'',description:'',type:'水电',images:'',visit_date:'',time_slot:''});
const form=ref(blank());
const detailDialog=ref(false),current=ref<Repair|null>(null);
const staff=ref<User[]>([]),assignHandler=ref<number>(),nextStatus=ref<RepairStatus>();
const rescheduleForm=ref({visit_date:'',time_slot:''});
const canManage=usePermission('repair:manage');
const canReschedule=computed(()=>!!current.value&&authStore.user?.id===current.value.user_id&&current.value.status!=='done'&&current.value.status!=='closed');
const fmt=(d:Date)=>`${d.getFullYear()}-${String(d.getMonth()+1).padStart(2,'0')}-${String(d.getDate()).padStart(2,'0')}`;
const nextDays=computed(()=>Array.from({length:7},(_,i)=>{const d=new Date();d.setDate(d.getDate()+i);return fmt(d)}));
async function load(){items.value=await listRepairs(status.value)}
async function submit(){
  if(!form.value.visit_date||!form.value.time_slot){ElMessage.error('请选择未来七天内的上门日期和上午/下午时段');return}
  try{await createRepair(form.value);ElMessage.success('报修工单已提交');dialog.value=false;form.value=blank();load()}catch(e){ElMessage.error((e as Error).message)}
}
async function openDetail(v:Repair){
  current.value=v;detailDialog.value=true;
  rescheduleForm.value={visit_date:v.visit_date,time_slot:v.time_slot};
  assignHandler.value=v.handler_id;nextStatus.value=v.status;
  if(canManage.value&&!staff.value.length){try{staff.value=await getStaff()}catch(e){ElMessage.error((e as Error).message)}}
}
async function assign(){
  if(!current.value)return;
  if(!assignHandler.value){ElMessage.error('请选择要分派的师傅');return}
  try{current.value=await assignRepair(current.value.id,assignHandler.value);ElMessage.success('已分派');load()}catch(e){ElMessage.error((e as Error).message)}
}
async function changeStatus(){
  if(!current.value||!nextStatus.value)return;
  try{current.value=await updateRepairStatus(current.value.id,nextStatus.value);ElMessage.success('状态已更新');load()}catch(e){ElMessage.error((e as Error).message)}
}
async function reschedule(){
  if(!current.value)return;
  try{current.value=await rescheduleRepair(current.value.id,rescheduleForm.value.visit_date,rescheduleForm.value.time_slot);ElMessage.success('预约已修改');load()}catch(e){ElMessage.error((e as Error).message)}
}
onMounted(load)
</script>
