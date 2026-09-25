<template>
	<div class="crawler-quick-submit-container layout-padding">
		<el-card shadow="hover" class="layout-padding-auto">
			<template #header>
				<div class="card-header">
					<span>快速提交</span>
				</div>
			</template>
			<el-form ref="formRef" :model="state.form" :rules="rules" label-width="96px" class="quick-submit-form">
				<el-form-item label="采集类型" prop="type">
					<el-radio-group v-model="state.form.type">
						<el-radio-button label="detail">JavDB 详情页</el-radio-button>
						<el-radio-button label="page">JavDB 列表页</el-radio-button>
					</el-radio-group>
				</el-form-item>
				<el-form-item label="页面 URL" prop="url">
					<el-input v-model="state.form.url" type="textarea" :rows="4" placeholder="请输入 JavDB 页面地址" clearable></el-input>
				</el-form-item>
				<el-form-item>
					<el-button type="primary" :loading="state.loading" @click="onSubmit">提交任务</el-button>
					<el-button @click="onReset">重置</el-button>
				</el-form-item>
			</el-form>
		</el-card>
	</div>
</template>

<script setup lang="ts" name="crawlerQuickSubmit">
import { reactive, ref } from 'vue';
import { ElMessage, FormInstance, FormRules } from 'element-plus';
import { useCrawlerApi } from '/@/api/crawler';
import { getErrorMessage } from '/@/utils/business';

const formRef = ref<FormInstance>();
const state = reactive({
	loading: false,
	form: {
		type: 'detail',
		url: '',
	},
});

const rules = reactive<FormRules>({
	type: [{ required: true, message: '请选择采集类型', trigger: 'change' }],
	url: [{ required: true, message: '请输入页面 URL', trigger: 'blur' }],
});

const onSubmit = async () => {
	if (!formRef.value) return;
	const valid = await formRef.value.validate().catch(() => false);
	if (!valid) return;

	state.loading = true;
	try {
		const api = useCrawlerApi();
		if (state.form.type === 'page') {
			await api.submitJavDBPage({ url: state.form.url });
		} else {
			await api.submitJavDB({ url: state.form.url });
		}
		ElMessage.success('采集任务已提交');
	} catch (error: unknown) {
		ElMessage.error(getErrorMessage(error, '提交采集任务失败'));
	} finally {
		state.loading = false;
	}
};

const onReset = () => {
	state.form.url = '';
	state.form.type = 'detail';
	formRef.value?.clearValidate();
};
</script>

<style scoped lang="scss">
.crawler-quick-submit-container {
	.quick-submit-form {
		max-width: 760px;
	}

	.card-header {
		display: flex;
		align-items: center;
		justify-content: space-between;
	}
}
</style>
