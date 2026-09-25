<template>
	<div class="layout-padding">
		<el-card shadow="hover">
			<template #header><div class="header"><span>播放与 STRM</span><el-button :loading="loading" @click="load">刷新</el-button></div></template>
			<el-table :data="rows" v-loading="loading">
				<el-table-column prop="number" label="编号" width="130" />
				<el-table-column prop="title" label="标题" min-width="220" show-overflow-tooltip />
				<el-table-column prop="play_file_path" label="播放文件" min-width="220" show-overflow-tooltip />
				<el-table-column prop="play_file_size" label="大小" width="120" />
				<el-table-column prop="strm_path" label="STRM" min-width="220" show-overflow-tooltip />
				<el-table-column label="操作" width="190">
					<template #default="{ row }">
						<el-button text type="success" :disabled="!row.play_file_path" @click="play(row)">播放</el-button>
						<el-button text type="primary" :disabled="!row.strm_path" @click="copyText(row.strm_path)">复制</el-button>
						<el-button text type="warning" :loading="rebuilding === row.id" @click="rebuild(row)">重建</el-button>
					</template>
				</el-table-column>
			</el-table>
			<el-pagination
				class="mt15" background layout="total, sizes, prev, pager, next"
				v-model:current-page="page_num" v-model:page-size="page_size" :total="total"
				:page-sizes="[10, 20, 50]" @current-change="load" @size-change="load"
			/>
		</el-card>
	</div>
</template>

<script setup lang="ts" name="mediaList">
import { onMounted, ref } from 'vue';
import { ElMessage } from 'element-plus';
import { useMagnetApi } from '/@/api/magnet';
import commonFunction from '/@/utils/commonFunction';
const api = useMagnetApi();
const { copyText } = commonFunction();
const loading = ref(false);
const rebuilding = ref(0);
const rows = ref<MagnetType[]>([]);
const page_num = ref(1);
const page_size = ref(20);
const total = ref(0);
const load = async () => {
	loading.value = true;
	try {
		const res = await api.list({ page_num: page_num.value, page_size: page_size.value, status: 3 });
		rows.value = res.data?.list || [];
		total.value = res.data?.total || 0;
	} finally { loading.value = false; }
};
const play = (row: MagnetType) => window.open(`/api/play/${encodeURIComponent(row.number)}`, '_blank');
const rebuild = async (row: MagnetType) => {
	rebuilding.value = row.id;
	try {
		await api.rebuildSTRM({ id: row.id });
		ElMessage.success('STRM 已重建');
		await load();
	} finally { rebuilding.value = 0; }
};
onMounted(load);
</script>
<style scoped lang="scss">.header { display: flex; align-items: center; justify-content: space-between; }</style>
