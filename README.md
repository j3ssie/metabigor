<p align="center">
  <img alt="Metabigor" src="https://user-images.githubusercontent.com/23289085/143042137-28f8e7e5-e485-4dc8-a09b-10759a593210.png" height="140" />
  <br />
  <strong>Metabigor - An Intelligence Tool but without API key</strong>

  <p align="center">
  <a href="https://docs.osmedeus.org/donation/"><img src="https://img.shields.io/badge/Sponsors-0078D4?style=for-the-badge&logo=GitHub-Sponsors&logoColor=39ff14&labelColor=black&color=black"></a>
  <a href="https://twitter.com/OsmedeusEngine"><img src="https://img.shields.io/badge/%40OsmedeusEngine-0078D4?style=for-the-badge&logo=Twitter&logoColor=39ff14&labelColor=black&color=black"></a>
  <a href="https://github.com/j3ssie/osmedeus/releases"><img src="https://img.shields.io/github/release/j3ssie/metabigor?style=for-the-badge&labelColor=black&color=2fc414&logo=Github"></a>
  </p>
</p>

***


# What is Metabigor?

Metabigor is Intelligence tool, its goal is to do OSINT tasks and more but without any API key.

# Installation

```shell
go install github.com/j3ssie/metabigor@latest
```

# Main features

- Searching information about IP Address, ASN and Organization.
- Wrapper for running rustscan, masscan and nmap more efficient on IP/CIDR.
- Finding more related domains of the target by applying various techniques (certificate, whois, Google Analytics, etc).
- Get Summary about IP address (powered by [**@thebl4ckturtle**](https://github.com/theblackturtle))


# Usage

## Discovery IP of a company/organization - `metabigor net`

The difference between net and **netd** command is that **netd** will get the dynamic result from the third-party source while net command will get the static result from the database.


```bash
# discovery IP of a company/organization
echo "company" | metabigor net --org -o /tmp/result.txt

# discovery IP of an ASN
echo "ASN1111" | metabigor net --asn -o /tmp/result.txt
cat list_of_ASNs | metabigor net --asn -o /tmp/result.txt

echo "ASN1111" | metabigor netd --asn -o /tmp/result.txt
```

*** 

## Finding more related domains of the target by applying various techniques (certificate, whois, Google Analytics, etc) - `metabigor related`

> Note some of the results are not 100% accurate. Please do a manual check first before put it directly to other tools to scan.

Some specific technique require different input so please see the usage of each technique.


## Using certificate to find related domains on crt.sh

```bash
# Getting more related domains by searching for certificate info
echo 'Target Inc' | metabigor cert --json | jq -r '.Domain' | unfurl format %r.%t | sort -u # this is old command

# Getting more related domains by searching for certificate info
echo 'example Inc' | metabigor related -s 'cert'
```

## Wrapper for running rustscan, masscan and nmap more efficient on IP/CIDR - `metabigor scan` 

This command will require you to install `masscan`, `rustscan` and `nmap` first or at least the pre-scan result of them.

```bash
# Only run masscan full ports
echo '1.2.3.4/24' | metabigor scan -o result.txt

# only run nmap detail scan based on pre-scan data
echo '1.2.3.4:21' | metabigor scan -s -c 10
echo '1.2.3.4:21' | metabigor scan --tmp /tmp/raw-result/ -s -o result.txt

# run nmap detail scan based on pre-scan data of rustscan
echo '1.2.3.4 -> [80,443,2222]' | metabigor scan -R

# only run scan with zmap
cat ranges.txt | metabigor scan -p '443,80' -z
```

***

## Using Reverse Whois to find related domains

```bash
echo 'example.com' | metabigor related -s 'whois'
```

## Getting more related by searching for Google Analytics ID

```bash
# Get it directly from the URL
echo 'https://example.com' | metabigor related -s 'google-analytic'

# You can also search it directly from the UA ID too
metabigor related -s 'google-analytic' -i 'UA-9152XXX' --debug
```

*** 

## Get Summary about IP address (powered by [**@thebl4ckturtle**](https://github.com/theblackturtle)) - `metabigor ipc`

This will show you the summary of the IP address provided like ASN, Organization, Country, etc.


```bash
cat list_of_ips.txt | metabigor ipc --json
```


## Extract Shodan IPInfo from internetdb.shodan.io

```bash
echo '1.2.3.4' | metabigor ip -open
1.2.3.4:80
1.2.3.4:443

# lookup CIDR range
echo '1.2.3.4/24' | metabigor ip -open -c 20
1.2.3.4:80
1.2.3.5:80

# get raw JSON response
echo '1.2.3.4' | metabigor ip -json
```


# Demo

[![asciicast](https://asciinema.org/a/301745.svg)](https://asciinema.org/a/301745)

*** 

# Painless integrate Jaeles into your recon workflow?

<p align="center">
  <img alt="OsmedeusEngine" src="https://raw.githubusercontent.com/osmedeus/assets/main/logo-transparent.png" height="200" />
  <p align="center">
    This project was part of Osmedeus Engine. Check out how it was integrated at <a href="https://twitter.com/OsmedeusEngine">@OsmedeusEngine</a>
  </p>
</p>

# Credits

Logo from [flaticon](https://image.flaticon.com/icons/svg/1789/1789851.svg)
by [freepik](https://www.flaticon.com/authors/freepik)

# Disclaimer

This tool is for educational purposes only. You are responsible for your own actions. If you mess something up or break
any laws while using this software, it's your fault, and your fault only.

# License

`Metabigor` is made with ♥ by [@j3ssiejjj](https://twitter.com/j3ssiejjj) and it is released under the MIT license.

# Donation

[![paypal](https://www.paypalobjects.com/en_US/i/btn/btn_donateCC_LG.gif)](https://paypal.me/j3ssiejjj)

[!["Buy Me A Coffee"](https://www.buymeacoffee.com/assets/img/custom_images/orange_img.png)](https://www.buymeacoffee.com/j3ssie)
