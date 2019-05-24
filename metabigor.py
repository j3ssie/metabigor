#!/usr/bin/env python3
# -*- coding: utf-8 -*-

import os, sys
import argparse
from pprint import pprint
from core import config
from core import utils

from modules import fofa
from modules import shodan
from modules import censys

__author__ = '@j3ssiejjj'
__version__ = 'v1.0'

options = {}

def parsing_argument(args):
    global options

    if args.config:
        options['config'] = args.config

    if args.query:
        options['query'] = args.query

    if args.query_list:
        options['query_list'] = args.query_list

    options = config.config(options, args)

    if options.get('query_list'):
        queris = utils.just_read(options.get('query_list')).splitlines()

        for query in queris:
            options['query'] = query
            single_query(options)
    else:
        single_query(options)


def single_query(options):
    utils.print_debug(options, options)
    utils.print_info("Query: {0}".format(options.get('query')))
    if not options.get('source'):
        utils.print_bad("You need to specify Search engine")
        return

    if 'fofa' in options.get('source'):
        fofa.Fofa(options)

    if 'shodan' in options.get('source'):
        shodan.Shodan(options)

    if 'censys' in options.get('source'):
        censys.Censys(options)


def main():
    config.banner(__author__, __version__)
    parser = argparse.ArgumentParser(
        description="Command line Search Engines without any API key")

    parser.add_argument('-c', '--config', action='store', dest='config',
                        help='config file', default='config.conf')

    parser.add_argument('--cookies', action='store',
                        dest='cookies', help='content of cookies cookie')

    parser.add_argument('-s', '--source', action='store',
                        dest='source', help='name of search engine (e.g: shodan, censys, fofa)')

    parser.add_argument('-S', '--source_list', action='store',
                        dest='source_list', help='List of name of search engine (e.g: shodan, censys, fofa)')

    parser.add_argument('-q', '--query', action='store',
                        dest='query', help='Query from search engine')

    parser.add_argument('-Q', '--query_list', action='store',
                        dest='query_list', help='List of query from search engine')

    parser.add_argument('-d', '--outdir', action='store',
                        dest='outdir', help='Directory output', default='.')

    parser.add_argument('-o', '--output', action='store',
                        dest='output', help='Output file name', default='output')

    parser.add_argument('--raw', action='store',
                        dest='raw', help='Directory to store raw query', default='raw')

    parser.add_argument('--proxy', action='store',
                        dest='proxy', help='Proxy for doing request to search engine e.g: http://127.0.0.1:8080 ')

    parser.add_argument('-b', action='store_true', dest='brute', help='Auto brute force the country code')

    parser.add_argument('--disable_pages', action='store_true', dest='disable_pages', help="Don't loop though the pages")

    parser.add_argument('--store_content', action='store_true',
                        dest='store_content', help="Store the raw HTML souce or not")

    parser.add_argument('-hh', action='store_true', dest='helps', help='Print more help')

    parser.add_argument('--debug', action='store_true', dest='debug', help='Print debug output')
    parser.add_argument('--update', action='store_true',
                        dest='update', help='Update lastest version from git')

    args = parser.parse_args()

    if len(sys.argv) == 1 or args.helps:
        config.custom_help()
    if args.update:
        config.update()

    parsing_argument(args)


if __name__ == '__main__':
    main()
