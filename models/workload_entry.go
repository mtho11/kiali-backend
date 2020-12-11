package models

import (
	"github.com/kiali/kiali/kubernetes"
)

// WorkloadEntries workloadEntries
//
// This is used for returning an array of WorkloadEntry
//
// swagger:model workloadEntries
// An array of workloadEntry
// swagger:allOf
type WorkloadEntries []WorkloadEntry

// WorkloadEntry workloadEntry
//
// This is used for returning an WorkloadEntry
//
// swagger:model workloadEntry
type WorkloadEntry struct {
	IstioBase
	Spec struct {
		Address        interface{} `json:"address"`
		Ports          interface{} `json:"ports"`
		Labels         interface{} `json:"labels"`
		Network        interface{} `json:"network"`
		Locality       interface{} `json:"locality"`
		Weight         interface{} `json:"weight"`
		ServiceAccount interface{} `json:"serviceAccount"`
	} `json:"spec"`
}

func (wes *WorkloadEntries) Parse(workloadEntries []kubernetes.IstioObject) {
	for _, we := range workloadEntries {
		workloadEntry := WorkloadEntry{}
		workloadEntry.Parse(we)
		*wes = append(*wes, workloadEntry)
	}
}

func (we *WorkloadEntry) Parse(workloadEntry kubernetes.IstioObject) {
	we.IstioBase.Parse(workloadEntry)
	we.Spec.Address = workloadEntry.GetSpec()["address"]
	we.Spec.Ports = workloadEntry.GetSpec()["ports"]
	we.Spec.Labels = workloadEntry.GetSpec()["labels"]
	we.Spec.Network = workloadEntry.GetSpec()["network"]
	we.Spec.Locality = workloadEntry.GetSpec()["locality"]
	we.Spec.Weight = workloadEntry.GetSpec()["weight"]
	we.Spec.ServiceAccount = workloadEntry.GetSpec()["serviceAccount"]
}
