<template>
	<div class="layout-padding">
		<el-card shadow="hover">
			<template #header>
				<div class="header">
					<span>运行时配置（只读）</span>
					<el-button :loading="loading" @click="load">刷新</el-button>
				</div>
			</template>
			<el-alert title="敏感字段已由后端脱敏，配置修改后需通过部署环境重启应用生效。" type="info" show-icon class="mb15" />
			<el-collapse v-model="activeNames">
				<el-collapse-item v-for="(values, group) in config" :key="group" :title="groupLabels[group] || group" :name="group">
					<el-descriptions :column="2" border>
						<el-descriptions-item v-for="(value, key) in flatten(values)" :key="key" :label="key">
							{{ format(value) }}
						</el-descriptions-item>
					</el-descriptions>
					<div v-if="testActions[group]" class="test-action">
						<el-button type="primary" plain :loading="testing === group" @click="test(group)">
							测试连接
						</el-button>
					</div>
				</el-collapse-item>
			</el-collapse>
		</el-card>
	</div>
</template>

<script setup lang="ts" name="settingsRuntime">
import { onMounted, ref } from 'vue';
import { ElMessage } from 'element-plus';
import { useSettingsApi } from '/@/api/settings';

const api = useSettingsApi();
const loading = ref(false);
const testing = ref('');
const config = ref<Record<string, any>>({});
const activeNames = ref<string[]>(['app', 'download', 'cloud_driver']);
const groupLabels: Record<string, string> = {
	port: '应用端口', app: '应用配置', log_level: '日志级别', log_dir: '日志目录', jwt_secret: 'JWT',
	download: '下载配置', cloud_driver: '网盘配置', strm: 'STRM 配置', crawler: '爬虫配置',
	aria2: 'aria2 配置', db: '数据库配置',
};
const testActions: Record<string, () => Promise<any>> = {
	cloud_driver: api.testCloudDriver,
	aria2: api.testAria2,
	crawler: api.testDrissionRod,
};

const flatten = (value: any, prefix = ''): Record<string, any> => {
	if (value === null || typeof value !== 'object' || Array.isArray(value)) return { [prefix || 'value']: value };
	return Object.entries(value).reduce((result, [key, child]) => {
		const name = prefix ? `${prefix}.${key}` : key;
		return Object.assign(result, child !== null && typeof child === 'object' ? flatten(child, name) : { [name]: child });
	}, {});
};
const format = (value: any) => value === '' || value === null || value === undefined ? '-' : String(value);
const load = async () => {
	loading.value = true;
	try {
		const res = await api.list();
		config.value = res.data || {};
	} finally {
		loading.value = false;
	}
};
const test = async (group: string) => {
	testing.value = group;
	try {
		const res = await testActions[group]();
		const result = res.data;
		result.ok ? ElMessage.success(`连接正常，耗时 ${result.latency_ms}ms`) : ElMessage.error(result.message || '连接失败');
	} finally {
		testing.value = '';
	}
};
onMounted(load);
</script>

<style scoped lang="scss">
.header { display: flex; align-items: center; justify-content: space-between; }
.test-action { margin-top: 12px; }
</style>
