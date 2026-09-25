import request from '/@/utils/request';

/**
 * 运维监控 API 接口集合
 * @method health 获取聚合健康检查
 * @method jobs 获取调度任务状态
 * @method logs 获取日志内容
 * @method version 获取版本信息
 */
export function useOpsApi() {
	return {
		health: () => {
			return request({
				url: '/api/v1/ops/health',
				method: 'get',
			});
		},
		jobs: () => {
			return request({
				url: '/api/v1/ops/jobs',
				method: 'get',
			});
		},
		logs: (params?: object) => {
			return request({
				url: '/api/v1/ops/logs',
				method: 'get',
				params,
			});
		},
		version: () => {
			return request({
				url: '/api/v1/ops/version',
				method: 'get',
			});
		},
	};
}
