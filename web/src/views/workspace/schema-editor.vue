<template>
 <div>
  <div class="v21-field"><label for="empty-policy">空值更新</label><select id="empty-policy" v-model="model.empty_value_policy"><option value="preserve">保留已有非空值</option><option value="overwrite">允许空值覆盖</option></select></div>
  <div class="fields-title"><strong>字段结构</strong><el-button size="small" @click="addField">添加字段</el-button></div>
  <div v-for="(field,index) in model.fields" :key="index" class="field-editor">
   <input v-model="field.key" :aria-label="`字段 ${index+1} 的键`" placeholder="字段键" />
   <input v-model="field.label" :aria-label="`字段 ${index+1} 的名称`" placeholder="显示名称" />
   <select v-model="field.type" :aria-label="`字段 ${index+1} 的类型`"><option v-for="kind in types" :key="kind">{{kind}}</option></select>
   <label><input v-model="field.required" type="checkbox" />必填</label><label><input v-model="field.multiple" type="checkbox" />多值</label>
   <button class="v21-link text-button" @click="model.fields.splice(index,1)">删除</button>
  </div>
  <div class="v21-field"><label>唯一键字段</label><el-select v-model="model.unique_key_fields" multiple placeholder="选择必填的单值字段" aria-label="唯一键字段" style="width:100%"><el-option v-for="field in keyCandidates" :key="field.key" :value="field.key" :label="field.label||field.key" /></el-select><small class="v21-muted">按选择顺序组合成唯一键；键字段必须必填、单值，且不是 JSON 类型。</small></div>
 </div>
</template>
<script setup lang="ts">
import { computed,watch } from 'vue';
import type { DatasetSchema } from './api';
const props=defineProps<{model:DatasetSchema}>();
const types=['string','integer','number','boolean','datetime','url','json'];
const keyCandidates=computed(()=>props.model.fields.filter(f=>f.key.trim()&&f.required&&!f.multiple&&f.type!=='json'));
watch(keyCandidates,candidates=>{props.model.unique_key_fields=props.model.unique_key_fields.filter(key=>candidates.some(f=>f.key===key));},{deep:true});
function addField(){props.model.fields.push({key:'',label:'',type:'string',required:false,multiple:false});}
</script>
<style scoped>
.fields-title{display:flex;align-items:center;justify-content:space-between;gap:15px;margin-bottom:15px}
.field-editor{display:grid;grid-template-columns:1fr 1fr 105px 65px 65px 45px;gap:6px;align-items:center;margin-bottom:7px}
.field-editor input:not([type=checkbox]),.field-editor select{min-width:0;border:1px solid #cfd8e4;border-radius:7px;padding:8px}
.field-editor label{font-size:12px;white-space:nowrap}.text-button{border:0;background:transparent;padding:0}
@media(max-width:760px){.field-editor{grid-template-columns:1fr 1fr 100px}.field-editor>*{min-width:0}}
</style>
