<template>
	<div class="layout-padding">
		<el-row :gutter="12">
			<el-col :xs="24" :lg="10">
				<el-card shadow="hover">
					<template #header><div class="header"><span>采集引擎</span><el-button :loading="loading" @click="load">刷新</el-button></div></template>
					<el-descriptions :column="1" border>
						<el-descriptions-item label="Worker 数">{{ status.worker_count || 0 }}</el-descriptions-item>
						<el-descriptions-item label="执行中">{{ status.running || 0 }}</el-descriptions-item>
						<el-descriptions-item label="队列长度">{{ status.queue_length || 0 }}</el-descriptions-item>
					</el-descriptions>
					<el-table :data="status.workers || []" class="mt15" max-height="360">
						<el-table-column prop="id" label="Worker" width="90" />
						<el-table-column label="状态" width="90">
							<template #default="{ row }"><el-tag :type="row.running ? 'warning' : 'success'">{{ row.running ? '执行中' : '空闲' }}</el-tag></template>
						</el-table-column>
						<el-table-column prop="current_url" label="当前任务" show-overflow-tooltip />
					</el-table>
				</el-card>
			</el-col>
			<el-col :xs="24" :lg="14">
				<el-card shadow="hover">
					<template #header><div class="header"><span>采集源</span><el-button type="primary" plain :loading="running === '*'" @click="run('')">运行全部</el-button></div></template>
					<el-table :data="providers" v-loading="loading">
						<el-table-column prop="name" label="名称" min-width="140" />
						<el-table-column prop="cron" label="Cron" min-width="160" />
						<el-table-column label="状态" width="90">
							<template #default="{ row }"><el-tag :type="row.enabled ? 'success' : 'info'">{{ row.enabled ? '启用' : '禁用' }}</el-tag></template>
						</el-table-column>
						<el-table-column label="操作" width="100">
							<template #default="{ row }"><el-button text type="primary" :loading="running === row.name" @click="run(row.name)">运行</el-button></template>
						</el-table-column>
					</el-table>
				</el-card>
			</el-col>
		</el-row>
	</div>
</template>

<script setup lang="ts" name="crawlerStatus">
import { onMounted, ref } from 'vue';
import { ElMessage } from 'element-plus';
import { useCrawlerApi } from '/@/api/crawler';
import type { CrawlerProvider, CrawlerStatus } from '/@/utils/business';
const api = useCrawlerApi();
const loading = ref(false);
const running = ref('');
const status = ref<CrawlerStatus>({ worker_count: 0, running: 0, queue_length: 0, workers: [] });
const providers = ref<CrawlerProvider[]>([]);
const load = async () => {
	loading.value = true;
	try {
		const [statusRes, providerRes] = await Promise.all([api.status(), api.providers()]);
		status.value = statusRes.data || {};
		providers.value = providerRes.data || [];
	} finally { loading.value = false; }
};
const run = async (name: string) => {
	running.value = name || '*';
	try {
		await api.run({ name });
		ElMessage.success(name ? `已触发 ${name}` : '已触发全部采集源');
		setTimeout(load, 500);
	} finally { running.value = ''; }
};
onMounted(load);
</script>

<style scoped lang="scss">.header { display: flex; align-items: center; justify-content: space-between; }</style>
