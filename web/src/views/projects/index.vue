<template>
  <div class="a01-page">
    <header class="a01-header">
      <div><div class="eyebrow">SCRAPIO / WORKSPACE</div><h1>项目与数据集</h1><p>按采集目标组织数据集和采集器。数据集定义唯一键、字段及空值更新方式。</p></div>
      <div class="header-actions"><el-button @click="openProject()">新建项目</el-button><el-button :disabled="!currentProject" @click="openProject(currentProject)">编辑项目</el-button><el-button type="primary" :disabled="!selectedProject" @click="openDataset()">新建数据集</el-button></div>
    </header>
    <el-alert v-if="error" :title="error" type="error" show-icon :closable="false" class="notice" />
    <div class="summary-grid">
      <div class="summary-card"><span>当前项目</span><strong>{{ currentProject?.name || '选择项目' }}</strong><small>{{ currentProject?.goal || '为采集目标创建工作空间' }}</small></div>
      <div class="summary-card"><span>数据集</span><strong>{{ filteredDatasets.length }}</strong><small>定义字段与唯一键</small></div>
      <div class="summary-card"><span>采集器</span><strong>{{ filteredWorkflows.length }}</strong><small>关联到当前项目</small></div>
    </div>
    <div class="workspace-grid">
      <section class="panel project-panel"><div class="panel-head"><h2>项目</h2><el-button text @click="reload">刷新</el-button></div>
        <div v-if="loading" class="empty">加载中…</div><div v-else-if="!projects.length" class="empty">还没有项目，先新建一个项目。</div>
        <button v-for="project in projects" :key="project.id" class="project-row" :class="{ active: selectedProject === project.id }" @click="selectedProject = project.id">
          <span class="project-mark">{{ project.name.slice(0, 1) }}</span><span><strong>{{ project.name }}</strong><small>{{ project.code }} · {{ project.status }}</small></span>
        </button>
      </section>
      <div class="main-column">
        <section class="panel"><div class="panel-head"><div><h2>数据集</h2><p>相同类型的记录可以由多个采集器提供</p></div></div>
          <div v-if="!filteredDatasets.length" class="empty">当前项目还没有数据集。创建后可设置字段和唯一键。</div>
          <div v-else class="dataset-grid"><button v-for="dataset in filteredDatasets" :key="dataset.id" class="dataset-card" @click="openDetail(dataset)"><span class="dataset-icon">{{ dataset.name.slice(0, 1) }}</span><strong>{{ dataset.name }}</strong><small>{{ dataset.record_type }} · Schema v{{ dataset.schema_version }}</small><span class="dataset-link">查看字段 →</span></button></div>
        </section>
        <section class="panel"><div class="panel-head"><div><h2>采集器归属</h2><p>旧工作流已归入默认项目；未关联数据集的历史定义仍可查询</p></div></div>
          <div v-if="!filteredWorkflows.length" class="empty">当前项目暂无采集器。</div>
          <el-table v-else :data="filteredWorkflows" stripe><el-table-column prop="name" label="采集器" min-width="160" /><el-table-column prop="code" label="标识" min-width="130" /><el-table-column label="数据集" min-width="150"><template #default="scope">{{ datasetName(scope.row.dataset_id) }}</template></el-table-column><el-table-column prop="resource_type" label="记录类型" width="120" /></el-table>
        </section>
      </div>
    </div>
    <el-dialog v-model="projectDialog" :title="editingProject ? '编辑项目' : '新建项目'" width="500px"><el-form label-position="top"><el-form-item label="项目名称"><el-input v-model="projectForm.name" /></el-form-item><el-form-item v-if="!editingProject" label="项目标识"><el-input v-model="projectForm.code" placeholder="例如 research" /></el-form-item><el-form-item label="目标"><el-input v-model="projectForm.goal" type="textarea" /></el-form-item><el-form-item label="负责人"><el-input v-model="projectForm.owner" /></el-form-item><el-form-item v-if="editingProject" label="状态"><el-select v-model="projectForm.status"><el-option label="运行中" value="active" /><el-option label="已暂停" value="paused" /><el-option label="已归档" value="archived" /></el-select></el-form-item></el-form><template #footer><el-button @click="projectDialog = false">取消</el-button><el-button type="primary" :loading="saving" @click="saveProject">保存</el-button></template></el-dialog>
    <el-dialog v-model="datasetDialog" :title="editingDataset ? '更新数据集 Schema' : '新建数据集'" width="720px"><el-form label-position="top"><div v-if="!editingDataset" class="form-grid"><el-form-item label="数据集名称"><el-input v-model="datasetForm.name" /></el-form-item><el-form-item label="标识"><el-input v-model="datasetForm.code" placeholder="例如 article" /></el-form-item><el-form-item label="记录类型"><el-input v-model="datasetForm.record_type" /></el-form-item><el-form-item label="空值更新"><el-select v-model="datasetForm.empty_value_policy"><el-option label="保留现有非空值" value="preserve" /><el-option label="允许空值覆盖" value="overwrite" /></el-select></el-form-item></div>
      <el-form-item v-else label="空值更新"><el-select v-model="datasetForm.empty_value_policy"><el-option label="保留现有非空值" value="preserve" /><el-option label="允许空值覆盖" value="overwrite" /></el-select></el-form-item>
      <div class="field-head"><strong>字段</strong><el-button text type="primary" @click="addField">添加字段</el-button></div>
      <div v-for="(field, index) in datasetForm.fields" :key="index" class="field-row"><el-input v-model="field.key" placeholder="字段键" /><el-input v-model="field.label" placeholder="显示名称" /><el-select v-model="field.type"><el-option v-for="type in fieldTypes" :key="type" :label="type" :value="type" /></el-select><el-checkbox v-model="field.required">必填</el-checkbox><el-checkbox v-model="field.multiple">多值</el-checkbox><el-button text type="danger" @click="datasetForm.fields.splice(index, 1)">移除</el-button></div>
      <el-form-item label="唯一键字段"><el-select v-model="datasetForm.unique_key_fields" multiple placeholder="选择必填且非多值字段" style="width:100%"><el-option v-for="field in keyCandidates" :key="field.key" :label="field.label || field.key" :value="field.key" /></el-select></el-form-item>
      <p class="hint">保存 Schema 会生成新版本，历史字段版本保留供后续运行追溯。</p>
      </el-form><template #footer><el-button @click="datasetDialog = false">取消</el-button><el-button type="primary" :loading="saving" @click="saveDataset">保存</el-button></template></el-dialog>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, reactive, ref, watch } from 'vue';
import { ElMessage } from 'element-plus';
import request from '/@/utils/request';

interface Project { id:number; code:string; name:string; goal:string; owner:string; status:string }
interface Dataset { id:number; project_id:number; code:string; name:string; record_type:string; schema_version:number; unique_key_fields:string; empty_value_policy:string }
interface Field { key:string; label:string; type:string; required:boolean; multiple:boolean }
interface Workflow { id:number; project_id:number; dataset_id?:number; name:string; code:string; resource_type:string }
const projects = ref<Project[]>([]), datasets = ref<Dataset[]>([]), workflows = ref<Workflow[]>([]);
const selectedProject = ref<number>(0), loading = ref(false), saving = ref(false), error = ref('');
const projectDialog = ref(false), datasetDialog = ref(false), editingDataset = ref<Dataset | null>(null), editingProject = ref<Project | null>(null);
const projectForm = reactive({ code:'', name:'', goal:'', owner:'', status:'active' });
const datasetForm = reactive({ code:'', name:'', record_type:'', empty_value_policy:'preserve', unique_key_fields:[] as string[], fields:[] as Field[] });
const fieldTypes = ['string','integer','number','boolean','datetime','url','json'];
const currentProject = computed(() => projects.value.find(item => item.id === selectedProject.value));
const filteredDatasets = computed(() => datasets.value.filter(item => item.project_id === selectedProject.value));
const filteredWorkflows = computed(() => workflows.value.filter(item => item.project_id === selectedProject.value));
const keyCandidates = computed(() => datasetForm.fields.filter(item => item.required && !item.multiple && item.type !== 'json'));
watch(keyCandidates, candidates => { datasetForm.unique_key_fields = datasetForm.unique_key_fields.filter(key => candidates.some(item => item.key === key)); });
function datasetName(id?:number) { return datasets.value.find(item => item.id === id)?.name || '未关联'; }
function apiError(err:any) { return err?.msg || err?.message || '请求失败'; }
async function reload() { loading.value = true; error.value = ''; try { const [p,d,w]:any[] = await Promise.all([request({url:'/api/v2/projects/list',method:'get'}),request({url:'/api/v2/datasets/list',method:'get'}),request({url:'/api/v2/workflows/list',method:'get',params:{page:1,size:500}})]); projects.value=p.data || []; datasets.value=d.data || []; workflows.value=w.data?.list || []; if (!projects.value.some(item => item.id === selectedProject.value)) selectedProject.value=projects.value[0]?.id || 0; } catch (err:any) { error.value=apiError(err); } finally { loading.value=false; } }
function openProject(project?:Project) { editingProject.value=project || null; Object.assign(projectForm,project ? {code:project.code,name:project.name,goal:project.goal,owner:project.owner,status:project.status} : {code:'',name:'',goal:'',owner:'',status:'active'}); projectDialog.value=true; }
async function saveProject() { saving.value=true; try { const result:any=await request({url:editingProject.value?'/api/v2/projects/update':'/api/v2/projects/create',method:'post',data:editingProject.value?{id:editingProject.value.id,...projectForm}:projectForm}); projectDialog.value=false; await reload(); selectedProject.value=result.data.id; ElMessage.success('项目已保存'); } catch(err:any) { ElMessage.error(apiError(err)); } finally { saving.value=false; } }
function addField() { datasetForm.fields.push({key:'',label:'',type:'string',required:false,multiple:false}); }
function openDataset() { editingDataset.value=null; Object.assign(datasetForm,{code:'',name:'',record_type:'',empty_value_policy:'preserve',unique_key_fields:[],fields:[{key:'',label:'',type:'string',required:true,multiple:false}]}); datasetDialog.value=true; }
async function openDetail(dataset:Dataset) { try { const result:any=await request({url:'/api/v2/datasets/detail',method:'get',params:{id:dataset.id}}); editingDataset.value=dataset; Object.assign(datasetForm,{code:dataset.code,name:dataset.name,record_type:dataset.record_type,empty_value_policy:dataset.empty_value_policy,unique_key_fields:JSON.parse(dataset.unique_key_fields),fields:(result.data.fields || []).map((item:any) => ({key:item.field_key,label:item.label,type:item.field_type,required:item.required,multiple:item.multiple}))}); datasetDialog.value=true; } catch(err:any) { ElMessage.error(apiError(err)); } }
async function saveDataset() { saving.value=true; try { const payload={...datasetForm,project_id:selectedProject.value}; await request({url:editingDataset.value?'/api/v2/datasets/schema/update':'/api/v2/datasets/create',method:'post',data:editingDataset.value?{id:editingDataset.value.id,...payload}:payload}); datasetDialog.value=false; await reload(); ElMessage.success('数据集已保存'); } catch(err:any) { ElMessage.error(apiError(err)); } finally { saving.value=false; } }
onMounted(reload);
</script>

<style scoped>
.a01-page{--ink:#202a46;--muted:#66738a;--line:#e2e7f0;--brand:#4059ad;--canvas:#f5f7fb;padding:30px 36px 70px;background:var(--canvas);min-height:100%;color:var(--ink)}.a01-header{display:flex;justify-content:space-between;gap:20px;align-items:start;margin-bottom:25px}.eyebrow{font-size:11px;letter-spacing:.13em;color:var(--brand);font-weight:800}.a01-header h1{font-size:27px;margin:5px 0}.a01-header p,.panel-head p{font-size:13px;color:var(--muted);margin:0}.header-actions{display:flex;gap:8px}.notice{margin-bottom:16px}.summary-grid{display:grid;grid-template-columns:repeat(3,1fr);gap:16px;margin-bottom:18px}.summary-card,.panel{background:white;border:1px solid var(--line);border-radius:13px;box-shadow:0 10px 30px rgba(28,30,84,.05)}.summary-card{padding:20px;display:grid;gap:5px}.summary-card span,.summary-card small,.dataset-card small{color:var(--muted);font-size:12px}.summary-card strong{font-size:23px}.workspace-grid{display:grid;grid-template-columns:250px minmax(0,1fr);gap:18px}.main-column{display:grid;align-content:start;gap:18px}.panel{padding:21px;min-width:0}.panel-head{display:flex;justify-content:space-between;align-items:center;margin-bottom:18px}.panel-head h2{margin:0 0 3px;font-size:16px}.project-row{border:0;background:transparent;width:100%;display:flex;align-items:center;gap:10px;text-align:left;padding:10px;border-radius:9px;cursor:pointer;color:var(--ink)}.project-row.active,.project-row:hover{background:#edf1fc}.project-row strong,.project-row small{display:block}.project-row small{color:var(--muted);font-size:11px}.project-mark,.dataset-icon{display:grid;place-items:center;background:#dfe6fa;color:var(--brand);border-radius:9px;font-weight:800}.project-mark{width:35px;height:35px}.dataset-icon{width:42px;height:42px}.dataset-grid{display:grid;grid-template-columns:repeat(auto-fill,minmax(195px,1fr));gap:12px}.dataset-card{border:1px solid var(--line);background:#fff;border-radius:10px;padding:16px;min-height:145px;display:grid;justify-items:start;gap:5px;cursor:pointer;color:var(--ink);text-align:left}.dataset-card:hover{border-color:var(--brand)}.dataset-link{font-size:12px;color:var(--brand);font-weight:700}.empty{padding:34px;text-align:center;color:var(--muted);background:#f8f9fc;border-radius:9px}.form-grid{display:grid;grid-template-columns:1fr 1fr;gap:0 16px}.field-head{display:flex;justify-content:space-between;align-items:center;margin:8px 0}.field-row{display:grid;grid-template-columns:1fr 1fr 110px 65px 65px 48px;gap:7px;align-items:center;margin-bottom:8px}.field-row .el-checkbox{margin-right:0}.hint{color:var(--muted);font-size:12px}@media(max-width:850px){.a01-page{padding:20px}.a01-header{flex-direction:column}.summary-grid{grid-template-columns:1fr}.workspace-grid{grid-template-columns:1fr}.form-grid{grid-template-columns:1fr}.field-row{grid-template-columns:1fr 1fr}.field-row>*{min-width:0}}
</style>
