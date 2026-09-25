<template>
	<div class="home-container layout-padding">
		<el-row :gutter="12" class="mb15">
			<el-col v-for="item in summaryCards" :key="item.label" :xs="24" :sm="12" :md="8" :lg="4">
				<el-card shadow="hover" class="summary-card">
					<div class="summary-card__label">{{ item.label }}</div>
					<div class="summary-card__value">{{ item.value }}</div>
				</el-card>
			</el-col>
		</el-row>

		<el-row :gutter="12">
			<el-col :xs="24" :lg="14">
				<el-card shadow="hover" class="layout-padding-auto">
					<template #header>
						<div class="card-header">
							<span>资源状态</span>
							<el-button text type="primary" :loading="state.loading" @click="getSummary">刷新</el-button>
						</div>
					</template>
					<el-table :data="state.summary.status_counts" v-loading="state.loading" height="320">
						<el-table-column prop="name" label="状态" min-width="120"></el-table-column>
						<el-table-column prop="count" label="数量" width="120"></el-table-column>
						<el-table-column label="占比" min-width="180">
							<template #default="scope">
								<el-progress :percentage="getPercent(scope.row.count)" :stroke-width="10"></el-progress>
							</template>
						</el-table-column>
					</el-table>
				</el-card>
			</el-col>
			<el-col :xs="24" :lg="10" class="home-side">
				<el-card shadow="hover" class="layout-padding-auto">
					<template #header>
						<div class="card-header">
							<span>系统概览</span>
						</div>
					</template>
					<el-descriptions :column="1" border>
						<el-descriptions-item label="资源总数">{{ state.summary.total }}</el-descriptions-item>
						<el-descriptions-item label="今日新增">{{ state.summary.today_created }}</el-descriptions-item>
						<el-descriptions-item label="待提交">{{ state.summary.pending_count }}</el-descriptions-item>
						<el-descriptions-item label="下载中">{{ state.summary.downloading }}</el-descriptions-item>
						<el-descriptions-item label="已完成">{{ state.summary.completed }}</el-descriptions-item>
						<el-descriptions-item label="失败">{{ state.summary.failed }}</el-descriptions-item>
						<el-descriptions-item label="生成时间">{{ state.summary.generated_at || '-' }}</el-descriptions-item>
					</el-descriptions>
				</el-card>
			</el-col>
		</el-row>
	</div>
</template>

<script setup lang="ts" name="home">
import { computed, onMounted, reactive } from 'vue';
import { ElMessage } from 'element-plus';
import { useDashboardApi } from '/@/api/dashboard';
import { getErrorMessage } from '/@/utils/business';

type StatusCount = {
	status: number;
	name: string;
	count: number;
};

type Summary = {
	total: number;
	today_created: number;
	status_counts: StatusCount[];
	pending_count: number;
	downloading: number;
	completed: number;
	failed: number;
	generated_at: string;
};

const emptySummary = (): Summary => ({
	total: 0,
	today_created: 0,
	status_counts: [],
	pending_count: 0,
	downloading: 0,
	completed: 0,
	failed: 0,
	generated_at: '',
});

const state = reactive({
	loading: false,
	summary: emptySummary(),
});

const summaryCards = computed(() => [
	{ label: '资源总数', value: state.summary.total },
	{ label: '今日新增', value: state.summary.today_created },
	{ label: '待提交', value: state.summary.pending_count },
	{ label: '下载中', value: state.summary.downloading },
	{ label: '已完成', value: state.summary.completed },
	{ label: '失败', value: state.summary.failed },
]);

const getPercent = (count: number) => {
	if (!state.summary.total) return 0;
	return Number(((count / state.summary.total) * 100).toFixed(2));
};

const getSummary = async () => {
	state.loading = true;
	try {
		const api = useDashboardApi();
		const res = await api.summary();
		state.summary = { ...emptySummary(), ...(res.data || {}) };
	} catch (error: unknown) {
		ElMessage.error(getErrorMessage(error, '获取首页统计失败'));
	} finally {
		state.loading = false;
	}
};

onMounted(() => {
	getSummary();
});
</script>

<style scoped lang="scss">
.home-container {
	.summary-card {
		min-height: 96px;
	}

	.summary-card__label {
		color: var(--el-text-color-secondary);
		font-size: 13px;
	}

	.summary-card__value {
		margin-top: 12px;
		font-size: 28px;
		font-weight: 600;
		color: var(--el-text-color-primary);
	}

	.card-header {
		display: flex;
		align-items: center;
		justify-content: space-between;
	}

	.home-side {
		@media (max-width: 1199px) {
			margin-top: 12px;
		}
	}
}
</style>
