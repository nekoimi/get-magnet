import { ref } from 'vue';
export const projectId=ref<number>(Number(localStorage.getItem('scrapio.project.id'))||0);
export const projectRefresh=ref(0);
export function selectProject(id:number) { projectId.value=id; localStorage.setItem('scrapio.project.id',String(id)); }
export function refreshProjects(){projectRefresh.value++;}
