import {request} from '../utils/request';
import type{Repair,RepairStatus,RepairSlot}from '../types';

export interface RepairAppointmentInput{appointment_date:string;appointment_slot:RepairSlot}

export const listRepairs=(status?:string)=>request<Repair[]>(`/repairs${status?`?status=${status}`:''}`);
export const createRepair=(data:Pick<Repair,'title'|'description'|'type'|'images'>&RepairAppointmentInput)=>request<Repair>('/repairs',{method:'POST',body:JSON.stringify(data)});
export const assignRepair=(id:number,handler_id:number)=>request<Repair>(`/repairs/${id}/assign`,{method:'PATCH',body:JSON.stringify({handler_id})});
export const rescheduleRepair=(id:number,data:RepairAppointmentInput)=>request<Repair>(`/repairs/${id}/appointment`,{method:'PATCH',body:JSON.stringify(data)});
export const updateRepairStatus=(id:number,status:RepairStatus,rating?:number)=>request<Repair>(`/repairs/${id}/status`,{method:'PATCH',body:JSON.stringify(rating?{status,rating}:{status})});
