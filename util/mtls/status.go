package mtls

import (
	"github.com/kiali/kiali/kubernetes"
)

const (
	MTLSEnabled          = "MTLS_ENABLED"
	MTLSPartiallyEnabled = "MTLS_PARTIALLY_ENABLED"
	MTLSNotEnabled       = "MTLS_NOT_ENABLED"
	MTLSDisabled         = "MTLS_DISABLED"
)

type MtlsStatus struct {
	Namespace           string
	PeerAuthentications []kubernetes.IstioObject
	DestinationRules    []kubernetes.IstioObject
	AutoMtlsEnabled     bool
	AllowPermissive     bool
}

type TlsStatus struct {
	DestinationRuleStatus    string
	PeerAuthenticationStatus string
	OverallStatus            string
}

func (m MtlsStatus) hasPeerAuthnNamespacemTLSDefinition() string {
	for _, p := range m.PeerAuthentications {
		if _, mode := kubernetes.PeerAuthnHasMTLSEnabled(p); mode != "" {
			return mode
		}
	}

	return ""
}

func (m MtlsStatus) hasDesinationRuleEnablingNamespacemTLS() string {
	for _, dr := range m.DestinationRules {
		if _, mode := kubernetes.DestinationRuleHasNamespaceWideMTLSEnabled(m.Namespace, dr); mode != "" {
			return mode
		}
	}

	return ""
}

func (m MtlsStatus) NamespaceMtlsStatus() TlsStatus {
	drStatus := m.hasDesinationRuleEnablingNamespacemTLS()
	paStatus := m.hasPeerAuthnNamespacemTLSDefinition()
	return m.finalStatus(drStatus, paStatus)
}

func (m MtlsStatus) finalStatus(drStatus, paStatus string) TlsStatus {
	finalStatus := MTLSPartiallyEnabled

	mtlsEnabled := drStatus == "ISTIO_MUTUAL" || drStatus == "MUTUAL" || (drStatus == "" && m.AutoMtlsEnabled)
	mtlsDisabled := drStatus == "DISABLE" || (drStatus == "" && m.AutoMtlsEnabled)

	if (paStatus == "STRICT" || (paStatus == "PERMISSIVE" && m.AllowPermissive)) && mtlsEnabled {
		finalStatus = MTLSEnabled
	} else if paStatus == "DISABLE" && mtlsDisabled {
		finalStatus = MTLSDisabled
	} else if paStatus == "" && drStatus == "" {
		finalStatus = MTLSNotEnabled
	}

	return TlsStatus{
		DestinationRuleStatus:    drStatus,
		PeerAuthenticationStatus: paStatus,
		OverallStatus:            finalStatus,
	}
}

func (m MtlsStatus) MeshMtlsStatus() TlsStatus {
	drStatus := m.hasDestinationRuleMeshTLSDefinition()
	paStatus := m.hasPeerAuthnMeshTLSDefinition()
	return TlsStatus{
		DestinationRuleStatus:    drStatus,
		PeerAuthenticationStatus: paStatus,
		OverallStatus:            m.OverallMtlsStatus(TlsStatus{}, m.finalStatus(drStatus, paStatus)),
	}
}

func (m MtlsStatus) hasPeerAuthnMeshTLSDefinition() string {
	for _, mp := range m.PeerAuthentications {
		if _, mode := kubernetes.PeerAuthnHasMTLSEnabled(mp); mode != "" {
			return mode
		}
	}
	return ""
}

func (m MtlsStatus) hasDestinationRuleMeshTLSDefinition() string {
	for _, dr := range m.DestinationRules {
		if _, mode := kubernetes.DestinationRuleHasMTLSEnabledForHost("*.local", dr); mode != "" {
			return mode
		}
	}
	return ""
}

func (m MtlsStatus) OverallMtlsStatus(nsStatus, meshStatus TlsStatus) string {
	var status = MTLSPartiallyEnabled
	if nsStatus.hasDefinedTls() {
		status = nsStatus.OverallStatus
	} else if nsStatus.hasPartialTlsConfig() {
		status = m.inheritedOverallStatus(nsStatus, meshStatus)
	} else if meshStatus.hasDefinedTls() {
		status = meshStatus.OverallStatus
	} else if meshStatus.hasNoConfig() {
		status = MTLSNotEnabled
	} else if meshStatus.hasPartialDisabledConfig() {
		status = MTLSDisabled
	} else if meshStatus.hasHalfTlsConfigDefined(m.AutoMtlsEnabled, m.AllowPermissive) {
		status = MTLSEnabled
	} else if !m.AutoMtlsEnabled && meshStatus.hasPartialTlsConfig() {
		status = MTLSPartiallyEnabled
	}
	return status
}

func (m MtlsStatus) inheritedOverallStatus(nsStatus, meshStatus TlsStatus) string {
	var partialDRStatus, partialPAStatus = nsStatus.DestinationRuleStatus, nsStatus.PeerAuthenticationStatus
	if nsStatus.DestinationRuleStatus == "" {
		partialDRStatus = meshStatus.DestinationRuleStatus
	}

	if nsStatus.PeerAuthenticationStatus == "" {
		partialPAStatus = meshStatus.PeerAuthenticationStatus
	}

	return m.OverallMtlsStatus(TlsStatus{},
		m.finalStatus(partialDRStatus, partialPAStatus),
	)
}

func (t TlsStatus) hasDefinedTls() bool {
	return t.OverallStatus == MTLSEnabled || t.OverallStatus == MTLSDisabled
}

func (t TlsStatus) hasPartialTlsConfig() bool {
	return t.OverallStatus == MTLSPartiallyEnabled
}

func (t TlsStatus) hasHalfTlsConfigDefined(autoMtls, allowPermissive bool) bool {
	defined := false
	if autoMtls {
		defined = t.PeerAuthenticationStatus == "STRICT" && t.DestinationRuleStatus == "" ||
			(t.DestinationRuleStatus == "ISTIO_MUTUAL" || t.DestinationRuleStatus == "MUTUAL") && t.PeerAuthenticationStatus == ""

		if !defined && allowPermissive {
			defined = t.PeerAuthenticationStatus == "PERMISSIVE" && t.DestinationRuleStatus == ""
		}
	}

	return defined
}

func (t TlsStatus) hasNoConfig() bool {
	return t.PeerAuthenticationStatus == "" && t.DestinationRuleStatus == ""
}

func (t TlsStatus) hasPartialDisabledConfig() bool {
	return t.PeerAuthenticationStatus == "DISABLE" && t.DestinationRuleStatus == "" ||
		t.DestinationRuleStatus == "DISABLE" && t.PeerAuthenticationStatus == ""
}
