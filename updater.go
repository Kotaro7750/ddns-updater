package main

import (
	"context"
	"fmt"
	"log/slog"
	"net"

	"github.com/cloudflare/cloudflare-go/v5"
	"github.com/cloudflare/cloudflare-go/v5/dns"
	"github.com/cloudflare/cloudflare-go/v5/zones"
)

type RecordUpdater struct {
	ZoneName   string
	RecordName string
}

func (r *RecordUpdater) updateDNSRecord(addr net.IP) error {
	isAAAA := checkIfIPv6(addr)

	// 1. Create cloudflare api client
	client := cloudflare.NewClient()
	if client == nil {
		return fmt.Errorf("Failed to create Cloudflare client")
	}

	// 2. Get target Zone
	res, err := client.Zones.List(context.TODO(), zones.ZoneListParams{
		Name: cloudflare.F(r.ZoneName),
	})
	if err != nil {
		return fmt.Errorf("Failed to list zones: %w", err)
	}

	if len(res.Result) != 1 {
		return fmt.Errorf("Expected exactly one zone, found %d. Refine zone name %s", len(res.Result), r.ZoneName)
	}

	zoneId := res.Result[0].ID

	// 3. Check if the DNS record exists
	listType := dns.RecordListParamsTypeA
	if isAAAA {
		listType = dns.RecordListParamsTypeAAAA
	}

	record, err := client.DNS.Records.List(context.TODO(), dns.RecordListParams{
		ZoneID: cloudflare.F(zoneId),
		Name: cloudflare.F(dns.RecordListParamsName{
			Exact: cloudflare.F(fmt.Sprintf("%s.%s", r.RecordName, r.ZoneName)),
		}),
		Type: cloudflare.F(listType),
	})
	if err != nil {
		return fmt.Errorf("Failed to list DNS record %s: %w", r.RecordName, err)
	}

	// 4. Upsert the DNS record
	if len(record.Result) == 0 {
		slog.Info("DNS record not found, creating a new one", "name", r.RecordName, "type", listType, "content", addr.String())

		// 4.a DNS record not found, create a new one
		var recordParam dns.RecordNewParamsBodyUnion
		if isAAAA {
			recordParam = dns.AAAARecordParam{
				Name:    cloudflare.F(r.RecordName),
				Content: cloudflare.F(addr.String()),
				Type:    cloudflare.F(dns.AAAARecordTypeAAAA),
			}
		} else {
			recordParam = dns.ARecordParam{
				Name:    cloudflare.F(r.RecordName),
				Content: cloudflare.F(addr.String()),
				Type:    cloudflare.F(dns.ARecordTypeA),
			}
		}

		_, err := client.DNS.Records.New(context.TODO(), dns.RecordNewParams{
			ZoneID: cloudflare.F(zoneId),
			Body:   recordParam,
		})
		if err != nil {
			return fmt.Errorf("Failed to create DNS record %s: %w", r.RecordName, err)
		}

	} else if len(record.Result) == 1 {
		// 4.b DNS record found, update it
		slog.Info("DNS record found, updating it", "name", r.RecordName, "type", listType, "content", addr.String())

		var recordParam dns.RecordUpdateParamsBodyUnion
		if isAAAA {
			recordParam = dns.AAAARecordParam{
				Name:    cloudflare.F(r.RecordName),
				Content: cloudflare.F(addr.String()),
				Type:    cloudflare.F(dns.AAAARecordTypeAAAA),
			}
		} else {
			recordParam = dns.ARecordParam{
				Name:    cloudflare.F(r.RecordName),
				Content: cloudflare.F(addr.String()),
				Type:    cloudflare.F(dns.ARecordTypeA),
			}
		}

		_, err := client.DNS.Records.Update(context.TODO(), record.Result[0].ID, dns.RecordUpdateParams{
			ZoneID: cloudflare.F(zoneId),
			Body:   recordParam,
		})
		if err != nil {
			return fmt.Errorf("Failed to create DNS record %s: %w", r.RecordName, err)
		}
	} else {
		// 4.c More than one DNS record found, this is an error
		return fmt.Errorf("Found %d DNS records for %s, expected 0 or 1", len(record.Result), r.RecordName)
	}

	slog.Info("DNS record updated successfully", "name", r.RecordName, "type", listType, "content", addr.String())

	return nil
}

func checkIfIPv6(addr net.IP) bool {
	return addr.To4() == nil
}
