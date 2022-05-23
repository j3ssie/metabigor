package modules

import (
    "encoding/xml"
    "fmt"
    "github.com/PuerkitoBio/goquery"
    "github.com/j3ssie/metabigor/core"
    "regexp"
    "strings"
)

// NmapRuns nmap multiple scan XML to struct
type NmapRuns struct {
    XMLName          xml.Name `xml:"nmaprun"`
    Text             string   `xml:",chardata"`
    Scanner          string   `xml:"scanner,attr"`
    Args             string   `xml:"args,attr"`
    Start            string   `xml:"start,attr"`
    Startstr         string   `xml:"startstr,attr"`
    Version          string   `xml:"version,attr"`
    Xmloutputversion string   `xml:"xmloutputversion,attr"`
    Scaninfo         struct {
        Text        string `xml:",chardata"`
        Type        string `xml:"type,attr"`
        Protocol    string `xml:"protocol,attr"`
        Numservices string `xml:"numservices,attr"`
        Services    string `xml:"services,attr"`
    } `xml:"scaninfo"`
    Verbose struct {
        Text  string `xml:",chardata"`
        Level string `xml:"level,attr"`
    } `xml:"verbose"`
    Debugging struct {
        Text  string `xml:",chardata"`
        Level string `xml:"level,attr"`
    } `xml:"debugging"`
    Taskbegin []struct {
        Text string `xml:",chardata"`
        Task string `xml:"task,attr"`
        Time string `xml:"time,attr"`
    } `xml:"taskbegin"`
    Taskend []struct {
        Text      string `xml:",chardata"`
        Task      string `xml:"task,attr"`
        Time      string `xml:"time,attr"`
        Extrainfo string `xml:"extrainfo,attr"`
    } `xml:"taskend"`
    Host []struct {
        Text      string `xml:",chardata"`
        Starttime string `xml:"starttime,attr"`
        Endtime   string `xml:"endtime,attr"`
        Status    struct {
            Text      string `xml:",chardata"`
            State     string `xml:"state,attr"`
            Reason    string `xml:"reason,attr"`
            ReasonTtl string `xml:"reason_ttl,attr"`
        } `xml:"status"`
        Address struct {
            Text     string `xml:",chardata"`
            Addr     string `xml:"addr,attr"`
            Addrtype string `xml:"addrtype,attr"`
        } `xml:"address"`
        Hostnames struct {
            Text     string `xml:",chardata"`
            Hostname struct {
                Text string `xml:",chardata"`
                Name string `xml:"name,attr"`
                Type string `xml:"type,attr"`
            } `xml:"hostname"`
        } `xml:"hostnames"`
        Ports struct {
            Text       string `xml:",chardata"`
            Extraports struct {
                Text         string `xml:",chardata"`
                State        string `xml:"state,attr"`
                Count        string `xml:"count,attr"`
                Extrareasons []struct {
                    Text   string `xml:",chardata"`
                    Reason string `xml:"reason,attr"`
                    Count  string `xml:"count,attr"`
                } `xml:"extrareasons"`
            } `xml:"extraports"`
            Port []struct {
                Text     string `xml:",chardata"`
                Protocol string `xml:"protocol,attr"`
                Portid   string `xml:"portid,attr"`
                State    struct {
                    Text      string `xml:",chardata"`
                    State     string `xml:"state,attr"`
                    Reason    string `xml:"reason,attr"`
                    ReasonTtl string `xml:"reason_ttl,attr"`
                } `xml:"state"`
                Service struct {
                    Text       string `xml:",chardata"`
                    Name       string `xml:"name,attr"`
                    Tunnel     string `xml:"tunnel,attr"`
                    Method     string `xml:"method,attr"`
                    Conf       string `xml:"conf,attr"`
                    Product    string `xml:"product,attr"`
                    Devicetype string `xml:"devicetype,attr"`
                    Servicefp  string `xml:"servicefp,attr"`
                    Cpe        string `xml:"cpe"`
                } `xml:"service"`
                Script struct {
                    Text   string `xml:",chardata"`
                    ID     string `xml:"id,attr"`
                    Output string `xml:"output,attr"`
                    Elem   []struct {
                        Text string `xml:",chardata"`
                        Key  string `xml:"key,attr"`
                    } `xml:"elem"`
                } `xml:"script"`
            } `xml:"port"`
        } `xml:"ports"`
        Times struct {
            Text   string `xml:",chardata"`
            Srtt   string `xml:"srtt,attr"`
            Rttvar string `xml:"rttvar,attr"`
            To     string `xml:"to,attr"`
        } `xml:"times"`
    } `xml:"host"`
    Taskprogress []struct {
        Text      string `xml:",chardata"`
        Task      string `xml:"task,attr"`
        Time      string `xml:"time,attr"`
        Percent   string `xml:"percent,attr"`
        Remaining string `xml:"remaining,attr"`
        Etc       string `xml:"etc,attr"`
    } `xml:"taskprogress"`
    Runstats struct {
        Text     string `xml:",chardata"`
        Finished struct {
            Text    string `xml:",chardata"`
            Time    string `xml:"time,attr"`
            Timestr string `xml:"timestr,attr"`
            Elapsed string `xml:"elapsed,attr"`
            Summary string `xml:"summary,attr"`
            Exit    string `xml:"exit,attr"`
        } `xml:"finished"`
        Hosts struct {
            Text  string `xml:",chardata"`
            Up    string `xml:"up,attr"`
            Down  string `xml:"down,attr"`
            Total string `xml:"total,attr"`
        } `xml:"hosts"`
    } `xml:"runstats"`
}

// NmapRun nmap single scan XML to struct
type NmapRun struct {
    XMLName          xml.Name `xml:"nmaprun"`
    Text             string   `xml:",chardata"`
    Scanner          string   `xml:"scanner,attr"`
    Args             string   `xml:"args,attr"`
    Start            string   `xml:"start,attr"`
    Startstr         string   `xml:"startstr,attr"`
    Version          string   `xml:"version,attr"`
    Xmloutputversion string   `xml:"xmloutputversion,attr"`
    Scaninfo         struct {
        Text        string `xml:",chardata"`
        Type        string `xml:"type,attr"`
        Protocol    string `xml:"protocol,attr"`
        Numservices string `xml:"numservices,attr"`
        Services    string `xml:"services,attr"`
    } `xml:"scaninfo"`
    Verbose struct {
        Text  string `xml:",chardata"`
        Level string `xml:"level,attr"`
    } `xml:"verbose"`
    Debugging struct {
        Text  string `xml:",chardata"`
        Level string `xml:"level,attr"`
    } `xml:"debugging"`
    Taskbegin []struct {
        Text string `xml:",chardata"`
        Task string `xml:"task,attr"`
        Time string `xml:"time,attr"`
    } `xml:"taskbegin"`
    Taskend []struct {
        Text      string `xml:",chardata"`
        Task      string `xml:"task,attr"`
        Time      string `xml:"time,attr"`
        Extrainfo string `xml:"extrainfo,attr"`
    } `xml:"taskend"`
    Taskprogress []struct {
        Text      string `xml:",chardata"`
        Task      string `xml:"task,attr"`
        Time      string `xml:"time,attr"`
        Percent   string `xml:"percent,attr"`
        Remaining string `xml:"remaining,attr"`
        Etc       string `xml:"etc,attr"`
    } `xml:"taskprogress"`

    Host struct {
        Text      string `xml:",chardata"`
        Starttime string `xml:"starttime,attr"`
        Endtime   string `xml:"endtime,attr"`
        Status    struct {
            Text      string `xml:",chardata"`
            State     string `xml:"state,attr"`
            Reason    string `xml:"reason,attr"`
            ReasonTtl string `xml:"reason_ttl,attr"`
        } `xml:"status"`
        Address struct {
            Text     string `xml:",chardata"`
            Addr     string `xml:"addr,attr"`
            Addrtype string `xml:"addrtype,attr"`
        } `xml:"address"`
        Hostnames struct {
            Text     string `xml:",chardata"`
            Hostname struct {
                Text string `xml:",chardata"`
                Name string `xml:"name,attr"`
                Type string `xml:"type,attr"`
            } `xml:"hostname"`
        } `xml:"hostnames"`
        Ports struct {
            Text       string `xml:",chardata"`
            Extraports struct {
                Text         string `xml:",chardata"`
                State        string `xml:"state,attr"`
                Count        string `xml:"count,attr"`
                Extrareasons []struct {
                    Text   string `xml:",chardata"`
                    Reason string `xml:"reason,attr"`
                    Count  string `xml:"count,attr"`
                } `xml:"extrareasons"`
            } `xml:"extraports"`
            Port []struct {
                Text     string `xml:",chardata"`
                Protocol string `xml:"protocol,attr"`
                Portid   string `xml:"portid,attr"`
                State    struct {
                    Text      string `xml:",chardata"`
                    State     string `xml:"state,attr"`
                    Reason    string `xml:"reason,attr"`
                    ReasonTtl string `xml:"reason_ttl,attr"`
                } `xml:"state"`
                Service struct {
                    Text       string   `xml:",chardata"`
                    Name       string   `xml:"name,attr"`
                    Product    string   `xml:"product,attr"`
                    Devicetype string   `xml:"devicetype,attr"`
                    Method     string   `xml:"method,attr"`
                    Conf       string   `xml:"conf,attr"`
                    Version    string   `xml:"version,attr"`
                    Extrainfo  string   `xml:"extrainfo,attr"`
                    Ostype     string   `xml:"ostype,attr"`
                    Servicefp  string   `xml:"servicefp,attr"`
                    Cpe        []string `xml:"cpe"`
                } `xml:"service"`
                Script struct {
                    Text   string `xml:",chardata"`
                    ID     string `xml:"id,attr"`
                    Output string `xml:"output,attr"`
                    Elem   []struct {
                        Text string `xml:",chardata"`
                        Key  string `xml:"key,attr"`
                    } `xml:"elem"`
                } `xml:"script"`
            } `xml:"port"`
        } `xml:"ports"`
        Times struct {
            Text   string `xml:",chardata"`
            Srtt   string `xml:"srtt,attr"`
            Rttvar string `xml:"rttvar,attr"`
            To     string `xml:"to,attr"`
        } `xml:"times"`
    } `xml:"host"`
    Runstats struct {
        Text     string `xml:",chardata"`
        Finished struct {
            Text    string `xml:",chardata"`
            Time    string `xml:"time,attr"`
            Timestr string `xml:"timestr,attr"`
            Elapsed string `xml:"elapsed,attr"`
            Summary string `xml:"summary,attr"`
            Exit    string `xml:"exit,attr"`
        } `xml:"finished"`
        Hosts struct {
            Text  string `xml:",chardata"`
            Up    string `xml:"up,attr"`
            Down  string `xml:"down,attr"`
            Total string `xml:"total,attr"`
        } `xml:"hosts"`
    } `xml:"runstats"`
}

type Host struct {
    IPAddress string
    Hostname  string
    Ports     []Port
}

type Port struct {
    Protocol string
    PortID   string
    State    string
    Service  struct {
        Name    string
        Product string
        Cpe     string
    }
    Script struct {
        ID     string
        Output string
    }
}

// ParseNmapXML parse nmap XML result
func ParseNmapXML(raw string) NmapRun {
    // parsing content
    nmapRun := NmapRun{}
    err := xml.Unmarshal([]byte(raw), &nmapRun)
    if err != nil {
        core.ErrorF("Failed to parse Nmap XML file: %v", err)
        return nmapRun
    }
    return nmapRun
}

// ParseMultipleNmapXML parse nmap XML result
func ParseMultipleNmapXML(raw string) NmapRuns {
    nmapRuns := NmapRuns{}
    err := xml.Unmarshal([]byte(raw), &nmapRuns)
    if err != nil {
        core.ErrorF("Failed to parse Nmap XML file: %v", err)
        return nmapRuns
    }
    return nmapRuns
}

// GetHosts parse nmap XML and return  mutilehost object
func GetHosts(raw string) []Host {
    var hosts []Host
    if strings.Count(raw, "<address") <= 1 {
        return hosts
    }
    nmapObj := ParseMultipleNmapXML(raw)
    if nmapObj.Args == "" || len(nmapObj.Host) <= 0 {
        core.ErrorF("Failed to parse Nmap XML")
        return hosts
    }
    // really parse something here
    for _, nmapHost := range nmapObj.Host {
        var host Host
        core.DebugF("Parse XML for: %v", host.IPAddress)

        host.IPAddress = nmapHost.Address.Addr
        host.Hostname = nmapHost.Hostnames.Hostname.Name
        if len(nmapHost.Ports.Port) > 0 {
            for _, port := range nmapHost.Ports.Port {
                var item Port
                item.PortID = port.Portid
                item.Protocol = port.Protocol
                item.State = port.State.State
                // service
                item.Service.Name = port.Service.Name
                item.Service.Product = port.Service.Product
                //item.Service.Cpe = port.Service.Cpe
                item.Script.ID = port.Script.ID
                item.Script.Output = port.Script.Output
                host.Ports = append(host.Ports, item)
            }
            hosts = append(hosts, host)
        }
    }
    return hosts
}

// GetHost parse nmap XML and return host object
func GetHost(raw string) Host {
    var host Host
    nmapObj := ParseNmapXML(raw)
    if nmapObj.Args == "" {
        core.ErrorF("Failed to parse Nmap XML")
        return host
    }
    host.IPAddress = nmapObj.Host.Address.Addr
    host.Hostname = nmapObj.Host.Hostnames.Hostname.Name
    core.DebugF("Parse XML for: %v", host.IPAddress)
    if len(nmapObj.Host.Ports.Port) > 0 {
        for _, port := range nmapObj.Host.Ports.Port {
            var item Port
            item.PortID = port.Portid
            item.Protocol = port.Protocol
            item.State = port.State.State
            // service
            item.Service.Name = port.Service.Name
            item.Service.Product = port.Service.Product
            //item.Service.Cpe = port.Service.Cpe
            item.Script.ID = port.Script.ID
            item.Script.Output = port.Script.Output

            host.Ports = append(host.Ports, item)
        }
    }

    return host
}

// ParsingNmapWithGoquery parse result from nmap XML format using goquery
func ParsingNmapWithGoquery(raw string, options core.Options) map[string][]string {
    result := make(map[string][]string)

    doc, err := goquery.NewDocumentFromReader(strings.NewReader(raw))
    if err != nil {
        return result
    }
    doc.Find("host").Each(func(i int, h *goquery.Selection) {
        ip, _ := h.Find("address").First().Attr("addr")

        h.Find("port").Each(func(j int, s *goquery.Selection) {
            service, _ := s.Find("service").First().Attr("name")
            product, ok := s.Find("service").First().Attr("product")
            if !ok {
                product = ""
            }
            port, _ := s.Attr("portid")
            info := fmt.Sprintf("%v/%v/%v", port, service, product)
            result[ip] = append(result[ip], strings.TrimSpace(info))
        })

        if options.Scan.NmapScripts != "" {
            h.Find("script").Each(func(j int, s *goquery.Selection) {
                id, _ := s.Attr("id")
                scriptOutput, _ := s.Attr("output")

                if scriptOutput != "" {
                    // grep script output with grepString
                    if options.Scan.GrepString != "" {
                        var vulnerable bool
                        if strings.Contains(scriptOutput, options.Scan.GrepString) {
                            vulnerable = true
                        } else {
                            r, err := regexp.Compile(options.Scan.GrepString)
                            if err == nil {
                                matches := r.FindStringSubmatch(scriptOutput)
                                if len(matches) > 0 {
                                    vulnerable = true
                                }
                            }
                        }
                        if vulnerable {
                            vul := fmt.Sprintf("/vulnerable|%v", id)
                            result[ip] = append(result[ip], strings.TrimSpace(vul))
                        }
                    }

                    scriptOutput = strings.Replace(scriptOutput, "\n", "\\n", -1)
                    info := fmt.Sprintf("/script|%v;;out|%v", id, scriptOutput)
                    result[ip] = append(result[ip], strings.TrimSpace(info))
                }
            })
        }
    })

    return result
}
