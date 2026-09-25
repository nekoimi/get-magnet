import request from '/@/utils/request';

export function useUserApi() {
    return {
        getMe: () => {
            return request({
                url: '/api/v1/me',
                method: 'post'
            })
        }
    }
}
