package models

import (
	"github.com/kiali/kiali/config"
	"github.com/kiali/kiali/kubernetes"
)

// DestinationRules destinationRules
//
// This is used for returning an array of DestinationRules
//
// swagger:model destinationRules
// An array of destinationRule
// swagger:allOf
type DestinationRules struct {
	Permissions ResourcePermissions `json:"permissions"`
	Items       []DestinationRule   `json:"items"`
}

// DestinationRule destinationRule
//
// This is used for returning a DestinationRule
//
// swagger:model destinationRule
type DestinationRule struct {
	IstioBase
	Spec struct {
		Host          interface{} `json:"host,omitempty"`
		TrafficPolicy interface{} `json:"trafficPolicy,omitempty"`
		Subsets       interface{} `json:"subsets,omitempty"`
		ExportTo      interface{} `json:"exportTo,omitempty"`
	} `json:"spec"`
}

func (dRules *DestinationRules) Parse(destinationRules []kubernetes.IstioObject) {
	dRules.Items = []DestinationRule{}
	for _, dr := range destinationRules {
		destinationRule := DestinationRule{}
		destinationRule.Parse(dr)
		dRules.Items = append(dRules.Items, destinationRule)
	}
}

func (dRule *DestinationRule) Parse(destinationRule kubernetes.IstioObject) {
	dRule.IstioBase.Parse(destinationRule)
	dRule.Spec.Host = destinationRule.GetSpec()["host"]
	dRule.Spec.TrafficPolicy = destinationRule.GetSpec()["trafficPolicy"]
	dRule.Spec.Subsets = destinationRule.GetSpec()["subsets"]
	dRule.Spec.ExportTo = destinationRule.GetSpec()["exportTo"]
}

func (dRule *DestinationRule) HasCircuitBreaker(namespace string, serviceName string, version string) bool {
	if host, ok := dRule.Spec.Host.(string); ok && kubernetes.FilterByHost(host, serviceName, namespace) {
		// CB is set at DR level, so it's true for the service and all versions
		if isCircuitBreakerTrafficPolicy(dRule.Spec.TrafficPolicy) {
			return true
		}
		if subsets, ok := dRule.Spec.Subsets.([]interface{}); ok {
			cfg := config.Get()
			for _, subsetInterface := range subsets {
				if subset, ok := subsetInterface.(map[string]interface{}); ok {
					if trafficPolicy, ok := subset["trafficPolicy"]; ok && isCircuitBreakerTrafficPolicy(trafficPolicy) {
						// set the service true if it has a subset with a CB
						if version == "" {
							return true
						}
						if labels, ok := subset["labels"]; ok {
							if dLabels, ok := labels.(map[string]interface{}); ok {
								if versionValue, ok := dLabels[cfg.IstioLabels.VersionLabelName]; ok && versionValue == version {
									return true
								}
							}
						}
					}
				}
			}
		}
	}
	return false
}

func isCircuitBreakerTrafficPolicy(trafficPolicy interface{}) bool {
	if trafficPolicy == nil {
		return false
	}
	if dTrafficPolicy, ok := trafficPolicy.(map[string]interface{}); ok {
		if _, ok := dTrafficPolicy["connectionPool"]; ok {
			return true
		}
		if _, ok := dTrafficPolicy["outlierDetection"]; ok {
			return true
		}
	}
	return false
}
