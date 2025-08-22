package main

import "github.com/libdns/libdns"

type DnsClient interface {
	libdns.RecordGetter
	libdns.RecordSetter
}
