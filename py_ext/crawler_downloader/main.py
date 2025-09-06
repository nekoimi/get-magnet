from DrissionPage import Chromium, ChromiumOptions
from DrissionPage.common import Settings

Settings.set_language('zh_cn')  # 设置为中文时，填入'zh_cn'

if __name__ == '__main__':
    co = ChromiumOptions()
    co.set_proxy('http://127.0.0.1:12080')
    tab = Chromium(co).latest_tab
    tab.get(url='https://rucaptcha.com/42')

