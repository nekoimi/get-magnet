<template>
	<div class="download-queue-container layout-padding">
		<el-card shadow="hover" class="layout-padding-auto">
			<div class="queue-toolbar mb15">
				<el-button type="primary" :loading="state.schedulerLoading" @click="onRunSchedulerOnce">
					<el-icon><ele-Refresh /></el-icon>
					运行调度
				</el-button>
				<el-button :loading="state.schedulerLoading" @click="getScheduler">刷新状态</el-button>
				<el-tag v-if="state.scheduler">调度：{{ state.scheduler.enabled ? '启用' : '禁用' }}</el-tag>
				<el-tag v-if="state.scheduler" type="info">Cron：{{ state.scheduler.submit_cron }}</el-tag>
				<el-tag v-if="state.scheduler" type="info">批量：{{ state.scheduler.batch_size }}</el-tag>
			</div>

			<el-tabs v-model="state.activeStatus" @tab-change="onTabChange">
				<el-tab-pane label="待提交" name="0"></el-tab-pane>
				<el-tab-pane label="提交中" name="1"></el-tab-pane>
				<el-tab-pane label="下载中" name="2"></el-tab-pane>
				<el-tab-pane label="已完成" name="3"></el-tab-pane>
				<el-tab-pane label="失败" name="4"></el-tab-pane>
			</el-tabs>

			<div class="mb15" v-if="selectedIds.length > 0">
				<el-button v-if="state.activeStatus === '0'" type="success" :loading="state.batchLoading" @click="onBatchSubmit">
					批量提交 ({{ selectedIds.length }})
				</el-button>
				<el-button v-if="state.activeStatus === '4'" type="warning" :loading="state.batchLoading" @click="onBatchRetry">
					批量重试 ({{ selectedIds.length }})
				</el-button>
			</div>

			<el-table :data="state.tableData.data" v-loading="state.tableData.loading" style="width: 100%" @selection-change="handleSelectionChange">
				<el-table-column type="selection" width="55"></el-table-column>
				<el-table-column prop="number" label="编号" width="120" show-overflow-tooltip></el-table-column>
				<el-table-column prop="title" label="标题" min-width="220" show-overflow-tooltip></el-table-column>
				<el-table-column prop="origin" label="来源" width="100"></el-table-column>
				<el-table-column prop="followed_by" label="任务 ID" min-width="180" show-overflow-tooltip>
					<template #default="scope">
						<el-link v-if="scope.row.followed_by" type="primary" @click="copyText(scope.row.followed_by)">
							{{ scope.row.followed_by }}
						</el-link>
						<span v-else>-</span>
					</template>
				</el-table-column>
				<el-table-column prop="download_retry_count" label="重试" width="70"></el-table-column>
				<el-table-column label="最后提交" width="180"><template #default="{ row }">{{ formatDateTime(row.last_submit_at) }}</template></el-table-column>
				<el-table-column label="完成时间" width="180"><template #default="{ row }">{{ formatDateTime(row.download_completed_at) }}</template></el-table-column>
				<el-table-column prop="download_error" label="错误" min-width="180" show-overflow-tooltip></el-table-column>
				<el-table-column label="操作" width="220" fixed="right">
					<template #default="scope">
						<el-button v-if="scope.row.status === 0" size="small" text type="success" @click="onSubmit(scope.row)">提交</el-button>
						<el-button v-if="scope.row.status === 4" size="small" text type="warning" @click="onRetry(scope.row)">重试</el-button>
						<el-button v-if="scope.row.followed_by" size="small" text type="primary" @click="onOpenCloudTask(scope.row.followed_by)">网盘任务</el-button>
						<el-button v-if="scope.row.download_error" size="small" text type="danger" @click="onShowError(scope.row)">错误</el-button>
					</template>
				</el-table-column>
			</el-table>

			<el-pagination
				@size-change="onHandleSizeChange"
				@current-change="onHandleCurrentChange"
				class="mt15"
				:pager-count="5"
				:page-sizes="[10, 20, 30, 50, 100]"
				v-model:current-page="state.tableData.param.page_num"
				background
				v-model:page-size="state.tableData.param.page_size"
				layout="total, sizes, prev, pager, next, jumper"
				:total="state.tableData.total"
			></el-pagination>
		</el-card>

		<el-dialog v-model="state.cloudTask.visible" title="网盘任务详情" width="720px">
			<div v-loading="state.cloudTask.loading">
				<el-descriptions v-if="state.cloudTask.data" :column="1" border>
					<el-descriptions-item label="任务 ID">{{ state.cloudTask.data.task_id }}</el-descriptions-item>
					<el-descriptions-item label="状态">{{ state.cloudTask.data.status }}</el-descriptions-item>
					<el-descriptions-item label="进度">{{ state.cloudTask.data.progress || 0 }}%</el-descriptions-item>
					<el-descriptions-item label="保存路径">{{ state.cloudTask.data.save_path || '-' }}</el-descriptions-item>
					<el-descriptions-item label="错误">{{ state.cloudTask.data.error_message || '-' }}</el-descriptions-item>
					<el-descriptions-item label="Warnings">{{ (state.cloudTask.data.warnings || []).join('；') || '-' }}</el-descriptions-item>
				</el-descriptions>
				<el-table v-if="state.cloudTask.data" :data="state.cloudTask.data.files || []" class="mt15" max-height="260">
					<el-table-column prop="name" label="文件" show-overflow-tooltip></el-table-column>
					<el-table-column prop="path" label="路径" show-overflow-tooltip></el-table-column>
					<el-table-column prop="size" label="大小" width="120"></el-table-column>
				</el-table>
			</div>
		</el-dialog>
	</div>
</template>

<script setup lang="ts" name="downloadQueue">
import { onMounted, reactive, ref } from 'vue';
import { ElMessage, ElMessageBox } from 'element-plus';
import { useDownloadApi } from '/@/api/download';
import { useCloudDriverApi } from '/@/api/cloudDriver';
import commonFunction from '/@/utils/commonFunction';
import { formatDateTime, getErrorMessage } from '/@/utils/business';

const { copyText } = commonFunction();
const selectedIds = ref<number[]>([]);

const state = reactive({
	activeStatus: '0',
	batchLoading: false,
	schedulerLoading: false,
	scheduler: undefined as any,
	cloudTask: {
		visible: false,
		loading: false,
		data: undefined as any,
	},
	tableData: {
		data: [] as MagnetType[],
		total: 0,
		loading: false,
		param: {
			page_num: 1,
			page_size: 20,
		},
	},
});

const getTableData = async () => {
	state.tableData.loading = true;
	try {
		const api = useDownloadApi();
		const res = await api.queue({
			status: Number(state.activeStatus),
			page_num: state.tableData.param.page_num,
			page_size: state.tableData.param.page_size,
		});
		state.tableData.data = res.data.list || [];
		state.tableData.total = res.data.total || 0;
	} catch (error: unknown) {
		ElMessage.error(getErrorMessage(error, '获取下载队列失败'));
	} finally {
		state.tableData.loading = false;
	}
};

const getScheduler = async () => {
	state.schedulerLoading = true;
	try {
		const api = useDownloadApi();
		const res = await api.scheduler();
		state.scheduler = res.data;
	} catch (error: unknown) {
		ElMessage.error(getErrorMessage(error, '获取调度状态失败'));
	} finally {
		state.schedulerLoading = false;
	}
};

const onRunSchedulerOnce = async () => {
	state.schedulerLoading = true;
	try {
		const api = useDownloadApi();
		const res = await api.runSchedulerOnce();
		state.scheduler = res.data;
		ElMessage.success('调度已执行');
		getTableData();
	} catch (error: unknown) {
		ElMessage.error(getErrorMessage(error, '运行调度失败'));
	} finally {
		state.schedulerLoading = false;
	}
};

const handleSubmitResult = (data: any) => {
	const success = data?.success || 0;
	const failed = data?.failed || 0;
	ElMessage[failed > 0 ? 'warning' : 'success'](`处理完成，成功 ${success} 条，失败 ${failed} 条`);
	selectedIds.value = [];
	getTableData();
};

const onSubmit = async (row: MagnetType) => {
	const api = useDownloadApi();
	const res = await api.submit({ id: row.id });
	handleSubmitResult(res.data);
};

const onRetry = async (row: MagnetType) => {
	const api = useDownloadApi();
	const res = await api.retry({ id: row.id });
	handleSubmitResult(res.data);
};

const onBatchSubmit = async () => {
	state.batchLoading = true;
	try {
		const api = useDownloadApi();
		const res = await api.submit({ ids: selectedIds.value });
		handleSubmitResult(res.data);
	} catch (error: unknown) {
		ElMessage.error(getErrorMessage(error, '批量提交失败'));
	} finally {
		state.batchLoading = false;
	}
};

const onBatchRetry = async () => {
	state.batchLoading = true;
	try {
		const api = useDownloadApi();
		const res = await api.retry({ ids: selectedIds.value });
		handleSubmitResult(res.data);
	} catch (error: unknown) {
		ElMessage.error(getErrorMessage(error, '批量重试失败'));
	} finally {
		state.batchLoading = false;
	}
};

const onOpenCloudTask = async (taskID: string) => {
	state.cloudTask.visible = true;
	state.cloudTask.loading = true;
	state.cloudTask.data = undefined;
	try {
		const api = useCloudDriverApi();
		const res = await api.task(taskID);
		state.cloudTask.data = res.data;
	} catch (error: unknown) {
		ElMessage.error(getErrorMessage(error, '查询网盘任务失败'));
	} finally {
		state.cloudTask.loading = false;
	}
};

const onShowError = (row: MagnetType) => {
	ElMessageBox.alert(row.download_error || '-', `下载错误：${row.number}`, {
		confirmButtonText: '关闭',
	});
};

const handleSelectionChange = (selection: MagnetType[]) => {
	selectedIds.value = selection.map((item) => item.id);
};

const onTabChange = () => {
	selectedIds.value = [];
	state.tableData.param.page_num = 1;
	getTableData();
};

const onHandleSizeChange = (val: number) => {
	state.tableData.param.page_size = val;
	getTableData();
};

const onHandleCurrentChange = (val: number) => {
	state.tableData.param.page_num = val;
	getTableData();
};

onMounted(() => {
	getScheduler();
	getTableData();
});
</script>

<style scoped lang="scss">
.download-queue-container {
	.queue-toolbar {
		display: flex;
		flex-wrap: wrap;
		align-items: center;
		gap: 8px;
	}
}
</style>
