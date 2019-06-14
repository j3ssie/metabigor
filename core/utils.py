import os 
import re 
import json
import base64 
import random
import time
import urllib.parse
from configparser import ConfigParser, ExtendedInterpolation
from bs4 import BeautifulSoup

# Console colors
W = '\033[1;0m'   # white
R = '\033[1;31m'  # red
G = '\033[1;32m'  # green
O = '\033[1;33m'  # orange
B = '\033[1;34m'  # blue
Y = '\033[1;93m'  # yellow
P = '\033[1;35m'  # purple
C = '\033[1;36m'  # cyan
GR = '\033[1;37m'  # gray
colors = [G, R, B, P, C, O, GR]

info = '{0}[*]{1} '.format(B, GR)
ques = '{0}[?]{1} '.format(C, GR)
bad = '{0}[-]{1} '.format(R, GR)
good = '{0}[+]{1} '.format(G, GR)

debug = '{1}[{0}DEBUG{1}]'.format(G, GR)

full_country_code = ['AF', 'AL', 'DZ', 'AS', 'AD', 'AO', 'AI', 'AQ', 'AG', 'AR', 'AM', 'AW', 'AU', 'AT', 'AZ', 'BS', 'BH', 'BD', 'BB', 'BY', 'BE', 'BZ', 'BJ', 'BM', 'BT', 'BO', 'BA', 'BW', 'BV', 'BR', 'IO', 'BN', 'BG', 'BF', 'BI', 'KH', 'CM', 'CA', 'CV', 'KY', 'CF', 'TD', 'CL', 'CN', 'CX', 'CC', 'CO', 'KM', 'CG', 'CD', 'CK', 'CR', 'CI', 'HR', 'CU', 'CY', 'CZ', 'DK', 'DJ', 'DM', 'DO', 'EC', 'EG', 'EH', 'SV', 'GQ', 'ER', 'EE', 'ET', 'FK', 'FO', 'FJ', 'FI', 'FR', 'GF', 'PF', 'TF', 'GA', 'GM', 'GE', 'DE', 'GH', 'GI', 'GR', 'GL', 'GD', 'GP', 'GU', 'GT', 'GN', 'GW', 'GY', 'HT', 'HM', 'HN', 'HK', 'HU', 'IS', 'IN', 'ID', 'IR', 'IQ', 'IE', 'IL', 'IT', 'JM', 'JP', 'JO', 'KZ', 'KE', 'KI', 'KP', 'KR', 'KW', 'KG', 'LA', 'LV', 'LB', 'LS', 'LR', 'LY', 'LI', 'LT', 'LU', 'MO', 'MK', 'MG', 'MW', 'MY', 'MV', 'ML', 'MT', 'MH', 'MQ', 'MR', 'MU', 'YT', 'MX', 'FM', 'MD', 'MC', 'MN', 'MS', 'MA', 'MZ', 'MM', 'NA', 'NR', 'NP', 'NL', 'AN', 'NC', 'NZ', 'NI', 'NE', 'NG', 'NU', 'NF', 'MP', 'NO', 'OM', 'PK', 'PW', 'PS', 'PA', 'PG', 'PY', 'PE', 'PH', 'PN', 'PL', 'PT', 'PR', 'QA', 'RE', 'RO', 'RU', 'RW', 'SH', 'KN', 'LC', 'PM', 'VC', 'WS', 'SM', 'ST', 'SA', 'SN', 'CS', 'SC', 'SL', 'SG', 'SK', 'SI', 'SB', 'SO', 'ZA', 'GS', 'ES', 'LK', 'SD', 'SR', 'SJ', 'SZ', 'SE', 'CH', 'SY', 'TW', 'TJ', 'TZ', 'TH', 'TL', 'TG', 'TK', 'TO', 'TT', 'TN', 'TR', 'TM', 'TC', 'TV', 'UG', 'UA', 'AE', 'GB', 'US', 'UM', 'UY', 'UZ', 'VE', 'VU', 'VN', 'VG', 'VI', 'WF', 'YE', 'ZW']


'''
 Beatiful print
'''

def print_debug(options, text):
    if options['debug']:
        print(debug, text)


def print_banner(text):
    print('{1}--~~~[ {2}{0}{1} ]~~~--'.format(text, G, C))


def print_info(text):
    print(info + text)


def print_ques(text):
    print(ques + text)


def print_good(text):
    print(good + text)


def print_bad(text):
    print(bad + text)


def check_output(output):
    abs_path = os.path.abspath(output)
    print('{1}--==[ Check the output: {2}{0}'.format(abs_path, G, P))


'''
 String utils
'''


def random_sleep(min=2, max=5):
    time.sleep(random.randint(min, max))


# just beatiful soup the html
def soup(html):
    soup = BeautifulSoup(html, "lxml")
    return soup

def get_json(text):
    return json.loads(text)


def get_query(url):
    return urllib.parse.urlparse(url).query


# get country code from query
def get_country_code(query, source='shodan'):
    try:
        if source == 'shodan':
            m = re.search('country:\"[a-zA-Z]+\"', query)
            country_code = m.group().split(':')[1].strip('"')
        elif source == 'fofa':
            m = re.search('country=[a-zA-Z]+', query)
            country_code = m.group().split('=')[1].strip('"')

        elif source == 'censys':
            m = re.search(
                '(country\_code:.[a-zA-Z]+)|(country:.([\"]?)[a-zA-Z]+.[a-zA-Z]+([\"]?))', query)
            country_code = m.group()

        return country_code
    except:
        return False


# get city name from query
def get_city_name(query, source='shodan'):
    if source == 'shodan':
        m = re.search('city:\"[a-zA-Z]+\"', query)
        city_code = m.group().split(':')[1].strip('"')

    elif source == 'fofa':
        m = re.search('city=[a-zA-Z]+', query)
        country_code = m.group().split('=')[1].strip('"')

    return city_code

# get cve number
def get_cve(source):
    m = re.search('CVE-\d{4}-\d{4,7}', source)
    if m:
        cve = m.group()
        return cve
    else:
        return 'N/A'


def url_encode(string_in):
    return urllib.parse.quote(string_in)


def url_decode(string_in):
    return urllib.parse.unquote(string_in)


def just_b64_encode(string_in):
    return base64.b64encode(string_in.encode()).decode()


def just_b64_decode(string_in):
    return base64.b64decode(string_in.encode()).decode()


'''
 File utils
'''


# get credentials
def get_cred(options, source):
    config_file = options.get('config')
    config = ConfigParser(interpolation=ExtendedInterpolation())
    config.read(config_file)

    if 'fofa' in source:
        cred = config.get('Credentials', 'fofa')
    if 'shodan' in source:
        cred = config.get('Credentials', 'shodan')
    if 'censys' in source:
        cred = config.get('Credentials', 'censys')
    
    print_debug(options, cred)
    username = cred.split(':')[0].strip()
    password = cred.split(':')[1].strip()

    return username, password


# set session 
def set_session(options, cookies, source):
    print_debug(options, cookies)
    config_file = options.get('config')
    config = ConfigParser(interpolation=ExtendedInterpolation())
    config.read(config_file)

    if 'fofa' in source:
        config.set('Cookies', 'fofa', cookies)
    if 'shodan' in source:
        config.set('Cookies', 'shodan', cookies)
    if 'censys' in source:
        config.set('Cookies', 'censys', cookies)

    with open(config_file, 'w') as configfile:
        config.write(configfile)


def make_directory(directory):
    if not os.path.exists(directory):
        print_good('Make new directory: {0}'.format(directory))
        os.makedirs(directory)


def just_read(filename):
    if os.path.isfile(filename):
        with open(filename, 'r') as f:
            data = f.read()
        return data

    return False


def just_write(filename, data, is_json=False, verbose=False):
    real_path = os.path.normpath(filename)
    try:
        print_good("Writing {0}".format(filename)) if verbose else None
        if is_json:
            with open(real_path, 'a+') as f:
                json.dump(data, f)
        else:
            with open(real_path, 'a+') as f:
                f.write(data)
    except:
        print_bad("Writing fail: {0}".format(real_path))
        return False


# unique and strip the blank line
def just_cleanup(filename):
    check_output(filename)
    if os.path.isfile(filename):
        with open(filename, 'r') as f:
            raw = f.read().splitlines()

        data = [x for x in raw if str(x) != '']

        with open(filename, 'w+') as o:
            for item in set(data):
                o.write(item + "\n")

    return False

