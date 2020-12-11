package gateways

import (
	"regexp"
	"strconv"
	"strings"

	"k8s.io/apimachinery/pkg/labels"

	"github.com/kiali/kiali/kubernetes"
	"github.com/kiali/kiali/models"
	"github.com/kiali/kiali/util/intutil"
)

type MultiMatchChecker struct {
	GatewaysPerNamespace [][]kubernetes.IstioObject
	existingList         map[string][]Host
}

const (
	GatewayCheckerType = "gateway"
	wildCardMatch      = "*"
)

type Host struct {
	Port            int
	Hostname        string
	Namespace       string
	ServerIndex     int
	HostIndex       int
	GatewayRuleName string
}

// Check validates that no two gateways share the same host+port combination
func (m MultiMatchChecker) Check() models.IstioValidations {
	validations := models.IstioValidations{}
	m.existingList = map[string][]Host{}

	for _, nsG := range m.GatewaysPerNamespace {
		for _, g := range nsG {
			gatewayRuleName := g.GetObjectMeta().Name
			gatewayNamespace := g.GetObjectMeta().Namespace

			selectorString := ""
			if selectorRaw, found := g.GetSpec()["selector"]; found {
				if selector, ok := selectorRaw.(map[string]interface{}); ok {
					selectorMap := map[string]string{}
					for k, v := range selector {
						selectorMap[k] = v.(string)
					}
					selectorString = labels.Set(selectorMap).String()
				}
			}

			if specServers, found := g.GetSpec()["servers"]; found {
				if servers, ok := specServers.([]interface{}); ok {
					for i, def := range servers {
						if serverDef, ok := def.(map[string]interface{}); ok {
							hosts := parsePortAndHostnames(serverDef)
							for hi, host := range hosts {
								host.ServerIndex = i
								host.HostIndex = hi
								host.GatewayRuleName = gatewayRuleName
								host.Namespace = gatewayNamespace
								duplicate, dhosts := m.findMatch(host, selectorString)
								if duplicate {
									// The above is referenced by each one below..
									currentHostValidation := createError(host.GatewayRuleName, host.Namespace, host.ServerIndex, host.HostIndex)

									// CurrentHostValidation is always the first one, so we skip it
									for i := 1; i < len(dhosts); i++ {
										dh := dhosts[i]
										refValidation := createError(dh.GatewayRuleName, dh.Namespace, dh.ServerIndex, dh.HostIndex)
										refValidation = refValidation.MergeReferences(currentHostValidation)
										currentHostValidation = currentHostValidation.MergeReferences(refValidation)
										validations = validations.MergeValidations(refValidation)
									}
									validations = validations.MergeValidations(currentHostValidation)
								}
								m.existingList[selectorString] = append(m.existingList[selectorString], host)
							}
						}
					}
				}
			}
		}
	}

	return validations
}

func createError(gatewayRuleName, namespace string, serverIndex, hostIndex int) models.IstioValidations {
	key := models.IstioValidationKey{Name: gatewayRuleName, Namespace: namespace, ObjectType: GatewayCheckerType}
	checks := models.Build("gateways.multimatch",
		"spec/servers["+strconv.Itoa(serverIndex)+"]/hosts["+strconv.Itoa(hostIndex)+"]")
	rrValidation := &models.IstioValidation{
		Name:       gatewayRuleName,
		ObjectType: GatewayCheckerType,
		Valid:      true,
		Checks: []*models.IstioCheck{
			&checks,
		},
	}

	return models.IstioValidations{key: rrValidation}
}

func parsePortAndHostnames(serverDef map[string]interface{}) []Host {
	var port int
	if portDef, found := serverDef["port"]; found {
		if ports, ok := portDef.(map[string]interface{}); ok {
			if numberDef, found := ports["number"]; found {
				if portNumber, err := intutil.Convert(numberDef); err == nil {
					port = portNumber
				}
			}
		}
	}

	if hostDef, found := serverDef["hosts"]; found {
		if hostnames, ok := hostDef.([]interface{}); ok {
			hosts := make([]Host, 0, len(hostnames))
			for _, hostinterface := range hostnames {
				if hostname, ok := hostinterface.(string); ok {
					hosts = append(hosts, Host{
						Port:     port,
						Hostname: hostname,
					})
				}
			}
			return hosts
		}
	}
	return nil
}

// findMatch uses a linear search with regexp to check for matching gateway host + port combinations. If this becomes a bottleneck for performance, replace with a graph or trie algorithm.
func (m MultiMatchChecker) findMatch(host Host, selector string) (bool, []Host) {
	duplicates := make([]Host, 0)
	for groupSelector, hostGroup := range m.existingList {
		if groupSelector != selector {
			continue
		}

		for _, h := range hostGroup {
			if h.Port == host.Port {
				// wildcardMatches will always match
				if host.Hostname == wildCardMatch || h.Hostname == wildCardMatch {
					duplicates = append(duplicates, host)
					duplicates = append(duplicates, h)
					continue
				}

				// Either one could include wildcards, so we need to check both ways and fix "*" -> ".*" for regexp engine
				current := strings.ToLower(strings.Replace(host.Hostname, "*", ".*", -1))
				previous := strings.ToLower(strings.Replace(h.Hostname, "*", ".*", -1))

				// Escaping dot chars for RegExp. Dot char means all possible chars.
				// This protects this validation to false positive for (api-dev.example.com and api.dev.example.com)
				escapedCurrent := strings.Replace(host.Hostname, ".", "\\.", -1)
				escapedPrevious := strings.Replace(h.Hostname, ".", "\\.", -1)

				// We anchor the beginning and end of the string when it's
				// to be used as a regex, so that we don't get spurious
				// substring matches, e.g., "example.com" matching
				// "foo.example.com".
				currentRegexp := strings.Join([]string{"^", escapedCurrent, "$"}, "")
				previousRegexp := strings.Join([]string{"^", escapedPrevious, "$"}, "")

				if regexp.MustCompile(currentRegexp).MatchString(previous) ||
					regexp.MustCompile(previousRegexp).MatchString(current) {
					duplicates = append(duplicates, host)
					duplicates = append(duplicates, h)
					continue
				}
			}
		}
	}
	return len(duplicates) > 0, duplicates
}
