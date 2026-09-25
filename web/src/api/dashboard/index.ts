import request from '/@/utils/request';

/**
 * 首页统计 API 接口集合
 * @method summary 获取管理首页统计信息
 */
export function useDashboardApi() {
	return {
		summary: () => {
			return request({
				url: '/api/v1/dashboard/summary',
				method: 'get',
			});
		},
	};
}
