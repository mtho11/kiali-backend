package destinationrules

import (
	"github.com/kiali/kiali/kubernetes"
	"github.com/kiali/kiali/models"
)

type DisabledNamespaceWideMTLSChecker struct {
	DestinationRule kubernetes.IstioObject
	MTLSDetails     kubernetes.MTLSDetails
}

// Check if a the PeerAuthn is allows non-mtls traffic when DestinationRule explicitly disables mTLS ns-wide
func (m DisabledNamespaceWideMTLSChecker) Check() ([]*models.IstioCheck, bool) {
	validations := make([]*models.IstioCheck, 0)

	// Stop validation if DestinationRule doesn't explicitly disables mTLS
	if _, mode := kubernetes.DestinationRuleHasNamespaceWideMTLSEnabled(m.DestinationRule.GetObjectMeta().Namespace, m.DestinationRule); mode != "DISABLE" {
		return validations, true
	}

	// otherwise, check among PeerAuthentications for a rule enabling mTLS
	for _, mp := range m.MTLSDetails.PeerAuthentications {
		if enabled, mode := kubernetes.PeerAuthnHasMTLSEnabled(mp); enabled {
			// If PeerAuthn has mTLS enabled in STRICT mode
			// traffic going through DestinationRule won't work
			if mode == "STRICT" {
				check := models.Build("destinationrules.mtls.policymtlsenabled", "spec/trafficPolicy/tls/mode")
				return append(validations, &check), false
			} else {
				// If PeerAuthn has mTLS enabled in PERMISSIVE mode
				// traffic going through DestinationRule will work
				// no need for further analysis in MeshPeerAuthentications
				return validations, true
			}
		}
	}

	// In case any PeerAuthn enables mTLS, check among MeshPeerAuthentications for a rule enabling it
	for _, mp := range m.MTLSDetails.MeshPeerAuthentications {
		if strictMode := kubernetes.PeerAuthnHasStrictMTLS(mp); strictMode {
			check := models.Build("destinationrules.mtls.meshpolicymtlsenabled", "spec/trafficPolicy/tls/mode")
			return append(validations, &check), false
		}
	}

	return validations, true
}
