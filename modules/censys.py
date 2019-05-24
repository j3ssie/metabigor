import time
from core import sender
from core import utils

from bs4 import BeautifulSoup


class Censys():
    """docstring for Censys"""

    def __init__(self, options):
        self.options = options
        #setting stuff depend on search engine
        self.base_url = "https://censys.io"

        self.options['censys_query'] = options['query']
        self.cookies = {"auth_tkt": options.get('Cookies_censys')}
        self.output = self.options['outdir'] + \
            "/{0}-censys.txt".format(self.options['output'])

        utils.print_banner("Starting scraping from Censys")
        self.logged_in = self.check_session()
        # really do something
        self.initial()
        self.conclude()
        

    # check if the session still valid
    def check_session(self):
        utils.print_debug(self.options, "Checking session for Censys")
        sess_url = 'https://censys.io/account'

        r = sender.send_get(self.options, sess_url, self.cookies)

        if r.status_code == 302 or '/login' in r.text:
            utils.print_bad(
                "Look like Censys session is invalid.")
            new_cookie = self.do_login()
            if new_cookie:
                utils.print_good("Reauthen success")
                self.cookies = {
                    "auth_tkt": new_cookie}
                return True
            
            return False
        elif r.status_code == 200:
            utils.print_good("Getting result as authenticated user")
            return True

        return False

    # really sending first request
    def initial(self):
        # prepare url
        query = utils.url_encode(self.options['censys_query'])
        url = 'https://censys.io/ipv4/_search?q={1}&page={0}'.format(
            str(1), query)

        self.sending(url)

        # brute the country
        if self.options['brute']:
            self.brute_country_code(query)

        # repeat the routine with filter by city
        query_by_countries = self.optimize(query)
        if query_by_countries:
            for item in query_by_countries:
                utils.print_info(
                    "Get more result by filter with {0} country".format(item.get('country')))
                self.sending(item.get('url'))

    # really sending request and looping through the page
    def sending(self, url):
        # sending request and return the response
        r = sender.send_get(self.options, url, self.cookies)
        if r:
            response = r.text
            if self.options['store_content']:
                ts = str(int(time.time()))
                raw_file = self.options['raw'] + \
                    "/censys/{0}_{1}".format(utils.url_encode(
                        url.replace(self.base_url, '')).replace('/', '_'), ts)
                utils.just_write(raw_file, response)

            soup = utils.soup(response)
            self.analyze(soup)

            # checking if there is many pages or not
            if not self.options['disable_pages']:
                utils.print_info("Continue grab more pages")
                self.pages(self.get_num_pages(soup))

    # parse the html and get the result
    def analyze(self, soup):
        result = []
        # custom here
        divs = soup.findAll(True, {'class': ['SearchResult', 'result']})
        for div in divs:
            element = {
                'raw_ip': 'N/A',
                'result_title': 'N/A',
                'external_url': 'N/A'
            }

            # getting sumary div
            link_sum = div.find_all("a", "SearchResult__title-text")[0]
            element['raw_ip'] = link_sum.get('href').replace('/ipv4/', '')  # ip
            element['external_url'] = link_sum.get(
                'href').replace('/ipv4/', '')

            element['result_title'] = link_sum.span.text.replace(
                '(', '').replace(')', '')

            utils.print_debug(self.options, element)
            result.append(element)

        output = []
        for item in result:
            if item.get('raw_ip'):
                output.append(item.get('external_url'))
            elif item.get('external_url'):
                output.append(item.get('raw_ip'))
            elif item.get('result_title'):
                output.append(item.get('result_title'))

        really_data = "\n".join(output)
        print(really_data)
        utils.just_write(self.output, really_data + "\n")

    # get number of page
    def get_num_pages(self, soup):
        # soup = utils.soup(r.text)
        summary_tag = soup.find_all(
            'span', 'SearchResultSectionHeader__statistic')
        if len(summary_tag) == 0:
            return False

        for tag in summary_tag:
            if 'Page:' in tag.text:
                page_num = tag.text.split('Page: ')[1].split('/')[1]

            if 'Results:' in tag.text:
                results_total = tag.text.split('Results: ')[1].replace(',','')

        utils.print_good("Detect posible {0} pages per {1} result".format(
            page_num, results_total))
        return page_num

    # keep doing if there have many pages
    def pages(self, page_num):
        for i in range(2, int(page_num) + 1):
            utils.print_info("Get more result from page: {0}".format(str(i)))
            utils.random_sleep(1, 2)

            query = utils.url_encode(self.options['censys_query'])
            url = 'https://censys.io/ipv4/_search?q={1}&page={0}'.format(
                str(i), query)

            r = sender.send_get(self.options, url, self.cookies)
            if r.status_code == 200:
                response = r.text
                if 'class="alert alert-danger"' in response:
                    utils.print_bad(
                        "Reach to the limit at page {0}".format(str(i)))
                    return
                else:
                    soup = utils.soup(response)
                    self.analyze(soup)

    # analyze for country and city for more result
    def optimize(self, query):
        utils.print_good("Analyze metadata page for more result")

        raw_query = utils.url_decode(query)
        if 'location.country' in raw_query:
            country = utils.get_country_code(raw_query, source='censys')
            query = raw_query.replace(country, '').replace(
                'AND ' + country, '').replace('and ' + country, '')

        url = 'https://censys.io/ipv4/metadata?q={0}'.format(query)
        r = sender.send_get(self.options, url, self.cookies)

        if r.status_code == 200:
            soup = utils.soup(r.text)
        else:
            return False

        query_by_countries = []
        # check if query have country filter or not
        divs = soup.find_all("div", 'left-table')
        country_tables = []
        for div in divs:
            if 'Country Breakdown' in div.h6.text:
                country_tables = div.find_all('tr')

        for row in country_tables:
            item = {
                'url': 'N/A',
                'country': 'N/A'
            }

            tds = row.find('td')
            for td in tds:
                if td.findChildren('a'):
                    item['url'] = self.base_url + td.a.get('href')
                    item['country'] = td.a.text
                query_by_countries.append(item)

        utils.print_debug(self.options, query_by_countries)
        return query_by_countries

    # just loop though the country code without analyze
    def brute_country_code(self, query):
        utils.print_info("Brute the query with country code")
        # clean the country filter
        raw_query = utils.url_decode(query)
        if 'location.country' in raw_query:
            country = utils.get_country_code(raw_query, source='censys')
            query = raw_query.replace(country, '').replace(
                'AND ' + country, '').replace('and ' + country, '')

        raw_query += ' and location.country_code:"[replace]"'
        for country_code in utils.full_country_code:
            query = utils.url_encode(
                raw_query.replace('[replace]', country_code))
            url = 'https://censys.io/ipv4/_search?q={1}&page={0}'.format(str(1), query)
            self.sending(url)

    def do_login(self):
        utils.print_info("Reauthen using credentials from: {0}".format(
            self.options.get('config')))

        login_url = 'https://censys.io/login'
        r = sender.send_get(self.options, login_url, cookies=None)

        if r.status_code == 200:
            cookies = r.cookies
            form = utils.soup(r.text).find_all("form")
            if form:
                inputs = form[0].findChildren('input')

            for tag in inputs:
                if tag.get('name') == 'csrf_token':
                    csrf_token = tag.get('value')

            username, password = utils.get_cred(self.options, source='censys')

            data = {"csrf_token": csrf_token, "came_from": "/",
                    "from_censys_owned_external": "False", "login": username, "password": password}

            really_login_url = 'https://censys.io/login'
            r1 = sender.send_post(
                self.options, really_login_url, cookies, data, follow=False)

            if r1.status_code == 302:
                for item in r1.cookies.items():
                    if item.get('auth_tkt'):
                        censys_cookies = item.get('auth_tkt')
                        utils.set_session(
                            self.options, censys_cookies, source='censys')
                        return censys_cookies
        return False


    # unique stuff
    def conclude(self):
        utils.just_cleanup(self.output)
