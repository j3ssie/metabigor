import os
import time
import shutil
import requests
import urllib3
urllib3.disable_warnings(urllib3.exceptions.InsecureRequestWarning)

# just in case we need a real browser
try:
    from selenium import webdriver
    from selenium.webdriver.chrome.options import Options
except:
    pass

from . import config
from . import utils


'''
 Sending stuff
'''

normal_headers = {"User-Agent": "Mozilla/5.0 (X11; FreeBSD amd64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/40.0.2214.115 Safari/537.36", "Accept": "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8", "Accept-Language": "en-US,en;q=0.5", "Accept-Encoding": "gzip, deflate", "Connection": "close"}

post_headers = {"User-Agent": "Mozilla/5.0 (X11; FreeBSD amd64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/40.0.2214.115 Safari/537.36", "Accept": "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8", "Accept-Language": "en-US,en;q=0.5",
                "Accept-Encoding": "gzip, deflate", "Content-Type": "application/x-www-form-urlencoded", "Connection": "close", "Upgrade-Insecure-Requests": "1"}

json_headers = {"User-Agent": "Mozilla/5.0 (X11; FreeBSD amd64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/40.0.2214.115 Safari/537.36",
                "Accept": "application/json", "Accept-Language": "en-US,en;q=0.5", "Accept-Encoding": "gzip, deflate", "Content-Type": "application/json"}


def send_get(options, url, cookies=None, headers=normal_headers):
    utils.print_debug(options, url)
    if options.get('proxy'):
        r = requests.get(url, verify=False, headers=headers,
                         cookies=cookies, proxies=options.get('proxy'))
    else:
        r = requests.get(url, verify=False, headers=headers, cookies=cookies)

    return r


def send_post(options, url, data, cookies=None, is_json=False, headers=normal_headers, retry=3):
    r = just_post(options, url, data, cookies, is_json, headers=headers)
    utils.print_debug(options, r.status_code)
    if r.status_code >= 500:
        if retry:
            count = 0
            while count < retry:
                utils.random_sleep(3, 6)
                utils.print_debug(options, "Retry the request")
                r = just_post(options, url, data, cookies,
                              is_json, headers=headers)
                if r.status_code == 200:
                    return r
                count += 1
    return r


def just_post(options, url, data, cookies=None, is_json=False, headers=normal_headers):
    utils.print_debug(options, url)
    utils.print_debug(options, data)
    if options.get('proxy'):
        if is_json:
            r = requests.post(url, verify=False,
                              headers=json_headers, cookies=cookies, json=data, proxies=options.get('proxy'))
        else:
            r = requests.post(url, verify=False, headers=headers,
                              cookies=cookies, proxies=options.get('proxy'))

    else:
        if is_json:
            r = requests.post(url, verify=False,
                              headers=json_headers, cookies=cookies, json=data)
        else:
            r = requests.post(url, verify=False,
                              headers=headers, cookies=cookies)

    return r


# open url with chromedriver
def sending_with_chrome(options, url, delay=5):
    utils.print_debug(options, url)
    try:
        chrome_options = Options()
        chrome_options.add_argument("--headless")
        chrome_options.add_argument("--no-sandbox")
        chrome_options.add_argument("--ignore-certificate-errors")

        # select chrome path
        chrome_options.binary_location = config.get_chrome_binary()

        current_path = os.path.dirname(os.path.realpath(__file__))
        chromedrive_check = shutil.which(current_path + "/chromedriver")
        if not chromedrive_check:
            utils.print_bad("Some thing wrong with chromedriver path")
            return False

        chromedriver = current_path + '/chromedriver'
        browser = webdriver.Chrome(
            executable_path=chromedriver, options=chrome_options)

        browser.get(url)

        # wait for get the right response
        time.sleep(delay)
        response = browser.page_source
        browser.close()

        return response
    except:
        utils.print_bad("Some thing bad with Selenium")
        return False
