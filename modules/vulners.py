import json
from core import sender
from core import utils


class Vulners():
    """docstring for Vulners"""

    def __init__(self, options):
        self.options = options
        self.base_url = 'https://vulners.com/api/v3/burp/software/'

        self.query = (self.options['product'] + " " + self.options['version']).strip()
        self.output = self.options['outdir'] + \
            "/{0}-vulners.csv".format(self.options['output'])
        utils.print_banner("Starting scraping from Vulners")
        utils.print_info("Query for: " + self.query)
        self.initial()

    # really sending first request
    def initial(self):
        if self.options['version'] == '':
            utils.print_bad("Vulners module need to provided version")
            return 

        data = {"software": "cpe:/a:{0}:{0}".format(self.options['product'].lower()),
                "version": self.options['version'].lower(), "type": "cpe"}
        self.sending(data)

    # really sending request and looping through the page
    def sending(self, data):
        # sending request and return the response
        r = sender.send_post(self.options, self.base_url, data, is_json=True)
        if r.status_code == 200:
            response = json.loads(r.text)
            if response.get('result') == "OK":
                output = self.analyze(response)
                # write csv here
                self.conclude(output)

    # parse the html and get the result
    def analyze(self, response):
        warns = response.get('data').get('search')
        total = response.get('total')

        if total == 0:
            utils.print_info(
                "No exploit found for {0}".format(self.query))
            return False

        # store raw json
        raw_file_path = self.options['raw'] + '/vulners_{0}.json'.format(
            self.query.replace(' ', '_'))
        if self.options.get('store_content'):
            utils.just_write(raw_file_path, response, is_json=True)
            utils.print_debug(
                self.options, "Writing raw response to: {0}".format(raw_file_path))

        results = []
        for warn in warns:
            item = {
                'Query': self.query,
                'Title': warn.get('_source').get('title'),
                'Score': warn.get('_source').get('cvss').get('score'),
                'External_url': warn.get('_source').get('href'),
                'CVE': warn.get('_source').get('id'),
                'ID': warn.get('_id'),
                'Published': warn.get('_source').get('published'),
                'Source': "https://vulners.com/cve/" + warn.get('_id'),
                'Warning': 'Info',
                'Raw': raw_file_path,
            }
            utils.print_debug(self.options, item)
            results.append(item)

        return results

    # writing to csv file
    def conclude(self, output):
        head = ','.join([str(x).title() for x in output[0].keys()]) + "\n"
        body = ''
        for item in output:
            clean_body = [str(x).replace(',', '%2C') for x in item.values()]
            body += ','.join(clean_body) + "\n"

        utils.check_output(self.output)
        utils.just_write(self.output, head + body)
