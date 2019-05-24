import time
from core import sender
from core import utils


class Fofa():
    """docstring for Fofa"""
    def __init__(self, options):
        self.options = options
        # setting stuff depend on search engine
        self.base_url = "https://fofa.so"
        self.options['fofa_query'] = options['query']
        self.cookies = {"_fofapro_ars_session": options.get('Cookies_fofa')}
        self.output = self.options['outdir'] + \
            "/{0}-fofa.txt".format(self.options['output'])

        # really do something
        utils.print_banner("Starting scraping from Fofa Pro")

        self.logged_in = self.check_session()

        self.initial()
        self.conclude()
    
    # check if the session still valid
    def check_session(self):
        utils.print_debug(self.options, "Checking session for FoFa")
        sess_url = 'https://fofa.so/user/users/info'
        r = sender.send_get(self.options, sess_url, self.cookies)

        if r.status_code == 302 or '/login' in r.text:
            utils.print_bad("Look like fofa session is invalid. You gonna get very litlle result")
            new_cookie = self.do_login()
            if new_cookie:
                utils.print_good("Reauthen success")
                self.cookies = {
                    "_fofapro_ars_session": new_cookie}

            return False
        elif r.status_code == 200:
            utils.print_good("Getting result as authenticated user")
            return True

        return False

    # really sending first request
    def initial(self):
        # prepare url
        query = utils.url_encode(utils.just_b64_encode(self.options['fofa_query']))
        url = 'https://fofa.so/result?page={0}&qbase64={1}'.format(
            str(1), query)

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
                time.sleep(1)

    # really sending request and looping through the page
    def sending(self, url):
        # sending request and return the response
        utils.print_debug(self.options, url)

        r = sender.send_get(self.options, url, self.cookies)
        if r:
            response = r.text
            if self.options['store_content']:
                ts = str(int(time.time()))
                raw_file = self.options['raw'] + \
                    "/fofa/{0}_{1}".format(utils.url_encode(
                        url.replace(self.base_url, '')).replace('/', '_'), ts)
                utils.just_write(raw_file, response)

            soup = utils.soup(response)
            self.analyze(soup)
            # checking if there is many pages or not
            page_num = self.check_pages(soup)
            # if you're log in and have many results
            if page_num and self.logged_in and not self.options['disable_pages']:
                utils.print_info("Continue grab more pages")
                self.pages(page_num)

    # parse the html and get the result
    def analyze(self, soup):
        result = []
        # custom here
        divs = soup.find_all("div", "list_mod_t")
        for div in divs:
            result = div.a.get('href')
            # don't know why sometimes we get this false positive
            if '/result?qbase64=' not in result:
                print(result)
                result.append(result)
        utils.just_write(self.output, "\n".join(result) + "\n")


    # checking if there is many pages or not
    def check_pages(self, soup):
        summary_div = soup.find_all("div", "list_jg")
        # detect pages
        if len(summary_div) > 0:
            raw_summary = summary_div[0].text
            # utils.print_debug(self.options, raw_summary)
            results_total = raw_summary.split(
                'Total results: ')[1].split(' (IP results:')[0]

            page_num = str(int(int(results_total.replace(',', '')) / 10))
            utils.print_good("Detect {0} results and {1} pages".format(results_total, page_num))
            return page_num

        return False

    # keep doing if there have many pages
    def pages(self, page_num):
        for i in range(2, int(page_num) + 1):
            utils.print_info("Get more result from page: {0}".format(str(i)))
  
            query = utils.url_encode(
                        utils.just_b64_encode(self.options['fofa_query']))
            url = 'https://fofa.so/result?page={0}&qbase64={1}'.format(
                str(i), query)
            utils.print_debug(self.options, url)
            r = sender.send_get(self.options, url, self.cookies)

            if r.status_code == 200:
                response = r.text 
                if 'class="error"' in response:
                    utils.print_bad("Reach to the limit at page {0}".format(str(i)))
                    return
                else:
                    soup = utils.soup(response)
                    self.analyze(soup)

    # analyze for country and city for more result
    def optimize(self, query):
        utils.print_good("Analyze result by country and city for more result")
        # custom headers for stats 
        custom_headers = {"User-Agent": "Mozilla/5.0 (X11; FreeBSD amd64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/40.0.2214.115 Safari/537.36", "Accept": "text/html, application/javascript, application/ecmascript, application/x-ecmascript, */*; q=0.01", "Accept-Language": "en-US,en;q=0.5", "Accept-Encoding": "gzip, deflate", "X-Requested-With": "XMLHttpRequest", "Connection": "close"}
        url = 'https://fofa.so/search/result_stats?qbase64={0}'.format(query)
        r = sender.send_get(self.options, url, self.cookies, headers=custom_headers)

        if r.status_code == 200:
            html_data = r.text.replace('\/', '/').replace('\\"', '"').replace("\'", "'")
            soup = utils.soup(html_data)
            query_by_cities = []
            # custom here
            country_divs = soup.find_all("div", "class_sf")
            for div in country_divs:
                city_links = div.findChildren("a")
                for link in city_links:
                    query_by_cities.append({
                        'url': self.base_url + link.get('href'),
                        'city': link.text,
                    })
            
            # utils.print_debug(self.options, query_by_cities)
            return query_by_cities
        return False

    # just loop though the country code without analyze
    def brute_country_code(self, query):
        utils.print_info("Brute the query with country code")
        #  && country=US
        raw_query = utils.just_b64_decode(utils.url_decode(query))
        if 'country' in query:
            raw_query = query.replace(utils.get_country_code(
                raw_query, source='fofa'), '[replace]')
        else:
            raw_query += '&& country:"[replace]"'

        for country_code in utils.full_country_code:
            query = raw_query.replace('[replace]', country_code)
            query = utils.url_encode(utils.just_b64_encode(query))
            url = 'https://fofa.so/result?page={0}&qbase64={1}'.format(
                str(1), query)
                
            self.sending(url)

    def do_login(self):
        utils.print_info("Reauthen using credentials from: {0}".format(self.options.get('config')))

        login_url = 'https://i.nosec.org/login?service=http%3A%2F%2Ffofa.so%2Fusers%2Fservice'
        r = sender.send_get(self.options, login_url, cookies=None)

        if r.status_code == 200:
            cookies = r.cookies
            form = utils.soup(r.text).find(id="login-form")
            inputs = form.findChildren('input')

            for tag in inputs:
                if tag.get('name') == 'authenticity_token':
                    authenticity_token = tag.get('value')
                if tag.get('name') == 'lt':
                    lt = tag.get('value')
                if tag.get('name') == 'authenticity_token':
                    authenticity_token = tag.get('value')

            username, password = utils.get_cred(self.options, source='fofa')
        
            data = {"utf8": "\xe2\x9c\x93", "authenticity_token": authenticity_token,
                    "lt": lt, "service": "http://fofa.so/users/service", "username": username, "password": password, "rememberMe": "1", "button": ''}

            really_login_url = 'https://i.nosec.org/login'
            r1 = sender.send_post(
                self.options, really_login_url, cookies, data)

            if r1.status_code == 200:
                fofa_cookie = r1.cookies.get('_fofapro_ars_session')
                utils.set_session(self.options, fofa_cookie, source='fofa')
                return fofa_cookie
        return False

    # unique stuff
    def conclude(self):
        utils.just_cleanup(self.output)
