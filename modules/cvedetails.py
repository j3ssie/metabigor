import json
from core import sender
from core import utils


class Cvedetails():
    """docstring for Cvedetails"""

    def __init__(self, options):
        self.options = options
        self.query = self.options['product'].strip()
        self.baseURL = "https://www.cvedetails.com"
        self.output = self.options['outdir'] + \
            "/{0}-cvedetails.csv".format(self.options['output'])
        utils.print_banner("Starting scraping from Cvedetails resources")
        utils.print_info("Query for: " + self.query)
        self.initial()

    # really sending first request
    def initial(self):
        product = utils.url_encode(self.query)
        url = 'https://www.cvedetails.com/product-search.php?vendor_id=0&search={0}'.format(
            product)

        # get summary table
        products = []
        r = sender.send_get(self.options, url, cookies=None)
        if r.status_code == 200:
            response = r.text
            if 'class="errormsg"' in response:
                utils.print_bad("No entry found for: {0}".format(self.query))
                return
            
            summary_table = utils.soup(response).find_all("table", "listtable")
            # <table class = "listtable"
            if summary_table:
                trs = summary_table[0].findChildren('tr')
                if len(trs) <= 1:
                    utils.print_bad(
                        "No entry found for: {0}".format(self.query))
                    return
                
                for tr in trs[1:]:
                    for td in tr.findChildren('td'):
                        if td.a:
                            if 'See all vulnerabilities' in td.a.get('title'):
                                products.append(td.a.get('href'))

        final = []
        # if found product and have vulnerabilities, go get it
        if products:
            for url in products:
                results = self.sending(self.baseURL + url)
                if results:
                    final.extend(results)
            # self.details(products)
        # print(final)
        # write final output
        self.conclude(final)

    # just sending stuff
    def sending(self, url):
        r = sender.send_get(self.options, url, cookies=None)
        results = []
        if r:
            response = r.text
            if 'class="errormsg"' in response:
                utils.print_bad("No entry found for: {0}".format(self.query))
                return

            soup = utils.soup(response)
            result = self.analyze(soup)
            if not result:
                return False

            # checking if we have more than one pages
            pages = self.check_pages(soup)
            if pages:
                utils.print_info("Detect pages {0} for query".format(str(len(pages))))
                for page in pages:
                    page_url = self.baseURL + page
                    r1 = sender.send_get(self.options, page_url, cookies=None)
                    if r1:
                        response1 = r1.text
                        soup1 = utils.soup(response1)
                        self.analyze(soup1)

            results.extend(result)
            return results

    # analyze resonse
    def analyze(self, soup):
        utils.print_debug(self.options, "Analyze response")

        results = []
        vuln_table = soup.find(id="vulnslisttable")
        if vuln_table:
            rows = vuln_table.find_all('tr', 'srrowns')
            full_rows = vuln_table.find_all('td', 'cvesummarylong')

            for i in range(len(rows)):
                item = {
                    'Query': "N/A",
                    'CVE': "N/A",
                    'CVE URL': "N/A",
                    'Type': "N/A",
                    'Score': "N/A",
                    'Condition': "N/A",
                    'Descriptions': "N/A",
                }
                row = rows[i]
                row_detail = row.findChildren('td')
                cve = row_detail[1].a.text
                cve_url = self.baseURL + row_detail[1].a.get('href')
                vuln_type = row_detail[4].text.strip()
                score = row_detail[7].div.text
                condition = row_detail[9].text

                desc = full_rows[i].text.strip()

                item = {
                    'Query': self.query,
                    'CVE': cve,
                    'CVE URL': cve_url,
                    'Type': vuln_type,
                    'Score': score,
                    'Condition': condition,
                    'Descriptions': desc,
                }
                results.append(item)
        
        if results:
            return results
        else:
            return False

    # Check if we have more than one page
    def check_pages(self, soup):
        utils.print_debug(self.options, "Checking for more pages")
        div_pages = soup.find_all('div', 'paging')
        if div_pages:
            pages = []
            links = div_pages[1].find_all('a')
            # print(links)
            for link in links:
                if '(This Page)' not in link.text:
                    pages.append(link.get('href'))
            return pages
        else:
            return False


    # writing to csv file
    def conclude(self, output):
        head = ','.join([str(x).title() for x in output[0].keys()]) + "\n"
        body = ''
        for item in output:
            clean_body = [str(x).replace(',', '%2C').replace(
                "\n", "%0a%0d") for x in item.values()]
            body += ','.join(clean_body) + "\n"

        utils.check_output(self.output)
        utils.just_write(self.output, head + body)
