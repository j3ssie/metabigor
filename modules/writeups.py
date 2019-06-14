import json
from core import sender
from core import utils


class Writeups():
    """docstring for Vulners"""

    def __init__(self, options):
        self.options = options
        self.query = self.options['product'].strip()
        self.output = self.options['outdir'] + \
            "/{0}-write-ups.csv".format(self.options['output'])
        utils.print_banner("Starting scraping from Write-ups resources")
        utils.print_info("Query for: " + self.query)
        self.initial()

    # really sending first request
    def initial(self):
        githubs = [
            'https://github.com/ngalongc/bug-bounty-reference',
            'https://github.com/pentesterland/pentesterland.github.io/blob/master/_pages/list-of-bug-bounty-writeups.md'
        ]

        output = []
        # github
        utils.print_info("Finding {0} write-up in Github resources".format(self.query))
        for github in githubs:
            output += self.github(github)

        # twitter   
        utils.print_info("Finding {0} sensitive tweets in Twitters resources".format(self.query))
        sensitive_tags = [
            '0day',
            'RCE',
            'bugbounty'
        ]
        for tags in sensitive_tags:
            output += self.tweet(tags)

        if output:
            self.conclude(output)


    def github(self, url):
        result = []
        r = sender.send_get(self.options, url, cookies=None)
        if r.status_code == 200:
            response = r.text
            # store raw json
            raw_file_path = self.options['raw'] + '/write_up_github_{0}.html'.format(
                self.query.replace(' ', '_'))
            if self.options.get('store_content'):
                utils.just_write(raw_file_path, response)
                utils.print_debug(self.options, "Writing raw response to: {0}".format(raw_file_path))

            soup = utils.soup(response)

            # Custom here
            body = soup.find_all('article', 'markdown-body')[0]
            links = body.findChildren('a')
            for link in links:
                if self.query.lower() in link.text.lower():
                    item = {
                        'Query': self.query,
                        'Title': link.text,
                        'Content': link.text,
                        'External_url': link.get('href'),
                        'Source': url,
                        'Warning': 'Write-Up',
                        'Raw': raw_file_path
                    }
                    utils.print_debug(self.options, item)
                    result.append(item)
                
        return result


    def tweet(self, tag):
        results = []
        query = '#{0} #{1}'.format(self.query, tag)
        # @TODO improve by increase the the position
        url = 'https://twitter.com/search?vertical=default&q={0}&src=unkn'.format(utils.url_encode(query))
        r = sender.send_get(self.options, url, cookies=None)
        if r.status_code == 200:
            response = r.text

            # store raw json
            raw_file_path = self.options['raw'] + '/tweets_{1}_{0}.html'.format(
                self.query.replace(' ', '_'), tag)
            if self.options.get('store_content'):
                utils.just_write(raw_file_path, response)
                utils.print_debug(self.options, "Writing raw response to: {0}".format(raw_file_path))
            soup = utils.soup(response)

            # Custom here
            divs = soup.find_all('div', 'original-tweet')
            for div in divs:
                content = div.findChildren('p', 'TweetTextSize')[0].text.strip()
                links = [x.get('data-expanded-url')
                         for x in div.findChildren('a') if 't.co' in x.get('href')]
                # print(links)
                if len(links) == 0:
                    external_url = 'N/A'
                else:
                    external_url = '|'.join([str(x) for x in links])

                item = {
                    'Query': self.query,
                    'Title': query,
                    'Content': content,
                    'External_url': external_url,
                    'Source': url,
                    'Warning': 'Tweet',
                    'Raw': raw_file_path
                }
                utils.print_debug(self.options, item)
                results.append(item)
            
        return results

    # writing to csv file
    def conclude(self, output):
        head = ','.join([str(x).title() for x in output[0].keys()]) + "\n"
        body = ''
        for item in output:
            clean_body = [str(x).replace(',', '%2C').replace("\n", "%0a%0d") for x in item.values()]
            body += ','.join(clean_body) + "\n"

        utils.check_output(self.output)
        utils.just_write(self.output, head + body)
