import request from '/@/utils/request';

/**
 * 系统设置 API 接口集合
 * @method list 获取只读运行配置
 * @method testCloudDriver 测试网盘中间服务连接
 * @method testAria2 测试 aria2 连接
 * @method testDrissionRod 测试 DrissionRod 连接
 */
export function useSettingsApi() {
	return {
		list: () => {
			return request({
				url: '/api/v1/settings',
				method: 'get',
			});
		},
		testCloudDriver: (data?: object) => {
			return request({
				url: '/api/v1/settings/testCloudDriver',
				method: 'post',
				data,
			});
		},
		testAria2: (data?: object) => {
			return request({
				url: '/api/v1/settings/testAria2',
				method: 'post',
				data,
			});
		},
		testDrissionRod: (data?: object) => {
			return request({
				url: '/api/v1/settings/testDrissionRod',
				method: 'post',
				data,
			});
		},
	};
}
