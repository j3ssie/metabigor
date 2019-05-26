import time
from core import sender
from core import utils

from bs4 import BeautifulSoup


class Shodan():
    """docstring for Shodan"""

    def __init__(self, options):
        self.options = options
        #setting stuff depend on search engine
        self.base_url = "https://shodan.io"

        self.options['shodan_query'] = options['query']
        self.cookies = {"polito": options.get('Cookies_shodan')}
        self.output = self.options['outdir'] + \
            "/{0}-shodan.txt".format(self.options['output'])

        #really do something
        utils.print_banner("Starting scraping from Shodan")

        self.logged_in = self.check_session()
        if self.logged_in:
            self.initial()
            self.conclude()
        else:
            utils.print_bad("Shodan only allowed authenticated user to query")


    # check if the session still valid
    def check_session(self):
        utils.print_debug(self.options, "Checking session for Shodaan")
        sess_url = 'https://account.shodan.io/'

        r = sender.send_get(self.options, sess_url, self.cookies)

        if r.status_code == 302 or '/login' in r.text:
            utils.print_bad(
                "Look like shodan session is invalid.")
            
            new_cookie = self.do_login()
            if new_cookie:
                utils.print_good("Reauthen success")
                self.cookies = {
                    "polito": new_cookie}
                return True

            return False
        elif r.status_code == 200:
            utils.print_good("Getting result as authenticated user")
            return True

        return False

    # really sending first request
    def initial(self):
        # prepare url
        query = utils.url_encode(self.options['shodan_query'])
        url = 'https://www.shodan.io/search?query={1}&page={0}'.format(str(1), query)

        self.sending(url)

        # brute the country
        if self.options['brute']:
            self.brute_country_code(query)

        # repeat the routine with filter by city
        query_by_cities = self.optimize(query)
        if query_by_cities:
            for item in query_by_cities:
                utils.print_info(
                    "Get more result by filter with {0} city".format(item.get('city')))
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
                    "/shodan/{0}_{1}".format(utils.url_encode(
                        url.replace(self.base_url, '')).replace('/', '_'), ts)
                utils.just_write(raw_file, response)

            soup = utils.soup(response)
            self.analyze(soup)

            # checking if there is many pages or not
            if self.logged_in and not self.options['disable_pages']:
                utils.print_info("Continue grab more pages")
                self.pages(self.get_num_pages(url))

    # parse the html and get the result
    def analyze(self, soup):
        result = []
        # custom here
        divs = soup.find_all("div", "search-result")
        for div in divs:
            element = {
                'raw_ip': 'N/A',
                'result_title': 'N/A',
                'external_url': 'N/A'
            }

            # getting sumary div
            div_sum = div.find_all("div", "search-result-summary")[0]
            element['raw_ip'] = div_sum.span.text  # ip

            div_detail = div.find_all("div", "ip")[0]
            links = div_detail.find_all("a")
            for link in links:
                if '/host/' in link.get('href'):
                    element['result_title'] = link.text

                if link.get('class') and 'fa-external-link' in link.get('class'):
                    element['external_url'] = link.get('href')

            utils.print_debug(self.options, element)
            result.append(element)

        output = []
        for item in result:
            if item.get('external_url') and item.get('external_url') != 'N/A':
                output.append(item.get('external_url'))
            elif item.get('result_title') and item.get('result_title') != 'N/A':
                output.append(item.get('result_title'))
            elif item.get('raw_ip') and item.get('raw_ip') != 'N/A':
                output.append(item.get('raw_ip'))

        really_data = "\n".join(output)
        print(really_data)
        utils.just_write(self.output, really_data + "\n")

    # get number of page
    def get_num_pages(self, url):
        summary_url = 'https://www.shodan.io/search/_summary?{0}'.format(
            utils.get_query(url))
        
        r = sender.send_get(self.options, summary_url, self.cookies)
        
        if r.status_code == 200:
            soup = utils.soup(r.text)
            results_total = soup.find_all('div', 'bignumber')[
                0].text.replace(',', '')
            page_num = str(int(int(results_total) / 10))
            utils.print_good("Detect posible {0} pages per {1} result".format(
                page_num, results_total))
            return page_num

        return False

    # keep doing if there have many pages
    def pages(self, page_num):
        for i in range(2, int(page_num) + 1):
            utils.print_info("Sleep for couple seconds because Shodan server is really strict")
            utils.random_sleep(3, 6)
            utils.print_info("Get more result from page: {0}".format(str(i)))

            query = utils.url_encode(self.options['shodan_query'])
            url = 'https://www.shodan.io/search?query={1}&page={0}'.format(
                str(i), query)

            r = sender.send_get(self.options, url, self.cookies)

            if r.status_code == 200:
                response = r.text
                if 'class="alert alert-error text-center"' in response:
                    utils.print_bad(
                        "Reach to the limit at page {0}".format(str(i)))
                    return
                else:
                    soup = utils.soup(response)
                    self.analyze(soup)

    # analyze for country and city for more result
    def optimize(self, query):
        url = 'https://www.shodan.io/search/_summary?query={0}'.format(query)
        utils.print_good("Analyze first page for more result")
        r = sender.send_get(self.options, url, self.cookies)
        
        if r.status_code == 200:
            soup = utils.soup(r.text)
        else:
            return False
        
        query_by_cities = []
        # check if query have country filter or not
        if 'country' in query:
            links = soup.find_all("a")
            country = utils.get_country_code(utils.url_decode(query))

            for link in links:
                if 'city' in link.get('href'):
                    item = {
                        'url': link.get('href'),
                        'city': link.text,
                        'country': country
                    }
                    utils.print_debug(self.options, item)
                    query_by_cities.append(item)
        else:
            links = soup.find_all("a")
            countries = []
            for link in links:
                if 'country' in link.get('href'):
                    countries.append(utils.get_country_code(utils.url_decode(link.get('href'))))
            utils.print_debug(self.options, countries)

            for country in countries:
                # seding request again to get city
                country_query = utils.url_encode(' country:"{0}"'.format(country))
                url = 'https://www.shodan.io/search/_summary?query={0}{1}'.format(
                    query, country_query)
                r1 = sender.send_get(self.options, url, self.cookies)
                utils.random_sleep(5, 8)
                utils.print_info(
                    "Sleep for couple seconds because Shodan server is really strict")
                if r1.status_code == 200:
                    soup1 = utils.soup(r1.text)
                    links = soup1.find_all("a")
                    # countries = []
                    for link in links:
                        if 'city' in link.get('href'):
                            # countries.append(utils.get_city_name(
                            #     utils.url_decode(link.get('href'))))
                            item = {
                                'url': link.get('href'),
                                'city': link.text,
                                'country': country
                            }
                            utils.print_debug(self.options, item)
                            query_by_cities.append(item)

        utils.print_debug(self.options, query_by_cities)
        return query_by_cities


    # just loop though the country code without analyze 
    def brute_country_code(self, query):
        utils.print_info("Brute the query with country code")
        if 'country' in query:
            raw_query = query.replace(utils.get_country_code(utils.url_decode(query)), '[replace]')
        else:
            raw_query += utils.url_encode(' country:"[replace]"')

        for country_code in utils.full_country_code:
            query = raw_query.replace('[replace]', country_code)
            url = 'https://www.shodan.io/search?query={1}&page={0}'.format(
                str(1), query)
            self.sending(url)

    # reauthen
    def do_login(self):
        utils.print_info("Reauthen using credentials from: {0}".format(
            self.options.get('config')))

        login_url = 'https://account.shodan.io/login'
        r = sender.send_get(self.options, login_url, cookies=None)

        if r.status_code == 200:
            cookies = r.cookies
            form = utils.soup(r.text).find_all("form")
            if form:
                inputs = form[0].findChildren('input')

            for tag in inputs:
                if tag.get('name') == 'csrf_token':
                    csrf_token = tag.get('value')

            username, password = utils.get_cred(self.options, source='shodan')
            data = {"username": username, "password": password, "grant_type": "password",
                          "continue": "https://www.shodan.io/", "csrf_token": csrf_token, "login_submit": "Login"}

            really_login_url = 'https://account.shodan.io/login'
            r1 = sender.send_post(
                self.options, really_login_url, cookies, data, follow=False)

            if r1.status_code == 302:
                for item in r1.cookies.items():
                    if item.get('polito'):
                        shodan_cookies = item.get('polito')
                        utils.set_session(
                            self.options, shodan_cookies, source='shodan')
                        return shodan_cookies
        
        return False


    # unique stuff
    def conclude(self):
        utils.just_cleanup(self.output)
