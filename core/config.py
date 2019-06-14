import os
import sys
import shutil
import platform
import requests
import zipfile
from urllib.parse import urlparse
from configparser import ConfigParser, ExtendedInterpolation

from . import utils
from . import sender

# Console colors
W = '\033[1;0m'   # white
R = '\033[1;31m'  # red
G = '\033[1;32m'  # green
B = '\033[1;34m'  # blue
Y = '\033[1;93m'  # yellow
P = '\033[1;35m'  # purple
C = '\033[1;36m'  # cyan
GR = '\033[1;37m'  # gray
colors = [G, R, B, P, C, GR]

# Possible paths for Google Chrome on porpular OS
chrome_paths = [
    "/usr/bin/chromium",
    "/usr/bin/google-chrome-stable",
    "/usr/bin/google-chrome",
    "/Applications/Google Chrome.app/Contents/MacOS/Google Chrome",
    "/Applications/Chromium.app/Contents/MacOS/Chromium",
    "C:/Program Files (x86)/Google/Chrome/Application/chrome.exe",
]


def get_chrome_binary():
    for chrome_binary in chrome_paths:
        if os.path.isfile(chrome_binary):
            return chrome_binary

    utils.print_bad("Not found Chrome binary on your system")


# get version of your chrome
def get_chrome_version():
    chrome_binary = get_chrome_binary()
    chrome_version = os.popen(
        '"{0}" -version'.format(chrome_binary)).read().lower()
    chrome_app = os.path.basename(os.path.normpath(chrome_binary)).lower()
    # just get some main release
    version = chrome_version.split(chrome_app)[1].strip().split(' ')[0]
    relative_version = '.'.join(version.split('.')[:2])
    return relative_version


def install_webdrive():
    current_path = os.path.dirname(os.path.realpath(__file__))
    chromedrive_check = shutil.which(current_path + "/chromedriver")

    if chromedrive_check:
        return current_path + "/chromedriver"

    utils.print_info("Download chromedriver")
    relative_version = get_chrome_version()

    if float(relative_version) < 73:
        utils.print_info("Unsupport Chromium version support detected: {0}".format(relative_version))
        utils.print_bad("You need to update your Chromium.(e.g: sudo apt install chromium -y)")
        return

    chrome_driver_url = 'https://sites.google.com/a/chromium.org/chromedriver/downloads'
    # predefine download url
    download_url = 'https://chromedriver.storage.googleapis.com/index.html?path=74.0.3729.6/'
    r = requests.get(chrome_driver_url, allow_redirects=True)
    if r.status_code == 200:
        soup = utils.soup(r.text)
        lis = soup.find_all("li")
        for li in lis:
            if 'If you are using Chrome version' in li.text:
                if relative_version in li.text:
                    download_url = li.a.get('href')

    parsed_url = urlparse(download_url)
    zip_chromdriver = parsed_url.scheme + "://" + parsed_url.hostname + \
        "/" + parsed_url.query.split('=')[1]
    
    os_check = platform.platform()
    if 'Darwin' in os_check:
        zip_chromdriver += "chromedriver_mac64.zip"
    elif 'Win' in os_check:
        zip_chromdriver += "chromedriver_win32.zip"
    elif 'Linux' in os_check:
        zip_chromdriver += "chromedriver_linux64.zip"
    else:
        zip_chromdriver += "chromedriver_linux64.zip"

    # utils.print_info("Download: {0}".format(zip_chromdriver))
    r3 = requests.get(zip_chromdriver, allow_redirects=True)

    open(current_path + "/chromedriver.zip", 'wb').write(r3.content)

    with open(current_path + '/chromedriver.zip', 'rb') as f:
        z = zipfile.ZipFile(f)
        for name in z.namelist():
            z.extract(name, current_path)

    os.chmod(current_path + "/chromedriver", 0o775)
    if not shutil.which(current_path + "/chromedriver"):
        utils.print_bad("Some thing wrong with chromedriver")
        sys.exit(-1)


def update():
    os.system('git fetch --all && git reset --hard origin/master')
    sys.exit(0)


def config(options, args):
    # just in case we need real browser
    # install_webdrive()
    config_path = options['config']

    cwd = str(os.getcwd())
    # loading config file
    if os.path.isfile(config_path):
        utils.print_info('Loading session from: {0}'.format(config_path))
        # config to logging some output
        config = ConfigParser(interpolation=ExtendedInterpolation())
        config.read(config_path)
    else:
        utils.print_info('New config file created: {0}'.format(config_path))
        shutil.copyfile(cwd + '/sample-config.conf', config_path)
        config = ConfigParser(interpolation=ExtendedInterpolation())
        config.read(config_path)

    if args.outdir:
        options['outdir'] = args.outdir

    options['module'] = args.module if args.module else None
    options['target'] = args.target if args.target else None
    options['debug'] = args.debug if args.debug else None
    options['relatively'] = args.relatively if args.relatively else False
    options['brute'] = args.brute if args.brute else None
    options['disable_pages'] = args.disable_pages if args.disable_pages else None
    options['store_content'] = args.store_content if args.store_content else None

    if args.output:
        options['output'] = args.output

    if args.proxy:
        options['proxy'] = {
            'http': args.proxy,
            'https': args.proxy
        }

    # create output directory and raw html directory
    options['outdir'] = args.outdir
    utils.make_directory(args.outdir)
    options['raw'] = args.raw

    if options['store_content']:
        utils.make_directory(args.raw)
        utils.make_directory(args.raw + "/fofa/")
        utils.make_directory(args.raw + "/shodan/")
        utils.make_directory(args.raw + "/censys/")

    # source search engine
    options['source'] = args.source if args.source else None
    options['source_list'] = args.source_list if args.source_list else None

    if args.cookies:
        if options['source']:
            config.set('Cookies', options['source'], str(args.cookies))
            utils.print_good("Set cookies for {0}".format(options['source']))
        else:
            utils.print_bad(
                "You need to specific the source for set the cookies")

    # save the config
    with open(config_path, 'w') as configfile:
        config.write(configfile)

    config = ConfigParser(interpolation=ExtendedInterpolation())
    config.read(config_path)
    sections = config.sections()

    # fofa_cookies = config.get('Cookies', fofa)
    for sec in sections:
        for key in config[sec]:
            options[sec + "_" + key] = config.get(sec, key)

    return options


def banner(__author__, __version__):
    print('''{1}
                        '@:           +#.  
                      .@@@@@        ;@@@@+ 
                      @@. ;@';#@@#;`@@``+@.
                      @'   @@@@@@@@@@.  `@#
                      @+  @@#.    .#@@  `@+
                      #@'@@;        :@@.@@ 
                       @@@;  {0}.,{1}      :@@@: 
                        @#  {0}#@@@`{1}     #@`  
                       ;@. {0}.@+,@@{1}     .@;  
                       #@  {0},@, @@{1}      @#  
                       @@   {0}@@@@;{1}      @@  
                       @@   {0}`@@;{1}       @@  
                       @@           .++@#  
                      #@@.         +@@@@@. 
                     `@'@#        '@#  :@@ 
                     +@`+@:       @@    :@:
                     @@  @@:      @'     @@
                     ;@.  @@#.    @+    `@+
                      @#   +@@@@@@@@`   #@`
                      '@#. `+@@@@#;@@+;@@+ 
                       '@@@@@#     .@@@@'  
                         ;@+`'''.format(GR, G))
                         
    print('''            
                  Metabigor {2}{0}{3} {4}by {2}{1}{4}

                          ¯\_(ツ)_/¯{3}
                              '''.format(__version__, __author__, P, GR, G))


def custom_help():
    utils.print_info(
        "Visit this page for complete usage: https://github.com/j3ssie/Metabigor/wiki")
    print('''{1}
{2}[*]{0} Setup session{1}
===============
Do command below or direct modify config.conf file
./metabigor.py -s shodan --cookies=<content of polito cookie>
./metabigor.py -s censys --cookies=<content of auth_tkt cookie>
./metabigor.py -s fofa --cookies=<content of _fofapro_ars_session cookie>
./metabigor.py -s zoomeye --cookies=<content of Cube-Authorization header>


{2}[*]{0} Basic Usage{1}
===============
./metabigor.py -s <source> -q '<your_query>' [options]
./metabigor.py -S <json file of multi source> [options]
./metabigor.py -m <module> -t <target> [options]

{2}[*]{0} More Options{1}
===============
  -d OUTDIR, --outdir OUTDIR
                        Directory output
  -o OUTPUT, --output OUTPUT
                        Output file name
  --raw RAW             Directory to store raw query
  --proxy PROXY         Proxy for doing request to search engine e.g:
                        http://127.0.0.1:8080
  -b                    Auto brute force the country code
  --disable_pages       Don't loop though the pages
  --store_content       Store the raw HTML souce or not
  -M                    Print available module and search engine supported
  --hh                  Print this message
  --debug               Print debug output


{2}[*]{0} Example commands{1}
===============
./metabigor.py -s fofa -q 'title="Dashboard - Confluence" && body=".org"'
./metabigor.py -s zoomeye -q 'app:"tomcat"'
./metabigor.py -s shodan -q 'port:"3389" os:"Windows"' --debug 
./metabigor.py -s shodan -Q list_of_query.txt --debug -o rdp.txt  -b --disable_pages
./metabigor.py -s censys -q '(scada) AND protocols: "502/modbus"' -o something  --debug --proxy socks4://127.0.0.1:9050

./metabigor.py -m exploit -t 'nginx|1.0'  --debug

            '''.format(G, GR, B))
    sys.exit(0)


def modules_help():
    utils.print_info(
        "Visit this page for complete usage: https://github.com/j3ssie/Metabigor/wiki")
    print('''{1}
{2}[*]{0} Available modules{1}
===============
  custom         Do query from specific search engine (default mode)
  exploit        Do query from multiple source to get CVE or exploit about app of software

{2}[*]{0} Usage{1}
===============
./metabigor.py -m exploit -t '<app>|version' [options]
./metabigor.py -m custom -q '<query>'

{2}[*]{0} Example commands{1}
===============
./metabigor.py -m custom -s shodan -Q list_of_query.txt --debug -o rdp.txt  -b --disable_pages

./metabigor.py -m exploit -t 'nginx|1.0'  --debug
./metabigor.py -m exploit -t 'tomcat|7' -d /tmp/ -o tomcat --debug

            '''.format(G, GR, B))
    sys.exit(0)
