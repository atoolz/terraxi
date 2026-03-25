package aws

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	awsutil "github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/route53"
	route53types "github.com/aws/aws-sdk-go-v2/service/route53/types"

	"github.com/atoolz/terraxi/pkg/types"
)

func init() {
	RegisterDiscoverer("aws_route53_zone", discoverRoute53Zones)
	RegisterDiscoverer("aws_route53_record", discoverRoute53Records)
}

// Route53 zones and records do not support inline tag filtering via ListHostedZones/ListResourceRecordSets.
func discoverRoute53Zones(ctx context.Context, p *Provider, filter types.Filter) ([]types.Resource, error) {
	if len(filter.Tags) > 0 {
		slog.Warn("Tag filtering not supported for Route53 resources (ListHostedZones does not return tags). Returning all zones.")
	}
	var resources []types.Resource
	var marker *string

	for {
		out, err := p.route53.ListHostedZones(ctx, &route53.ListHostedZonesInput{Marker: marker})
		if err != nil {
			if isAccessDenied(err) {
				return nil, fmt.Errorf("insufficient permissions for route53:ListHostedZones: %w", err)
			}
			return nil, fmt.Errorf("listing hosted zones: %w", err)
		}

		for _, zone := range out.HostedZones {
			// Zone ID format is "/hostedzone/Z1234", terraform wants just "Z1234"
			zoneID := awsutil.ToString(zone.Id)
			zoneID = strings.TrimPrefix(zoneID, "/hostedzone/")

			resources = append(resources, types.Resource{
				Type:   "aws_route53_zone",
				ID:     zoneID,
				Name:   awsutil.ToString(zone.Name),
				Region: "", // Route53 is global
			})
		}

		if !out.IsTruncated {
			break
		}
		marker = out.NextMarker
	}

	slog.Debug("Route53 zones discovery complete", "count", len(resources))
	return resources, nil
}

func discoverRoute53Records(ctx context.Context, p *Provider, filter types.Filter) ([]types.Resource, error) {
	// Records require a hosted zone. Discover zones first.
	zones, err := discoverRoute53Zones(ctx, p, filter)
	if err != nil {
		return nil, err
	}

	var resources []types.Resource
	for _, zone := range zones {
		var startName *string
		var startType route53types.RRType

		for {
			out, err := p.route53.ListResourceRecordSets(ctx, &route53.ListResourceRecordSetsInput{
				HostedZoneId:    &zone.ID,
				StartRecordName: startName,
				StartRecordType: startType,
			})
			if err != nil {
				if isAccessDenied(err) || isNotFound(err) {
					break
				}
				return nil, fmt.Errorf("listing records for zone %s: %w", zone.ID, err)
			}

			for _, rr := range out.ResourceRecordSets {
				rawName := awsutil.ToString(rr.Name)
				rrType := string(rr.Type)

				// Skip NS and SOA records for the zone apex (AWS-managed)
				if (rrType == "NS" || rrType == "SOA") && rawName == zone.Name {
					continue
				}

				// Strip trailing dot for import ID (Terraform expects no trailing dot)
				name := strings.TrimSuffix(rawName, ".")
				importID := fmt.Sprintf("%s_%s_%s", zone.ID, name, rrType)

				resources = append(resources, types.Resource{
					Type:   "aws_route53_record",
					ID:     importID,
					Name:   fmt.Sprintf("%s-%s", name, rrType),
					Region: "",
					Dependencies: []types.ResourceRef{
						{Type: "aws_route53_zone", ID: zone.ID},
					},
				})
			}

			if !out.IsTruncated {
				break
			}
			startName = out.NextRecordName
			startType = out.NextRecordType
		}
	}

	slog.Debug("Route53 records discovery complete", "count", len(resources))
	return resources, nil
}
