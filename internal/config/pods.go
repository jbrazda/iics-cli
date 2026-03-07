package config

import (
	"fmt"
	"sort"
	"strings"
)

// podURLs maps region/POD identifiers to their login base hostnames.
var podURLs = map[string]string{
	"US":     "dm-us.informaticacloud.com",
	"USW1":   "dm-us.informaticacloud.com",
	"USE2":   "dm-us.informaticacloud.com",
	"USW3":   "dm-us.informaticacloud.com",
	"USE4":   "dm-us.informaticacloud.com",
	"USW5":   "dm-us.informaticacloud.com",
	"USE6":   "dm-us.informaticacloud.com",
	"USW1-1": "dm1-us.informaticacloud.com",
	"USW3-1": "dm1-us.informaticacloud.com",
	"USW1-2": "dm2-us.informaticacloud.com",
	"CAC1":   "dm-na.informaticacloud.com",
	"APSE1":  "dm-ap.informaticacloud.com",
	"APNE1":  "dm1-ap.informaticacloud.com",
	"EMEA":   "dm-em.informaticacloud.com",
	"EMWE1":  "dm-em.informaticacloud.com",
	"APJ":    "dm-ap.informaticacloud.com",
}

// LoginURL constructs the full login URL for a region.
func LoginURL(region string) (string, error) {
	host, ok := podURLs[strings.ToUpper(region)]
	if !ok {
		return "", fmt.Errorf("unknown IICS region %q; valid regions: %s", region, ValidRegions())
	}
	return fmt.Sprintf("https://%s/saas/public/core/v3/login", host), nil
}

// ValidRegions returns a sorted comma-separated list of valid region identifiers.
func ValidRegions() string {
	regions := make([]string, 0, len(podURLs))
	for r := range podURLs {
		regions = append(regions, r)
	}
	sort.Strings(regions)
	return strings.Join(regions, ", ")
}
