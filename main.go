package main

import (
	"flag"
	"fmt"
	"net"
	"os"
	"strings"

	"github.com/miekg/dns"
)

type SPFRecord struct {
	IP4      []string
	IP6      []string
	Includes []string
}

func main() {
	var (
		ip4List     stringSlice
		ip6List     stringSlice
		includeList stringSlice
		tags        bool
	)

	flag.Var(&ip4List, "ip4", "IPv4 addresses to include (can be specified multiple times)")
	flag.Var(&ip6List, "ip6", "IPv6 addresses to include (can be specified multiple times)")
	flag.Var(&includeList, "include", "Domain names to include SPF records from (can be specified multiple times)")
	flag.BoolVar(&tags, "tags", false, "Add ip4 or ip6 tag to each IP address")
	flag.Parse()

	if len(includeList) == 0 && len(ip4List) == 0 && len(ip6List) == 0 {
		fmt.Fprintln(os.Stderr, "Error: At least one -ip4, -ip6, or -include argument is required")
		flag.Usage()
		os.Exit(1)
	}

	ips, err := flattenSPF(ip4List, ip6List, includeList)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	for _, ip := range ips {
		if tags {
			tag := "ip6"
			if net.ParseIP(strings.Split(ip, "/")[0]).To4() != nil {
				tag = "ip4"
			}
			fmt.Printf("%s:%s\n", tag, ip)
		} else {
			fmt.Println(ip)
		}
	}
}

func flattenSPF(ip4List, ip6List, includeList []string) ([]string, error) {
	var allIPs []string

	allIPs = append(allIPs, ip4List...)
	allIPs = append(allIPs, ip6List...)

	visited := make(map[string]bool)
	for _, domain := range includeList {
		ips, err := resolveDomain(domain, visited)
		if err != nil {
			return nil, fmt.Errorf("failed to resolve include domain %s: %w", domain, err)
		}
		allIPs = append(allIPs, ips...)
	}

	uniqueIPs := deduplicateIPs(allIPs)
	return uniqueIPs, nil
}

func resolveDomain(domain string, visited map[string]bool) ([]string, error) {
	domain = strings.ToLower(domain)

	if visited[domain] {
		return nil, nil
	}
	visited[domain] = true

	spfRecord, err := getSPFRecord(domain)
	if err != nil {
		return nil, err
	}

	var ips []string
	ips = append(ips, spfRecord.IP4...)
	ips = append(ips, spfRecord.IP6...)

	for _, includeDomain := range spfRecord.Includes {
		includeIPs, err := resolveDomain(includeDomain, visited)
		if err != nil {
			return nil, fmt.Errorf("failed to resolve include %s: %w", includeDomain, err)
		}
		ips = append(ips, includeIPs...)
	}

	return ips, nil
}

func getSPFRecord(domain string) (*SPFRecord, error) {
	c := new(dns.Client)
	m := new(dns.Msg)

	m.SetQuestion(dns.Fqdn(domain), dns.TypeTXT)
	m.RecursionDesired = true
	m.SetEdns0(4096, false)

	r, _, err := c.Exchange(m, getDNSResolver())
	if err != nil {
		return nil, fmt.Errorf("DNS query failed: %w", err)
	}

	if r.Rcode != dns.RcodeSuccess {
		return nil, fmt.Errorf("DNS query returned error code: %s", dns.RcodeToString[r.Rcode])
	}

	var spfTxt string
	for _, ans := range r.Answer {
		if txt, ok := ans.(*dns.TXT); ok {
			for _, s := range txt.Txt {
				if strings.HasPrefix(strings.ToLower(s), "v=spf1") {
					spfTxt = strings.ToLower(s)
					break
				}
			}
		}
	}

	if spfTxt == "" {
		return nil, fmt.Errorf("no SPF record found for domain %s", domain)
	}

	return parseSPFRecord(spfTxt)
}

func parseSPFRecord(spf string) (*SPFRecord, error) {
	record := &SPFRecord{
		IP4:      []string{},
		IP6:      []string{},
		Includes: []string{},
	}

	parts := strings.Fields(spf)
	if len(parts) == 0 || !strings.HasPrefix(parts[0], "v=spf1") {
		return nil, fmt.Errorf("invalid SPF record: %s", spf)
	}

	for _, part := range parts[1:] {
		if strings.HasPrefix(part, "ip4:") {
			ip := strings.TrimPrefix(part, "ip4:")
			if isValidIP(ip, 4) {
				record.IP4 = append(record.IP4, ip)
			}
		} else if strings.HasPrefix(part, "ip6:") {
			ip := strings.TrimPrefix(part, "ip6:")
			if isValidIP(ip, 6) {
				record.IP6 = append(record.IP6, ip)
			}
		} else if strings.HasPrefix(part, "include:") {
			domain := strings.TrimPrefix(part, "include:")
			if domain != "" {
				record.Includes = append(record.Includes, domain)
			}
		}
	}

	return record, nil
}

func isValidIP(ip string, version int) bool {
	if strings.Contains(ip, "/") {
		ip = strings.Split(ip, "/")[0]
	}
	parsedIP := net.ParseIP(ip)
	if parsedIP == nil {
		return false
	}
	if version == 4 {
		return parsedIP.To4() != nil
	}
	return parsedIP.To4() == nil && strings.Contains(ip, ":")
}

func deduplicateIPs(ips []string) []string {
	seen := make(map[string]bool)
	var result []string

	for _, ip := range ips {
		if !seen[ip] {
			seen[ip] = true
			result = append(result, ip)
		}
	}

	return result
}

func getDNSResolver() string {
	if resolver := os.Getenv("DNS_RESOLVER"); resolver != "" {
		return resolver
	}
	return "127.0.0.1:53"
}

type stringSlice []string

func (s *stringSlice) String() string {
	return strings.Join(*s, ",")
}

func (s *stringSlice) Set(value string) error {
	*s = append(*s, value)
	return nil
}
