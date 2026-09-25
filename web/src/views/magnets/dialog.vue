<template>
	<div class="magnet-dialog-container">
		<el-dialog :title="state.dialog.title" v-model="state.dialog.isShowDialog" width="769px">
			<el-form ref="magnetDialogFormRef" :model="state.ruleForm" size="default" label-width="100px">
				<el-row :gutter="35">
					<el-col :xs="24" :sm="12" :md="12" :lg="12" :xl="12" class="mb20">
						<el-form-item label="编号">
							<el-input v-model="state.ruleForm.number" placeholder="请输入编号" clearable></el-input>
						</el-form-item>
					</el-col>
					<el-col :xs="24" :sm="12" :md="12" :lg="12" :xl="12" class="mb20">
						<el-form-item label="来源">
							<el-input v-model="state.ruleForm.origin" placeholder="请输入来源" clearable></el-input>
						</el-form-item>
					</el-col>
					<el-col :xs="24" :sm="24" :md="24" :lg="24" :xl="24" class="mb20">
						<el-form-item label="标题">
							<el-input v-model="state.ruleForm.title" placeholder="请输入标题" clearable></el-input>
						</el-form-item>
					</el-col>
					<el-col :xs="24" :sm="12" :md="12" :lg="12" :xl="12" class="mb20">
						<el-form-item label="女优">
							<el-input v-model="state.ruleForm.actress0" placeholder="请输入女优名称" clearable></el-input>
						</el-form-item>
					</el-col>
					<el-col :xs="24" :sm="12" :md="12" :lg="12" :xl="12" class="mb20">
						<el-form-item label="状态">
							<el-select v-model="state.ruleForm.status" placeholder="请选择状态" clearable class="w100">
								<el-option label="待处理" :value="0"></el-option>
								<el-option label="处理中" :value="1"></el-option>
								<el-option label="已完成" :value="2"></el-option>
							</el-select>
						</el-form-item>
					</el-col>
					<el-col :xs="24" :sm="24" :md="24" :lg="24" :xl="24" class="mb20">
						<el-form-item label="优选链接">
							<el-input v-model="state.ruleForm.optimal_link" placeholder="请输入优选链接" clearable></el-input>
						</el-form-item>
					</el-col>
					<el-col :xs="24" :sm="24" :md="24" :lg="24" :xl="24" class="mb20">
						<el-form-item label="所有链接">
							<el-input
								v-model="state.linksText"
								type="textarea"
								placeholder="请输入所有链接，多个链接用换行分隔"
								:rows="3"
								@input="handleLinksChange"
							></el-input>
						</el-form-item>
					</el-col>
					<el-col :xs="24" :sm="12" :md="12" :lg="12" :xl="12" class="mb20">
						<el-form-item label="原始URL主机">
							<el-input v-model="state.ruleForm.raw_url_host" placeholder="请输入原始URL主机" clearable></el-input>
						</el-form-item>
					</el-col>
					<el-col :xs="24" :sm="12" :md="12" :lg="12" :xl="12" class="mb20">
						<el-form-item label="原始URL路径">
							<el-input v-model="state.ruleForm.raw_url_path" placeholder="请输入原始URL路径" clearable></el-input>
						</el-form-item>
					</el-col>
					<el-col :xs="24" :sm="24" :md="24" :lg="24" :xl="24" class="mb20">
						<el-form-item label="关注来源">
							<el-input v-model="state.ruleForm.followed_by" placeholder="请输入关注来源" clearable></el-input>
						</el-form-item>
					</el-col>
				</el-row>
			</el-form>
			<template #footer>
				<span class="dialog-footer">
					<el-button @click="onCancel" size="default">取 消</el-button>
					<el-button type="primary" @click="onSubmit" size="default" :loading="state.dialog.loading">{{ state.dialog.submitTxt }}</el-button>
				</span>
			</template>
		</el-dialog>
	</div>
</template>

<script setup lang="ts" name="magnetDialog">
import { reactive, ref } from 'vue';
import { ElMessage } from 'element-plus';
import { useMagnetApi } from '/@/api/magnet';
import { getErrorMessage } from '/@/utils/business';

// 定义子组件向父组件传值/事件
const emit = defineEmits(['refresh']);

// 定义变量内容
const magnetDialogFormRef = ref();
const linksText = ref('');
const state = reactive({
	ruleForm: {
		id: 0,
		origin: '',
		title: '',
		number: '',
		optimal_link: '',
		links: [] as string[],
		raw_url_host: '',
		raw_url_path: '',
		status: 0,
		actress0: '',
		followed_by: '',
	},
	dialog: {
		isShowDialog: false,
		type: '',
		title: '',
		submitTxt: '',
		loading: false,
	},
});

// 处理链接文本变化
const handleLinksChange = () => {
	if (linksText.value) {
		state.ruleForm.links = linksText.value.split('\n').filter((link) => link.trim() !== '');
	} else {
		state.ruleForm.links = [];
	}
};

// 打开弹窗
const openDialog = (type: string, row?: MagnetType) => {
	if (type === 'edit' && row) {
		state.ruleForm = { ...row };
		state.dialog.title = '编辑磁力链接';
		state.dialog.submitTxt = '修 改';
		// 将链接数组转换为文本
		linksText.value = state.ruleForm.links ? state.ruleForm.links.join('\n') : '';
	} else {
		state.dialog.title = '新增磁力链接';
		state.dialog.submitTxt = '新 增';
		// 重置表单
		state.ruleForm = {
			id: 0,
			origin: '',
			title: '',
			number: '',
			optimal_link: '',
			links: [],
			raw_url_host: '',
			raw_url_path: '',
			status: 0,
			actress0: '',
			followed_by: '',
		};
		linksText.value = '';
	}
	state.dialog.isShowDialog = true;
};

// 关闭弹窗
const closeDialog = () => {
	state.dialog.isShowDialog = false;
};

// 取消
const onCancel = () => {
	closeDialog();
};

// 提交
const onSubmit = async () => {
	// 简单验证
	if (!state.ruleForm.title) {
		ElMessage.warning('请输入标题');
		return;
	}

	state.dialog.loading = true;
	try {
		const api = useMagnetApi();
		if (state.ruleForm.id > 0) {
			// 更新
			await api.update(state.ruleForm);
			ElMessage.success('更新成功');
		} else {
			// 新增
			await api.create(state.ruleForm);
			ElMessage.success('新增成功');
		}
		closeDialog();
		emit('refresh');
	} catch (error: unknown) {
		ElMessage.error(getErrorMessage(error, '操作失败'));
	} finally {
		state.dialog.loading = false;
	}
};

// 暴露变量
defineExpose({
	openDialog,
});
</script>

<style scoped lang="scss">
.magnet-dialog-container {
	.w100 {
		width: 100%;
	}
}
</style>
