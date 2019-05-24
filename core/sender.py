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

from configparser import ConfigParser, ExtendedInterpolation

from . import config
from . import utils


##### Sending stuff
normal_headers = {"User-Agent": "Mozilla/5.0 (X11; FreeBSD amd64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/40.0.2214.115 Safari/537.36", "Accept": "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8", "Accept-Language": "en-US,en;q=0.5", "Accept-Encoding": "gzip, deflate", "Connection": "close"}

post_headers = {"User-Agent": "Mozilla/5.0 (X11; FreeBSD amd64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/40.0.2214.115 Safari/537.36", "Accept": "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8", "Accept-Language": "en-US,en;q=0.5",
                "Accept-Encoding": "gzip, deflate", "Content-Type": "application/x-www-form-urlencoded", "Connection": "close", "Upgrade-Insecure-Requests": "1"}


def send_get(options, url, cookies, headers=normal_headers):
    utils.print_debug(options, url)
    if options.get('proxy'):
        r = requests.get(url, verify=False, headers=headers,
                         cookies=cookies, proxies=options.get('proxy'))
    else:
        r = requests.get(url, verify=False, headers=headers, cookies=cookies)

    return r


def send_post(options, url, cookies, data, headers=post_headers, follow=True):
    utils.print_debug(options, url)
    utils.print_debug(options, data)

    if options.get('proxy'):
        r = requests.post(url, allow_redirects=follow, verify=False,
                          headers=headers, cookies=cookies, data=data, proxies=options.get('proxy'))
        
    else:
        r = requests.post(url, allow_redirects=follow, verify=False, headers=headers,
                          cookies=cookies, data=data)

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
