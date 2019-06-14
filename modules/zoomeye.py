import time
from core import sender
from core import utils

from bs4 import BeautifulSoup


class ZoomEye():
    """docstring for ZoomEye"""

    def __init__(self, options):
        self.options = options
        # setting stuff depend on search engine
        self.base_url = "https://www.zoomeye.org/"

        self.options['zoomeye_query'] = options['query']
        self.output = self.options['outdir'] + \
            "/{0}-zoomeye.txt".format(self.options['output'])

        # get jwt from config file
        self.jwt = {"Cube-Authorization": options.get('Cookies_zoomeye')}
        self.headers = {"User-Agent": "Mozilla/5.0 (X11; FreeBSD amd64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/40.0.2214.115 Safari/537.36", "Accept": "application/json", "Accept-Language": "en-US,en;q=0.5", "Accept-Encoding": "gzip, deflate", "Content-Type": "application/json"}
        
        # really do something
        utils.print_banner("Starting scraping from zoomeye")

        self.logged_in = self.check_session()
        self.initial()
        self.conclude()

    # check if the session still valid
    def check_session(self):
        utils.print_debug(self.options, "Checking session for ZoomEye")
        sess_url = 'https://www.zoomeye.org/user'

        # get jwt if it was set
        if self.jwt.get('Cube-Authorization') and self.jwt.get('Cube-Authorization') != 'None':
            self.headers['Cube-Authorization'] = self.jwt.get(
                'Cube-Authorization')

        r = sender.send_get(self.options, sess_url, headers=self.headers)

        if not r or 'login required' in r.text:
            utils.print_bad(
                "Look like ZoomEye session is invalid.")
            return False
        elif 'uuid' in r.text or 'nickname' in r.text:
            utils.print_good("Getting result as authenticated user")
            return True

        return False

    # really sending first request
    def initial(self):
        # prepare url
        query = utils.url_encode(self.options['zoomeye_query'])
        url = 'https://www.zoomeye.org/search?q={0}&t=host&p=1'.format(query)
        self.sending(url)

    # really sending request and looping through the page
    def sending(self, url):
        # sending request and return the response
        r = sender.send_get(self.options, url, headers=self.headers)
        if r:
            response = r.text

            if self.options['store_content']:
                ts = str(int(time.time()))
                raw_file = self.options['raw'] + \
                    "/zoomeye/{0}_{1}".format(utils.url_encode(
                        url.replace(self.base_url, '')).replace('/', '_'), ts)
                utils.just_write(raw_file, response)

            json_response = utils.get_json(response)
            self.analyze(json_response)

            # loop throuh pages if you're logged in
            page_num = self.get_num_pages(json_response)
            if self.logged_in and int(page_num) > 1:
                self.pages(page_num)

            # get aggs and found more result
            self.optimize(json_response)

    # parse the html and get the result
    def analyze(self, json_response):
        result = []
        # custom here
        items = json_response.get('matches')
        if not items:
            utils.print_bad("Look like we reach limit result")
            return False

        for item in items:
            external_url = item.get('portinfo').get(
                'service') + "://" + item.get('ip') + ":" + str(item.get('portinfo').get('port'))
            
            element = {
                'raw_ip': item.get('ip'),
                'raw_scheme': item.get('ip') + ":" + str(item.get('portinfo').get('port')),
                'external_url': external_url,
            }

            utils.print_debug(self.options, element)
            result.append(element)

        output = []
        for item in result:
            if item.get('external_url') and item.get('external_url') != 'N/A':
                output.append(item.get('external_url'))
            elif item.get('raw_scheme') and item.get('raw_scheme') != 'N/A':
                output.append(item.get('raw_scheme'))
            elif item.get('raw_ip') and item.get('raw_ip') != 'N/A':
                output.append(item.get('raw_ip'))

        really_data = "\n".join(output)
        print(really_data)
        utils.just_write(self.output, really_data + "\n")

    # get number of page
    def get_num_pages(self, json_response):
        results_total = int(json_response.get('total'))
        pageSize = int(json_response.get('pageSize'))
        page_num = str(int(results_total / pageSize))
        utils.print_good("Detect posible {0} pages per {1} result".format(page_num, str(results_total)))
        
        if int(page_num) > 0:
            return page_num
        return False

    # keep doing if there have many pages
    def pages(self, page_num):
        for i in range(2, int(page_num) + 1):
            utils.random_sleep(1, 2)
            utils.print_info("Get more result from page: {0}".format(str(i)))

            query = utils.url_encode(self.options['zoomeye_query'])
            url = 'https://www.zoomeye.org/search?q={0}&t=host&p={1}'.format(
                query, str(i))
            r = sender.send_get(self.options, url, headers=self.headers)

            if r.status_code == 200:
                response = r.text
                if '"msg": "forbidden"' in response:
                    utils.print_bad(
                        "Reach to the limit at page {0}".format(str(i)))
                    return
                else:
                    json_response = utils.get_json(response)
                    self.analyze(json_response)
                    self.optimize(json_response)

    # analyze for country and city for more result
    def optimize(self, json_response):
        analytics = json_response.get('aggs')
        if not analytics:
            return False

        # get analytics respoonse
        url = 'https://www.zoomeye.org/aggs/{0}'.format(analytics)
        r = sender.send_get(self.options, url, headers=self.headers)

        if r.status_code == 200:
            analytics_json = utils.get_json(r.text)
        else:
            return False

        analytics_countries = analytics_json.get('country')

        raw_query = self.options['zoomeye_query']
        clean_query = self.options['zoomeye_query']
        if 'country' in raw_query:
            country_code = utils.get_country_code(utils.url_decode(raw_query))
            # clean country and subdivisions if it exist
            clean_query = raw_query.replace(
                ' +country:', '').replace('"{0}"'.format(str(country_code)), '')

        for country_item in analytics_countries:
            utils.print_info("Optimize query by filter with coutry: {0}".format(
                country_item.get('name')))
            # loop through city
            for city in country_item.get('subdivisions'):
                real_query = clean_query + \
                    ' +country:"{0}"'.format(country_item.get('name')) + \
                    ' +subdivisions:"{0}"'.format(city.get('name'))

                query = utils.url_encode(real_query)

                url = 'https://www.zoomeye.org/search?q={0}&t=host'.format(query)
                r = sender.send_get(self.options, url, headers=self.headers)
                if r and r.status_code == 200:
                    json_response = utils.get_json(r.text)
                    self.analyze(json_response)

    # just loop though the country code without analyze
    # Dont' need it in this module
    def brute_country_code(self):
        pass

    # reauthen, currently not support reauthen for this search engine
    def do_login(self):
        pass

    # unique stuff
    def conclude(self):
        utils.just_cleanup(self.output)
