<p align="center">
  <img alt="Metabigor" src="https://image.flaticon.com/icons/svg/1789/1789851.svg" height="140" />
  <p align="center">Intelligence Tool but without API key</p>
  <p align="center">
    <a href="https://github.com/j3ssie/metabigor"><img alt="Release" src="https://img.shields.io/github/v/release/j3ssie/metabigor.svg"></a>
    <a href=""><img alt="Software License" src="https://img.shields.io/badge/license-MIT-brightgreen.svg?style=flat-square"></a>
  </p>
</p>

## What is Metabigor?

Metabigor is Intelligence tool, its goal is to do OSINT tasks and more but without any API key.

## Installation

```
GO111MODULE=on go get github.com/j3ssie/metabigor
```

## Main features

- Searching information about IP Address, ASN and Organization.
- Wrapper for running masscan and nmap on IP target.
- Do searching from command line on some search engine.

## Example Commands

```
Examples Commands:
# discovery IP of a company/organization
echo "company" | metabigor net --org -o /tmp/result.txt

# discovery IP of an ASN
echo "ASN1111" | metabigor net --asn -o /tmp/result.txt
cat list_of_ASNs | metabigor net --asn -o /tmp/result.txt

# Only run masscan full ports
echo '1.2.3.4/24' | metabigor scan -o result.txt

# Only run nmap detail scan based on pre-scan data
echo '1.2.3.4:21' | metabigor scan -s -c 10
echo '1.2.3.4:21' | metabigor scan --tmp /tmp/raw-result/ -s -o result.txt
echo '1.2.3.4 -> [80,443,2222]' | metabigor scan -R

# Only run scan with zmap
cat ranges.txt | metabigor scan -p '443,80' -z

# search result on fofa
echo 'title="RabbitMQ Management"' | metabigor search -x -v -o /tmp/result.txt

# certificate search info on crt.sh
echo 'Target' | metabigor cert

# Get Summary about IP address (powered by @thebl4ckturtle)
cat list_of_ips.txt | metabigor ipc --json
```

## Demo

[![asciicast](https://asciinema.org/a/301745.svg)](https://asciinema.org/a/301745)

## Credits

Logo from [flaticon](https://image.flaticon.com/icons/svg/1789/1789851.svg)
by [freepik](https://www.flaticon.com/authors/freepik)

## Disclaimer

This tool is for educational purposes only. You are responsible for your own actions. If you mess something up or break
any laws while using this software, it's your fault, and your fault only.

## License

`Metabigor` is made with ♥ by [@j3ssiejjj](https://twitter.com/j3ssiejjj) and it is released under the MIT license.

## Donation

[![paypal](https://www.paypalobjects.com/en_US/i/btn/btn_donateCC_LG.gif)](https://paypal.me/j3ssiejjj)
