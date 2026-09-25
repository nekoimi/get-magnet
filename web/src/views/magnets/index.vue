<template>
	<div class="magnets-container layout-padding">
		<el-card shadow="hover" class="layout-padding-auto">
			<div class="magnets-search mb15">
				<el-input
					size="default"
					placeholder="请输入标题或编号"
					v-model="state.tableData.param.keyword"
					style="max-width: 180px"
					clearable
					@clear="getTableData"
				>
				</el-input>
				<el-select
					v-model="state.tableData.param.status"
					placeholder="状态"
					size="default"
					class="ml10"
					style="max-width: 120px"
					clearable
					@change="getTableData"
				>
					<el-option label="全部" :value="undefined"></el-option>
					<el-option v-for="item in state.statusOptions" :key="item.value" :label="item.label" :value="item.value"></el-option>
				</el-select>
				<el-select
					v-model="state.tableData.param.origin"
					placeholder="来源"
					class="ml10"
					style="max-width: 130px"
					clearable
					@change="getTableData"
				>
					<el-option v-for="item in sourceOptions" :key="item.value" :label="item.label" :value="item.value" />
				</el-select>
				<el-select
					v-model="state.tableData.param.hasSTRM"
					placeholder="STRM"
					class="ml10"
					style="max-width: 130px"
					clearable
					@change="getTableData"
				>
					<el-option label="已生成 STRM" :value="true" />
					<el-option label="未生成 STRM" :value="false" />
				</el-select>
				<el-button size="default" type="primary" class="ml10" @click="getTableData">
					<el-icon>
						<ele-Search />
					</el-icon>
					查询
				</el-button>
				<el-button size="default" type="success" class="ml10" @click="onOpenAddMagnet('add')">
					<el-icon>
						<ele-FolderAdd />
					</el-icon>
					新增磁力链接
				</el-button>
			</div>
			<el-table :data="state.tableData.data" v-loading="state.tableData.loading" style="width: 100%" @selection-change="handleSelectionChange">
				<el-table-column type="selection" width="55"></el-table-column>
				<el-table-column type="index" label="序号" width="60" />
				<el-table-column prop="number" label="编号" width="120" show-overflow-tooltip></el-table-column>
				<el-table-column prop="title" label="标题" min-width="200" show-overflow-tooltip></el-table-column>
				<el-table-column prop="actress0" label="女优" width="120" show-overflow-tooltip></el-table-column>
				<el-table-column prop="origin" label="来源" width="100" show-overflow-tooltip></el-table-column>
				<el-table-column prop="optimal_link" label="优选链接" min-width="150" show-overflow-tooltip>
					<template #default="scope">
						<el-link v-if="scope.row.optimal_link" type="primary" @click="onCopyOptimalLink(scope.row.optimal_link)">
							{{ scope.row.optimal_link }}
						</el-link>
					</template>
				</el-table-column>
				<el-table-column prop="status" label="状态" width="100" show-overflow-tooltip>
					<template #default="scope">
						<el-tag :type="getStatusTagType(scope.row.status)">{{ getStatusLabel(scope.row.status) }}</el-tag>
					</template>
				</el-table-column>
				<el-table-column prop="followed_by" label="任务 ID" min-width="160" show-overflow-tooltip></el-table-column>
				<el-table-column prop="download_retry_count" label="重试" width="70" show-overflow-tooltip></el-table-column>
				<el-table-column label="最后提交" width="180"><template #default="{ row }">{{ formatDateTime(row.last_submit_at) }}</template></el-table-column>
				<el-table-column label="完成时间" width="180"><template #default="{ row }">{{ formatDateTime(row.download_completed_at) }}</template></el-table-column>
				<el-table-column prop="download_error" label="错误" min-width="160" show-overflow-tooltip></el-table-column>
				<el-table-column label="创建时间" width="180"><template #default="{ row }">{{ formatDateTime(row.created_at) }}</template></el-table-column>
				<el-table-column label="操作" width="340" fixed="right">
					<template #default="scope">
						<el-button size="small" text type="primary" @click="onOpenDetail(scope.row)">详情</el-button>
						<el-button
							v-if="scope.row.status === 0"
							size="small"
							text
							type="success"
							:loading="isActionLoading(scope.row.id)"
							@click="onSubmitDownload(scope.row)"
						>
							提交
						</el-button>
						<el-button
							v-if="scope.row.status === 4"
							size="small"
							text
							type="warning"
							:loading="isActionLoading(scope.row.id)"
							@click="onRetryDownload(scope.row)"
						>
							重试
						</el-button>
						<el-button v-if="scope.row.status === 3 && scope.row.play_file_path" size="small" text type="success" @click="onOpenPlay(scope.row)">
							播放
						</el-button>
						<el-button
							v-if="scope.row.status === 3 && scope.row.followed_by"
							size="small"
							text
							type="warning"
							:loading="isActionLoading(scope.row.id)"
							@click="onRebuildSTRM(scope.row)"
						>
							STRM
						</el-button>
						<el-button v-if="scope.row.download_error" size="small" text type="danger" @click="onShowError(scope.row)">错误</el-button>
						<el-button size="small" text type="primary" @click="onOpenEditMagnet('edit', scope.row)">编辑</el-button>
						<el-button size="small" text type="danger" @click="onRowDel(scope.row)">删除</el-button>
					</template>
				</el-table-column>
			</el-table>
			<div class="mt15" v-if="selectedIds.length > 0">
				<el-button size="default" type="primary" @click="onBatchCopyOptimalLinks">
					<el-icon>
						<ele-DocumentCopy />
					</el-icon>
					批量复制优选链接 ({{ selectedIds.length }})
				</el-button>
				<el-button size="default" type="danger" @click="onBatchDelete">
					<el-icon>
						<ele-Delete />
					</el-icon>
					批量删除 ({{ selectedIds.length }})
				</el-button>
			</div>
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
			>
			</el-pagination>
		</el-card>
		<el-drawer v-model="state.detail.visible" size="560px" title="资源详情">
			<div v-loading="state.detail.loading" class="magnet-detail" v-if="state.detail.data">
				<el-descriptions :column="1" border>
					<el-descriptions-item label="编号">{{ state.detail.data.magnet.number }}</el-descriptions-item>
					<el-descriptions-item label="标题">{{ state.detail.data.magnet.title }}</el-descriptions-item>
					<el-descriptions-item label="来源">{{ state.detail.data.magnet.origin }}</el-descriptions-item>
					<el-descriptions-item label="状态">
						<el-tag :type="getStatusTagType(state.detail.data.magnet.status)">{{ state.detail.data.status_label }}</el-tag>
					</el-descriptions-item>
					<el-descriptions-item label="链接数量">{{ state.detail.data.link_count }}</el-descriptions-item>
					<el-descriptions-item label="任务 ID">
						<el-link v-if="state.detail.data.download.task_id" type="primary" @click="copyText(state.detail.data.download.task_id)">
							{{ state.detail.data.download.task_id }}
						</el-link>
						<span v-else>-</span>
					</el-descriptions-item>
					<el-descriptions-item label="播放文件">{{ state.detail.data.post_process.play_file_path || '-' }}</el-descriptions-item>
					<el-descriptions-item label="STRM">
						<el-link v-if="state.detail.data.post_process.strm_path" type="primary" @click="copyText(state.detail.data.post_process.strm_path)">
							{{ state.detail.data.post_process.strm_path }}
						</el-link>
						<span v-else>-</span>
					</el-descriptions-item>
					<el-descriptions-item label="下载错误">{{ state.detail.data.download.error || '-' }}</el-descriptions-item>
				</el-descriptions>

				<div class="detail-actions mt15">
					<el-button v-if="state.detail.data.play_url" type="success" @click="onOpenPlayURL(state.detail.data.play_url)">打开播放地址</el-button>
					<el-button v-if="state.detail.data.play_url" @click="copyText(state.detail.data.play_url)">复制播放地址</el-button>
					<el-button v-if="state.detail.data.download.task_id" :loading="state.detail.cloudTaskLoading" @click="onLoadCloudTask">
						查询网盘任务
					</el-button>
				</div>

				<el-table v-if="state.detail.data.magnet.links?.length" :data="state.detail.data.magnet.links.map((link) => ({ link }))" class="mt15" max-height="180">
					<el-table-column prop="link" label="全部磁力链接" show-overflow-tooltip>
						<template #default="scope">
							<el-link type="primary" @click="copyText(scope.row.link)">{{ scope.row.link }}</el-link>
						</template>
					</el-table-column>
				</el-table>

				<el-card v-if="state.detail.cloudTask" shadow="never" class="mt15">
					<el-descriptions :column="1" border>
						<el-descriptions-item label="网盘状态">{{ state.detail.cloudTask.status }}</el-descriptions-item>
						<el-descriptions-item label="进度">{{ state.detail.cloudTask.progress || 0 }}%</el-descriptions-item>
						<el-descriptions-item label="保存路径">{{ state.detail.cloudTask.save_path || '-' }}</el-descriptions-item>
						<el-descriptions-item label="错误">{{ state.detail.cloudTask.error_message || '-' }}</el-descriptions-item>
						<el-descriptions-item label="Warnings">{{ (state.detail.cloudTask.warnings || []).join('；') || '-' }}</el-descriptions-item>
					</el-descriptions>
					<el-table :data="state.detail.cloudTask.files || []" class="mt15" max-height="180">
						<el-table-column prop="name" label="文件" show-overflow-tooltip></el-table-column>
						<el-table-column prop="path" label="路径" show-overflow-tooltip></el-table-column>
						<el-table-column prop="size" label="大小" width="120"></el-table-column>
					</el-table>
				</el-card>

				<el-timeline class="mt15">
					<el-timeline-item v-for="event in state.detail.data.events" :key="event.id" :timestamp="formatDateTime(event.created_at)">
						<div>{{ event.message || event.event_type }}</div>
						<div v-if="event.extra" class="event-extra">{{ event.extra }}</div>
					</el-timeline-item>
				</el-timeline>
			</div>
		</el-drawer>
		<MagnetDialog ref="magnetDialogRef" @refresh="getTableData()" />
	</div>
</template>

<script setup lang="ts" name="magnets">
import { defineAsyncComponent, reactive, onMounted, ref } from 'vue';
import { ElMessageBox, ElMessage } from 'element-plus';
import { useMagnetApi } from '/@/api/magnet';
import { useDownloadApi } from '/@/api/download';
import { useCloudDriverApi } from '/@/api/cloudDriver';
import commonFunction from '/@/utils/commonFunction';
import { formatDateTime, getErrorMessage } from '/@/utils/business';

// 引入组件
const MagnetDialog = defineAsyncComponent(() => import('/@/views/magnets/dialog.vue'));
const { copyText } = commonFunction();

type StatusOption = {
	label: string;
	value: number;
};

// 定义变量内容
const magnetDialogRef = ref();
const selectedIds = ref<number[]>([]);
const selectedRows = ref<MagnetType[]>([]);
const sourceOptions = ref<Array<{ label: string; value: string }>>([]);
const state = reactive<MagnetState>({
	statusOptions: [] as StatusOption[],
	actionLoadingIds: [] as number[],
	detail: {
		visible: false,
		loading: false,
		data: undefined,
		cloudTask: undefined,
		cloudTaskLoading: false,
	},
	tableData: {
		data: [],
		total: 0,
		loading: false,
		param: {
			page_num: 1,
			page_size: 10,
			keyword: '',
			status: undefined as number | undefined,
			origin: '',
			hasSTRM: undefined as boolean | undefined,
		},
	},
});

const statusTagTypes: Record<number, 'success' | 'warning' | 'info' | 'primary' | 'danger'> = {
	0: 'info',
	1: 'warning',
	2: 'primary',
	3: 'success',
	4: 'danger',
};

const getSourceOptions = async () => {
	const api = useMagnetApi();
	const res = await api.sourceOptions();
	sourceOptions.value = res.data || [];
};

const getStatusLabel = (status: number) => {
	return state.statusOptions.find((item) => item.value === status)?.label || '未知';
};

const getStatusTagType = (status: number) => {
	return statusTagTypes[status] || 'danger';
};

const isActionLoading = (id: number) => {
	return state.actionLoadingIds.includes(id);
};

const setActionLoading = (id: number, loading: boolean) => {
	if (loading) {
		if (!state.actionLoadingIds.includes(id)) state.actionLoadingIds.push(id);
		return;
	}
	state.actionLoadingIds = state.actionLoadingIds.filter((item) => item !== id);
};

const getStatusOptions = async () => {
	try {
		const api = useMagnetApi();
		const res = await api.statusOptions();
		state.statusOptions = res.data || [];
	} catch (error: unknown) {
		state.statusOptions = [
			{ label: '已采集', value: 0 },
			{ label: '提交中', value: 1 },
			{ label: '下载中', value: 2 },
			{ label: '已完成', value: 3 },
			{ label: '失败', value: 4 },
		];
	}
};

// 获取表格数据
const getTableData = async () => {
	state.tableData.loading = true;
	try {
		const api = useMagnetApi();
		const res = await api.list({
			page_num: state.tableData.param.page_num,
			page_size: state.tableData.param.page_size,
			keyword: state.tableData.param.keyword,
			status: state.tableData.param.status,
			origin: state.tableData.param.origin,
			has_strm: state.tableData.param.hasSTRM,
		});
		state.tableData.data = res.data.list || [];
		state.tableData.total = res.data.total || 0;
	} catch (error: unknown) {
		ElMessage.error(getErrorMessage(error, '获取数据失败'));
	} finally {
		state.tableData.loading = false;
	}
};

const handleSubmitResult = (data: any) => {
	const success = data?.success || 0;
	const failed = data?.failed || 0;
	if (failed > 0) {
		ElMessage.warning(`提交完成，成功 ${success} 条，失败 ${failed} 条`);
	} else {
		ElMessage.success(`提交成功 ${success} 条`);
	}
	getTableData();
};

const onSubmitDownload = async (row: MagnetType) => {
	setActionLoading(row.id, true);
	try {
		const api = useDownloadApi();
		const res = await api.submit({ ids: [row.id] });
		handleSubmitResult(res.data);
	} catch (error: unknown) {
		ElMessage.error(getErrorMessage(error, '提交下载失败'));
	} finally {
		setActionLoading(row.id, false);
	}
};

const onRetryDownload = async (row: MagnetType) => {
	setActionLoading(row.id, true);
	try {
		const api = useDownloadApi();
		const res = await api.retry({ ids: [row.id] });
		handleSubmitResult(res.data);
	} catch (error: unknown) {
		ElMessage.error(getErrorMessage(error, '重试下载失败'));
	} finally {
		setActionLoading(row.id, false);
	}
};

const onCopyOptimalLink = (link: string) => {
	copyText(link);
};

const onBatchCopyOptimalLinks = () => {
	const links = selectedRows.value.map((item) => item.optimal_link?.trim()).filter((link): link is string => Boolean(link));
	if (links.length === 0) {
		ElMessage.warning('选中的数据没有优选链接');
		return;
	}
	copyText(links.join('\n'));
};

const onOpenDetail = async (row: MagnetType) => {
	state.detail.visible = true;
	state.detail.loading = true;
	state.detail.cloudTask = undefined;
	try {
		const api = useMagnetApi();
		const res = await api.detail({ id: row.id });
		state.detail.data = res.data;
	} catch (error: unknown) {
		ElMessage.error(getErrorMessage(error, '获取详情失败'));
	} finally {
		state.detail.loading = false;
	}
};

const onShowError = (row: MagnetType) => {
	ElMessageBox.alert(row.download_error || '-', `下载错误：${row.number}`, {
		confirmButtonText: '关闭',
	});
};

const buildPlayURL = (row: MagnetType) => {
	const params = new URLSearchParams();
	if (row.play_file_id) params.set('file_id', row.play_file_id);
	if (row.play_file_path) params.set('path', row.play_file_path);
	const query = params.toString();
	return `/api/play/${encodeURIComponent(row.number)}${query ? `?${query}` : ''}`;
};

const onOpenPlay = (row: MagnetType) => {
	onOpenPlayURL(buildPlayURL(row));
};

const onOpenPlayURL = (url: string) => {
	window.open(url, '_blank');
};

const onRebuildSTRM = async (row: MagnetType) => {
	setActionLoading(row.id, true);
	try {
		const api = useMagnetApi();
		await api.rebuildSTRM({ id: row.id });
		ElMessage.success('STRM 已重新生成');
		getTableData();
		if (state.detail.visible && state.detail.data?.magnet.id === row.id) {
			onOpenDetail(row);
		}
	} catch (error: unknown) {
		ElMessage.error(getErrorMessage(error, '重建 STRM 失败'));
	} finally {
		setActionLoading(row.id, false);
	}
};

const onLoadCloudTask = async () => {
	const taskID = state.detail.data?.download.task_id;
	if (!taskID) return;
	state.detail.cloudTaskLoading = true;
	try {
		const api = useCloudDriverApi();
		const res = await api.task(taskID);
		state.detail.cloudTask = res.data;
	} catch (error: unknown) {
		ElMessage.error(getErrorMessage(error, '查询网盘任务失败'));
	} finally {
		state.detail.cloudTaskLoading = false;
	}
};

// 打开新增磁力链接弹窗
const onOpenAddMagnet = (type: string) => {
	magnetDialogRef.value.openDialog(type);
};

// 打开修改磁力链接弹窗
const onOpenEditMagnet = (type: string, row: MagnetType) => {
	magnetDialogRef.value.openDialog(type, row);
};

// 删除磁力链接
const onRowDel = (row: MagnetType) => {
	ElMessageBox.confirm(`此操作将永久删除编号为"${row.number}"的磁力链接，是否继续?`, '提示', {
		confirmButtonText: '确认',
		cancelButtonText: '取消',
		type: 'warning',
	})
		.then(async () => {
			try {
				const api = useMagnetApi();
				await api.delete({ ids: [row.id] });
				ElMessage.success('删除成功');
				getTableData();
			} catch (error: unknown) {
				ElMessage.error(getErrorMessage(error, '删除失败'));
			}
		})
		.catch(() => {});
};

// 批量删除
const onBatchDelete = () => {
	if (selectedIds.value.length === 0) {
		ElMessage.warning('请选择要删除的数据');
		return;
	}
	ElMessageBox.confirm(`此操作将永久删除选中的 ${selectedIds.value.length} 条数据，是否继续?`, '提示', {
		confirmButtonText: '确认',
		cancelButtonText: '取消',
		type: 'warning',
	})
		.then(async () => {
			try {
				const api = useMagnetApi();
				await api.delete({ ids: selectedIds.value });
				ElMessage.success('批量删除成功');
				selectedIds.value = [];
				selectedRows.value = [];
				getTableData();
			} catch (error: unknown) {
				ElMessage.error(getErrorMessage(error, '批量删除失败'));
			}
		})
		.catch(() => {});
};

// 表格选择变化
const handleSelectionChange = (selection: MagnetType[]) => {
	selectedRows.value = selection;
	selectedIds.value = selection.map((item) => item.id);
};

// 分页改变
const onHandleSizeChange = (val: number) => {
	state.tableData.param.page_size = val;
	getTableData();
};

// 分页改变
const onHandleCurrentChange = (val: number) => {
	state.tableData.param.page_num = val;
	getTableData();
};

// 页面加载时
onMounted(() => {
	getStatusOptions();
	getSourceOptions();
	getTableData();
});
</script>

<style scoped lang="scss">
.magnets-container {
	:deep(.el-card__body) {
		display: flex;
		flex-direction: column;
		flex: 1;
		overflow: auto;
		.el-table {
			flex: 1;
		}
	}
	.magnet-detail {
		.detail-actions {
			display: flex;
			flex-wrap: wrap;
			gap: 8px;
		}
		.event-extra {
			margin-top: 4px;
			color: var(--el-text-color-secondary);
			word-break: break-all;
		}
	}
}
</style>
