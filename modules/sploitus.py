import json
from core import sender
from core import utils


class Sploitus():
    """docstring for Sploitus"""
    def __init__(self, options):
        self.options = options
        self.base_url = 'https://sploitus.com/search'

        self.query = (self.options['product'] + " " + self.options['version']).strip()
        self.output = self.options['outdir'] + \
            "/{0}-sploitus.csv".format(self.options['output'])
        utils.print_banner("Starting scraping from Sploitus")
        utils.print_info("Query for: " + self.query)
        self.initial()

    # really sending first request
    def initial(self):
        # prepare data
        data = {"type": "exploits", "sort": "default",
                "query": self.query,
                "title": not self.options.get('relatively'), "offset": 0}
        self.sending(data)


    # really sending request and looping through the page
    def sending(self, data):
        # sending request and return the response
        r = sender.send_post(self.options, self.base_url, data, is_json=True)
        if r.status_code == 200:
            response = json.loads(r.text)
            output = self.analyze(response)

            if output:
                page_num = int(response.get('exploits_total')) / 10
                # checking if there is many pages or not
                if self.pages(page_num):
                    output += self.pages(page_num)
                # write csv here
                self.conclude(output)

    # parse the html and get the result
    def analyze(self, response):
        exploits = response.get('exploits')
        utils.print_debug(self.options, len(exploits))
        if len(exploits) == 0:
            utils.print_info(
                "No exploit found for {0}".format(self.query))
            return False

        # store raw json
        raw_file_path = self.options['raw'] + '/sploitus_{0}.json'.format(
            self.query.replace(' ', '_'))
        if self.options.get('store_content'):
            utils.just_write(raw_file_path, response, is_json=True)
            utils.print_debug(self.options, "Writing raw response to: {0}".format(raw_file_path))

        results = []
        for exploit in exploits:
            item = {
                'Query': self.query,
                'Title': exploit.get('title'),
                'Score': str(exploit.get('score')),
                'External_url': exploit.get('href'),
                'CVE': str(utils.get_cve(exploit.get('source'))),
                'ID': exploit.get('id'),
                'Published': exploit.get('published'),
                'Source': self.base_url + 'exploit?id=' + exploit.get('id'),
                'Warning': 'High',
                'Raw': raw_file_path,
            }
            utils.print_debug(self.options, item)
            results.append(item)

        return results

    # keep doing if there have many pages
    def pages(self, page_num):
        more_output = []
        for i in range(1, int(page_num) + 1):
            utils.print_debug(self.options, "Sleep for couple seconds")
            utils.random_sleep(1, 3)
            utils.print_info("Get more result from page: {0}".format(str(i)))

            data = {"type": "exploits", "sort": "default",
                    "query": self.query,
                    "title": not self.options.get('relatively'), "offset": i * 10}
            
            r = sender.send_post(
                self.options, self.base_url, data, is_json=True)
            if r.status_code == 200:
                response = json.loads(r.text)
                if self.analyze(response):
                    more_output += self.analyze(response)
                else:
                    return False

        return more_output

    # writing to csv file
    def conclude(self, output):
        head = ','.join([str(x) for x in output[0].keys()]) + "\n"
        body = ''
        for item in output:
            clean_body = [str(x).replace(',', '%2C') for x in item.values()]
            body += ','.join(clean_body) + "\n"
        
        utils.check_output(self.output)
        utils.just_write(self.output, head + body)

