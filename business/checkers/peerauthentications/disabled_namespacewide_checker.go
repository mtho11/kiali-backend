package peerauthentications

import (
	"github.com/kiali/kiali/kubernetes"
	"github.com/kiali/kiali/models"
)

type DisabledNamespaceWideChecker struct {
	PeerAuthn        kubernetes.IstioObject
	DestinationRules []kubernetes.IstioObject
}

func (c DisabledNamespaceWideChecker) Check() ([]*models.IstioCheck, bool) {
	validations := make([]*models.IstioCheck, 0)

	// Validation only affects to PeerAuthn disabling mTLS
	if _, mode := kubernetes.PeerAuthnHasMTLSEnabled(c.PeerAuthn); mode != "DISABLE" {
		return validations, true
	}

	for _, dr := range c.DestinationRules {
		// If ns-wide Destination Rule enabling mtls found, error found
		if _, mode := kubernetes.DestinationRuleHasNamespaceWideMTLSEnabled(c.PeerAuthn.GetObjectMeta().Namespace, dr); mode == "ISTIO_MUTUAL" || mode == "MUTUAL" {
			check := models.Build("peerauthentications.mtls.disabledestinationrulemissing", "spec/mtls")
			return append(validations, &check), false
		}

		if _, mode := kubernetes.DestinationRuleHasMeshWideMTLSEnabled(dr); mode == "ISTIO_MUTUAL" || mode == "MUTUAL" {
			check := models.Build("peerauthentications.mtls.disabledestinationrulemissing", "spec/mtls")
			return append(validations, &check), false
		}
	}

	return validations, true
}
