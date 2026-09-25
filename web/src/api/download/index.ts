import request from '/@/utils/request';

/**
 * 下载管理 API 接口集合
 * @method queue 获取下载队列
 * @method submit 手动提交下载
 * @method retry 重试失败下载
 * @method runSchedulerOnce 手动运行一轮下载调度
 * @method scheduler 获取下载调度器状态
 */
export function useDownloadApi() {
	return {
		queue: (params?: object) => {
			return request({
				url: '/api/v1/download/queue',
				method: 'get',
				params,
			});
		},
		submit: (data: object) => {
			return request({
				url: '/api/v1/download/submit',
				method: 'post',
				data,
			});
		},
		retry: (data: object) => {
			return request({
				url: '/api/v1/download/retry',
				method: 'post',
				data,
			});
		},
		runSchedulerOnce: () => {
			return request({
				url: '/api/v1/download/runSchedulerOnce',
				method: 'post',
			});
		},
		scheduler: () => {
			return request({
				url: '/api/v1/download/scheduler',
				method: 'get',
			});
		},
	};
}
