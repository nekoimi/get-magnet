<template>
	<div class="layout-padding">
		<el-card shadow="hover">
			<template #header><div class="header"><span>调度任务</span><el-button :loading="loading" @click="load">刷新</el-button></div></template>
			<el-table :data="jobs" v-loading="loading">
				<el-table-column prop="name" label="任务名" min-width="180" />
				<el-table-column prop="cron" label="Cron" width="150" />
				<el-table-column label="下次执行" min-width="180"><template #default="{ row }">{{ formatDateTime(row.next_run) }}</template></el-table-column>
				<el-table-column label="最近执行" min-width="180"><template #default="{ row }">{{ formatDateTime(row.last_run) }}</template></el-table-column>
				<el-table-column label="结果" width="100">
					<template #default="{ row }">
						<el-tag v-if="row.status" :type="row.status === 'success' ? 'success' : 'danger'">{{ row.status }}</el-tag>
						<span v-else>-</span>
					</template>
				</el-table-column>
				<el-table-column prop="duration_ms" label="耗时(ms)" width="110" />
				<el-table-column prop="error" label="错误" min-width="180" show-overflow-tooltip />
			</el-table>
		</el-card>
	</div>
</template>
<script setup lang="ts" name="opsJobs">
import { onMounted, ref } from 'vue';
import { useOpsApi } from '/@/api/ops';
import { formatDateTime } from '/@/utils/business';
import type { JobSnapshot } from '/@/utils/business';
const api = useOpsApi();
const loading = ref(false);
const jobs = ref<JobSnapshot[]>([]);
const load = async () => {
	loading.value = true;
	try { const res = await api.jobs(); jobs.value = res.data || []; }
	finally { loading.value = false; }
};
onMounted(load);
</script>
<style scoped lang="scss">.header { display: flex; align-items: center; justify-content: space-between; }</style>
