<p align="center">
  <img alt="Metabigor" src="https://image.flaticon.com/icons/svg/1774/1774457.svg" height="140" />
  <p align="center">Command line Search Engine without any API key</p>
  <p align="center">
    <a href="https://github.com/j3ssie/Metabigor"><img alt="python" src="https://img.shields.io/badge/python-3.6%2B-blue.svg"></a>
    <a href=""><img alt="Software License" src="https://img.shields.io/badge/license-MIT-brightgreen.svg?style=flat-square"></a>
    <a href=""><img alt="tested" src="https://img.shields.io/badge/tested-Linux%2fOSX-red.svg"></a>
  </p>
</p>

## What is Metabigor?
Metabigor allows you do query from command line to awesome Search Engines (like Shodan, Censys, Fofa, etc) without any API key.

## But Why Metabigor?
* Don't use your API key so you don't have to worry about litmit of API quotation.**\***

* Do query from command line without Premium account.**\***

* Get more result without Premium account. **\***

* But I have an Premium account why do I need this shit? 
    * Again Metabigor will not lose your API quotation.
    * Your query will optimized so you gonna get more result than using it by hand or API key.
    * Never get duplicate result.**\***

## How it works?
Metabigor gonna use your cookie or not to simulate search from browser and optimize the query to get more result.

## Search Engine currently supported
- [x] Shodan.
- [x] Censys.
- [x] Fofa Pro.

## Installation
```
git clone https://github.com/j3ssie/Metabigor
cd Metabigor
pip3 install -r requirements.txt
```

## Demo
[![asciicast](https://asciinema.org/a/jaARv3sMSOVYQ1yOsjeKZp8Ek.svg)](https://asciinema.org/a/jaARv3sMSOVYQ1yOsjeKZp8Ek)

## How to use

### Basic Usage

```
./metabigor.py -s <source> -q '<your_query>' [options]
```

Check out the [Advanced Usage](https://github.com/j3ssie/Metabigor/wiki/Advanced-Usage) to explore some awesome options

### Example commands

```
./metabigor.py -s fofa -q 'title="Dashboard - Confluence" && body=".org"' 
```

```
./metabigor.py -s fofa -q 'title="Dashboard - Confluence" && body=".org"' -b --disable_pages
```

```
./metabigor.py -s shodan -q 'port:"3389" os:"Windows"' --debug
```

### Options
```
[*] Setup session
===============
Do command below or direct modify config.conf file
./metabigor.py -s shodan --cookies=<content of polito cookie>
./metabigor.py -s censys --cookies=<content of auth_tkt cookie>
./metabigor.py -s fofa --cookies=<content of _fofapro_ars_session cookie>


[*] Basic Usage
===============
./metabigor.py -s <source> -q '<your_query>' [options]

[*] More Options
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
  --hh                  Print this message
  --debug               Print debug output


[*] Example commands
===============
./metabigor.py -s fofa -q 'title="Dashboard - Confluence" && body=".org"' -b
./metabigor.py -s fofa -q 'title="Dashboard - Confluence" && body=".org"' -b --disable_pages

./metabigor.py -s shodan -q 'port:"3389" os:"Windows"' --debug
./metabigor.py -s shodan -Q list_of_query.txt --debug -o rdp.txt

./metabigor.py -s censys -q '(scada) AND protocols: "502/modbus"' -o something  --debug --proxy socks4://127.0.0.1:9050

```


### TODO
* Predine query to do specific task like subdomain scan, portscan 
* Adding more search engine.
  * ZoomEye
  * Baidu


## Credits

Logo from [flaticon](https://www.flaticon.com/free-icon/metabolism_1774457) by [Vitaly Gorbachev
](https://www.flaticon.com/authors/vitaly-gorbachev) and ascii logo converted by [picascii](http://picascii.com/)

## Disclaimer

This tool is for educational purposes only. You are responsible for your own actions. If you mess something up or break any laws while using this software, it's your fault, and your fault only.

## Contact

[@j3ssiejjj](https://twitter.com/j3ssiejjj)
