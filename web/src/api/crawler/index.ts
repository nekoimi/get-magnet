import request from '/@/utils/request';

/**
 * 采集管理 API 接口集合
 * @method submitJavDB 提交 JavDB 详情页采集任务
 * @method submitJavDBPage 提交 JavDB 列表页采集任务
 * @method run 手动执行指定采集器
 * @method status 获取采集引擎状态
 * @method tasks 获取采集任务列表
 * @method providers 获取采集源列表
 * @method updateProvider 更新采集源配置
 */
export function useCrawlerApi() {
	return {
		submitJavDB: (data: object) => {
			return request({
				url: '/api/v1/crawler/submit/javdb',
				method: 'post',
				data,
			});
		},
		submitJavDBPage: (data: object) => {
			return request({
				url: '/api/v1/crawler/submit/javdbPage',
				method: 'post',
				data,
			});
		},
		run: (data: object) => {
			return request({
				url: '/api/v1/crawler/run',
				method: 'post',
				data,
			});
		},
		status: () => {
			return request({
				url: '/api/v1/crawler/status',
				method: 'get',
			});
		},
		tasks: (params?: object) => {
			return request({
				url: '/api/v1/crawler/tasks',
				method: 'get',
				params,
			});
		},
		providers: () => {
			return request({
				url: '/api/v1/crawler/providers',
				method: 'get',
			});
		},
		updateProvider: (data: object) => {
			return request({
				url: '/api/v1/crawler/providers/update',
				method: 'post',
				data,
			});
		},
	};
}
