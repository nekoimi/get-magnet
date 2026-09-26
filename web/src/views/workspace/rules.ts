import type { DatasetField,Definition,Template } from './api';

// Keep known template selectors and derive editable defaults for other schemas.
export function datasetDefinition(template:Template,fields:DatasetField[]):Definition {
 const rule:Definition=JSON.parse(JSON.stringify(template.definition));
 const extract=rule.nodes[0];
 const original:any[]=extract.config.fields||[];
 const json=extract.config.content_type==='json';
 const selectors:Record<string,string>={url:'link[rel=canonical]',title:'h1',body:'article',canonical_key:'.number',number:'.number',links:'a[href^="magnet:"]',actress:'.actress',published_at:'time[datetime]'};
 extract.config.fields=fields.map(field=>{
  const existing=original.find(item=>item.name===field.field_key);
  return {
   ...(existing||{}),name:field.field_key,
   selector:existing?.selector||(json?`$.${field.field_key}`:selectors[field.field_key]||`[data-field="${field.field_key}"]`),
   required:field.required,multiple:field.multiple,
   ...(!json&&!existing&&['url','links'].includes(field.field_key)?{attribute:'href'}:{}),
   ...(!json&&!existing&&field.field_key==='published_at'?{attribute:'datetime'}:{}),
   ...(!field.multiple&&['integer','number','boolean'].includes(field.field_type)?{type:field.field_type}:{}),
  };
 });
 // Article-specific transforms/validation cannot refer to absent schema fields.
 if(original.some(item=>!fields.some(field=>field.field_key===item.name)))rule.nodes=[extract];
 return rule;
}

export function editableDefinition(raw:string|Definition):Definition {
 const value=typeof raw==='string'?JSON.parse(raw):JSON.parse(JSON.stringify(raw));
 if(!value||typeof value!=='object'||Array.isArray(value)||!value.trigger||typeof value.trigger!=='object'||Array.isArray(value.trigger)||!Array.isArray(value.nodes))throw new Error('定义必须包含 trigger 对象和 nodes 数组');
 if(value.nodes.some((node:any)=>!node||typeof node!=='object'||typeof node.type!=='string'||!node.config||typeof node.config!=='object'||Array.isArray(node.config)))throw new Error('每个节点需要 type 和 config 对象');
 const extract=value.nodes.find((node:any)=>node.type==='extract');
 if(value.listing!==undefined&&(!value.listing||typeof value.listing!=='object'||Array.isArray(value.listing)||typeof value.listing.detail_selector!=='string'||(value.listing.next_selector!==undefined&&typeof value.listing.next_selector!=='string')||!Number.isInteger(value.listing.max_pages)||!Number.isInteger(value.listing.max_empty_pages)))throw new Error('listing 需要详情 CSS、可选下一页 CSS 和整数页数上限');
 if(extract&&(!Array.isArray(extract.config.fields)||extract.config.fields.some((field:any)=>!field||typeof field.name!=='string'||typeof field.selector!=='string')))throw new Error('提取字段需要 name 和 selector');
 if(!value.trigger.fetch)value.trigger.fetch={mode:'http'};
 if(typeof value.trigger.fetch!=='object'||Array.isArray(value.trigger.fetch))throw new Error('trigger.fetch 必须是对象');
 if(!value.trigger.fetch.mode)value.trigger.fetch.mode='http';
 return value;
}
