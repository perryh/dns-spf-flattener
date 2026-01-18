# dns-spf-flattener

A Go CLI tool to flatten DNS SPF records and output a list of IP addresses.

## Installation

```bash
go install github.com/perryh/dns-spf-flattener@latest
```

Or build from source:

```bash
go build -o dns-spf-flattener .
```

## Usage

```
dns-spf-flattener [options]
```

### Options

- `-ip4 value` - IPv4 addresses to include (can be specified multiple times)
- `-ip6 value` - IPv6 addresses to include (can be specified multiple times)
- `-include value` - Domain names to include SPF records from (can be specified multiple times)
- `-tags` - List IP addresses with `ip4` and `ip6` tags

### Examples

Flatten SPF records from include domains:

```bash
dns-spf-flattener -include gmail.com -include example.com
```

Combine manual IPs with include domains:

```bash
dns-spf-flattener -ip4 192.0.2.1 -ip4 192.0.2.2 -include example.com
```

Use IPv6 addresses:

```bash
dns-spf-flattener -ip6 2001:db8::1 -include example.com
```

Full example:

```bash
$ DNS_RESOLVER=1.1.1.1:53 ./dns-spf-flattener -ip4 1.2.3.4 -ip4 1.2.3.5 -ip6 2001:db8:3333:4444:5555:6666:7777:8888 -ip6 2001:db8:3333:4444:CCCC:DDDD:EEEE:FFFF -include google.com
1.2.3.4
1.2.3.5
2001:db8:3333:4444:5555:6666:7777:8888
2001:db8:3333:4444:CCCC:DDDD:EEEE:FFFF
74.125.0.0/16
209.85.128.0/17
2001:4860:4000::/36
2404:6800:4000::/36
2607:f8b0:4000::/36
2800:3f0:4000::/36
2a00:1450:4000::/36
2c0f:fb50:4000::/36
```

## How It Works

1. Resolves the SPF record (TXT record starting with `v=spf1`) for each include domain
2. Extracts `ip4:` and `ip6:` entries from the SPF record
3. Recursively resolves nested `include:` entries
4. Combines all discovered IPs with the manually provided `-ip4` and `-ip6` addresses
5. Deduplicates and outputs the final list of IP addresses

## Environment Variables

- `DNS_RESOLVER` - Custom DNS resolver address (default: `127.0.0.1:53`)

Example:
```bash
DNS_RESOLVER=8.8.8.8:53 dns-spf-flattener -include example.com
```
