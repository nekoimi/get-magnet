import request from '/@/utils/request';

/**
 * 磁力链接 API 接口集合
 * @method list 获取磁力链接列表
 * @method detail 获取磁力链接详情
 * @method create 创建磁力链接
 * @method update 更新磁力链接
 * @method delete 删除磁力链接
 * @method statusOptions 获取磁力链接状态选项
 */
export function useMagnetApi() {
	return {
		list: (data: object) => {
			return request({
				url: '/api/v1/magnets/list',
				method: 'post',
				data,
			});
		},
		detail: (params: object) => {
			return request({
				url: '/api/v1/magnets/detail',
				method: 'get',
				params,
			});
		},
		create: (data: object) => {
			return request({
				url: '/api/v1/magnets/create',
				method: 'post',
				data,
			});
		},
		update: (data: object) => {
			return request({
				url: '/api/v1/magnets/update',
				method: 'post',
				data,
			});
		},
		delete: (data: object) => {
			return request({
				url: '/api/v1/magnets/delete',
				method: 'post',
				data,
			});
		},
		statusOptions: () => {
			return request({
				url: '/api/v1/magnets/statusOptions',
				method: 'get',
			});
		},
		sourceOptions: () => {
			return request({
				url: '/api/v1/magnets/sourceOptions',
				method: 'get',
			});
		},
		markStatus: (data: object) => {
			return request({
				url: '/api/v1/magnets/markStatus',
				method: 'post',
				data,
			});
		},
		rebuildSTRM: (data: object) => {
			return request({
				url: '/api/v1/magnets/rebuildSTRM',
				method: 'post',
				data,
			});
		},
		rebuildSTRMBatch: (data: object) => {
			return request({
				url: '/api/v1/magnets/rebuildSTRMBatch',
				method: 'post',
				data,
			});
		},
	};
}
