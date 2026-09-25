<template>
	<div class="layout-padding">
		<el-row :gutter="12">
			<el-col :xs="24" :lg="16">
				<el-card shadow="hover">
					<template #header><div class="header"><span>服务健康</span><el-button :loading="loading" @click="load">刷新</el-button></div></template>
					<el-table :data="services" v-loading="loading">
						<el-table-column prop="name" label="服务" min-width="140" />
						<el-table-column label="状态" width="100">
							<template #default="{ row }"><el-tag :type="row.ok ? 'success' : 'danger'">{{ row.ok ? '正常' : '异常' }}</el-tag></template>
						</el-table-column>
						<el-table-column prop="latency_ms" label="耗时(ms)" width="110" />
						<el-table-column prop="message" label="详情" min-width="220" show-overflow-tooltip />
					</el-table>
				</el-card>
			</el-col>
			<el-col :xs="24" :lg="8">
				<el-card shadow="hover">
					<template #header><span>版本信息</span></template>
					<el-descriptions :column="1" border>
						<el-descriptions-item label="版本">{{ version.version || '-' }}</el-descriptions-item>
						<el-descriptions-item label="Commit">{{ version.commit || '-' }}</el-descriptions-item>
						<el-descriptions-item label="Go">{{ version.go_version || '-' }}</el-descriptions-item>
						<el-descriptions-item label="启动时间">{{ formatDateTime(version.started_at) }}</el-descriptions-item>
						<el-descriptions-item label="运行时长">{{ uptime }}</el-descriptions-item>
					</el-descriptions>
				</el-card>
			</el-col>
		</el-row>
	</div>
</template>

<script setup lang="ts" name="opsHealth">
import { computed, onMounted, ref } from 'vue';
import { useOpsApi } from '/@/api/ops';
import { formatDateTime } from '/@/utils/business';
import type { ServiceHealth, VersionInfo } from '/@/utils/business';
const api = useOpsApi();
const loading = ref(false);
const services = ref<Array<ServiceHealth & { name: string }>>([]);
const version = ref<VersionInfo>({});
const uptime = computed(() => `${Math.floor((version.value.uptime_seconds || 0) / 86400)}天 ${Math.floor(((version.value.uptime_seconds || 0) % 86400) / 3600)}小时`);
const load = async () => {
	loading.value = true;
	try {
		const [healthRes, versionRes] = await Promise.all([api.health(), api.version()]);
		const healthServices = (healthRes.data?.services || {}) as Record<string, ServiceHealth>;
		services.value = Object.entries(healthServices).map(([name, value]) => ({ name, ...value }));
		version.value = versionRes.data || {};
	} finally { loading.value = false; }
};
onMounted(load);
</script>
<style scoped lang="scss">.header { display: flex; align-items: center; justify-content: space-between; }</style>
