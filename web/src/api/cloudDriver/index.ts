import request from '/@/utils/request';

export function useCloudDriverApi() {
	return {
		health: () => {
			return request({
				url: '/api/v1/cloud-driver/health',
				method: 'get',
			});
		},
		task: (taskID: string) => {
			return request({
				url: `/api/v1/cloud-driver/tasks/${encodeURIComponent(taskID)}`,
				method: 'get',
			});
		},
	};
}
