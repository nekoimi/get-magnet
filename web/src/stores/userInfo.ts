import {defineStore} from 'pinia';
import Cookies from 'js-cookie';
import {Session} from '/@/utils/storage';
import {useUserApi} from "/@/api/user";

/**
 * 用户信息
 * @methods setUserInfos 设置用户信息
 */
export const useUserInfo = defineStore('userInfo', {
    state: (): UserInfosState => ({
        userInfos: {
            username: '',
            photo: '',
            time: 0,
            roles: [],
            authBtnList: [],
        },
    }),
    actions: {
        async setUserInfos() {
            // 存储用户信息到浏览器缓存
            if (Session.get('userInfo')) {
                this.userInfos = Session.get('userInfo');
            } else {
                this.userInfos = <UserInfos>await this.getApiUserInfo();
            }
        },
        // 模拟接口数据
        // https://gitee.com/lyt-top/vue-next-admin/issues/I5F1HP
        async getApiUserInfo() {
            return new Promise(async (resolve) => {
                let me = await useUserApi().getMe()
                Cookies.set('username', me.data.username);
                // 临时标记所有权限
                let defaultRoles: Array<string> = ['admin'];
                let defaultAuthBtnList: Array<string> = ['all'];
                // 用户信息模拟数据
                const userInfos = {
                    username: Cookies.get('username'),
                    photo: '',
                    time: new Date().getTime(),
                    roles: defaultRoles,
                    authBtnList: defaultAuthBtnList,
                };
                Session.set('userInfo', userInfos);
                resolve(userInfos);
            });
        },
    },
});
