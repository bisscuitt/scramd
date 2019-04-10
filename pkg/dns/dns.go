// Package dns handles DNS lookups for the Scramr mail server
package dns

import (
	"errors"
	"log"
	"net"
	"strings"
)

// func LookupRevRecord returns the reverse DNS record for an IP address
func LookupRevRecord(h string) (string, error) {
	addrs, err := net.LookupAddr(h)
	if err != nil {
		return "", err
	}

	for len(addrs) == 0 {
		return "", errors.New("No Reverse DNS found")
	}

	return addrs[0], nil
}

// func LookupMXHosts returns the MX records for a domain
func LookupMXHosts(d string) ([]string, error) {
	var addrs []string

	recs, err := net.LookupMX(d)
	if err != nil {
		// Continue to A Record lookup if we do not find any MX records
		if strings.HasSuffix(err.Error(), ": no such host") == false {
			log.Printf("Could not lookup MX records for %v: %v", d, err)
			return nil, err
		}
	} else {
		for k, v := range recs {
			log.Printf("MX Record Lookup returned: %v: %v - %v", k, v.Host, v.Pref)
			addrs = append(addrs, v.Host)
		}
	}

	// Fallback to A-Records if no MX records are found (RFC-5321)
	// https://en.wikipedia.org/wiki/MX_record#Fallback_to_the_address_record
	if addrs == nil {
		log.Printf("No MX Records Found for %v. Looking up A Records", d)
		arecs, err := net.LookupHost(d)

		// Lookup A Records
		if err != nil {
			log.Printf("Could not lookup A record for %v: %v", d, err)
			return nil, err
		}

		for k, v := range arecs {
			log.Printf("A Record Lookup returned: %v: %v", k, v)
			addrs = append(addrs, v)
		}
	}

	return addrs, nil
}
