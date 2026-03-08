package config

import (
	"fmt"
	"sort"
	"strings"
)

// podURLs maps region/POD identifiers to their login base hostname.
// see https://docs.informatica.com/integration-cloud/2024-1/administration-guide/administration-overview/regions-and-pods for details.
var podURLs = map[string]string{
	"APAUC1": "dm1-apau.informaticacloud.com",
	"APJ":    "dm-ap.informaticacloud.com",
	"APNE1":  "dm1-ap.informaticacloud.com",
	"APNE2":  "dm-apne.informaticacloud.com",
	"APSE1":  "dm-ap.informaticacloud.com",
	"APSE2":  "dm1-apse.informaticacloud.com",
	"CAC1":   "dm-na.informaticacloud.com",
	"EMC1":   "dm1-em.informaticacloud.com",
	"EMEA":   "dm-em.informaticacloud.com",
	"EMWE1":  "dm-em.informaticacloud.com",
	"UK1":    "dm-uk.informaticacloud.com",
	"US":     "dm-us.informaticacloud.com",
	"USE2":   "dm-us.informaticacloud.com",
	"USE4":   "dm-us.informaticacloud.com",
	"USE6":   "dm-us.informaticacloud.com",
	"USW1-1": "dm1-us.informaticacloud.com",
	"USW1-2": "dm2-us.informaticacloud.com",
	"USW1":   "dm-us.informaticacloud.com",
	"USW3-1": "dm1-us.informaticacloud.com",
	"USW3":   "dm-us.informaticacloud.com",
	"USW5":   "dm-us.informaticacloud.com",
}

// LoginURL constructs the full login URL for a region.
// Note: IICS actually provides three different login endpoints for each version of the api v1, v2, and v3.
// All endpoints return the same token. but return different login responses. This client uses the v3 endpoint.
// The v3 endpoint is the most recent and should be used for all new development.
// See https://docs.informatica.com/cloud-common-services/administrator/current-version/rest-api-reference/platform-rest-api-version-3-resources/login.html for details.
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
