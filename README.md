<p align="center">
  <img alt="Metabigor" src="https://image.flaticon.com/icons/svg/2303/2303030.svg" height="140" />
  <p align="center">Intelligence Tool but without API key</p>
  <p align="center">
    <a href="https://github.com/j3ssie/metabigor"><img alt="Release" src="https://img.shields.io/badge/version-1.1-red.svg"></a>
    <a href=""><img alt="Software License" src="https://img.shields.io/badge/license-MIT-brightgreen.svg?style=flat-square"></a>
  </p>
</p>

## What is Metabigor?

Metabigor is Intelligence tool, its goal is to do OSINT tasks and more but without any API key.

## Installation

```
go get -u github.com/j3ssie/metabigor
```

### Example Commands

```
# discovery IP of a company/organization
echo "company" | metabigor net --org -o /tmp/result.txt

# discovery IP of an ASN
echo "ASN1111" | metabigor net --asn -o /tmp/result.txt
cat list_of_ASNs | metabigor net --asn -o /tmp/result.txt

# running masscan on port 443 for a subnet
echo "1.2.3.4/24" | metabigor scan -p 443 -o /tmp/result.txt

# running masscan on all port and nmap on open port
cat list_of_IPs | metabigor scan --detail -o /tmp/result.txt

# search result on fofa
echo 'title="RabbitMQ Management"' | metabigor search -x -v -o /tmp/result.txt
```

## Credits

Logo from [flaticon](https://www.flaticon.com/free-icon/metabolism_1774457) by [freepik
](https://www.flaticon.com/authors/freepik)

## Disclaimer

This tool is for educational purposes only. You are responsible for your own actions. If you mess something up or break any laws while using this software, it's your fault, and your fault only.

## License

`Metabigor` is made with ♥  by [@j3ssiejjj](https://twitter.com/j3ssiejjj) and it is released under the MIT license.
