package gateways

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/kiali/kiali/config"
	"github.com/kiali/kiali/kubernetes"
	"github.com/kiali/kiali/models"
	"github.com/kiali/kiali/tests/data"
)

func TestCorrectGateways(t *testing.T) {
	conf := config.NewConfig()
	config.Set(conf)

	assert := assert.New(t)

	gwObject := data.AddServerToGateway(data.CreateServer([]string{"valid"}, 80, "http", "http"),
		data.CreateEmptyGateway("validgateway", "test", map[string]string{
			"app": "real",
		}))

	gws := [][]kubernetes.IstioObject{{gwObject}}

	validations := MultiMatchChecker{
		GatewaysPerNamespace: gws,
	}.Check()

	assert.Empty(validations)
	_, ok := validations[models.IstioValidationKey{ObjectType: "gateway", Namespace: "test", Name: "validgateway"}]
	assert.False(ok)
}

func TestCaseMatching(t *testing.T) {
	conf := config.NewConfig()
	config.Set(conf)

	assert := assert.New(t)

	gwObject := data.AddServerToGateway(data.CreateServer(
		[]string{
			"NOTFINE.example.com",
			"notfine.example.com",
		}, 80, "http", "http"),

		data.CreateEmptyGateway("foxxed", "test", map[string]string{
			"app": "canidae",
		}))

	gws := [][]kubernetes.IstioObject{{gwObject}}

	validations := MultiMatchChecker{
		GatewaysPerNamespace: gws,
	}.Check()

	assert.NotEmpty(validations)
	assert.Equal(1, len(validations))
	validation, ok := validations[models.IstioValidationKey{ObjectType: "gateway", Namespace: "test", Name: "foxxed"}]
	assert.True(ok)
	assert.True(validation.Valid)
}

func TestDashSubdomainMatching(t *testing.T) {
	conf := config.NewConfig()
	config.Set(conf)

	assert := assert.New(t)

	gwObject := data.AddServerToGateway(data.CreateServer(
		[]string{
			"api.dev.example.com",
			"api-dev.example.com",
		}, 80, "http", "http"),

		data.CreateEmptyGateway("foxxed", "test", map[string]string{
			"app": "canidae",
		}))

	gws := [][]kubernetes.IstioObject{{gwObject}}

	validations := MultiMatchChecker{
		GatewaysPerNamespace: gws,
	}.Check()

	assert.Empty(validations)
}

// Two gateways can share port+host unless they use different ingress
func TestSameHostPortConfigInDifferentIngress(t *testing.T) {
	conf := config.NewConfig()
	config.Set(conf)

	assert := assert.New(t)

	gwObject := data.AddServerToGateway(data.CreateServer([]string{"reviews"}, 80, "http", "http"),
		data.CreateEmptyGateway("validgateway", "test", map[string]string{
			"app": "istio-ingress-pub",
		}))

	// Another namespace
	gwObject2 := data.AddServerToGateway(data.CreateServer([]string{"reviews"}, 80, "http", "http"),
		data.CreateEmptyGateway("stillvalid", "test", map[string]string{
			"app": "istio-ingress-prv",
		}))

	gws := [][]kubernetes.IstioObject{{gwObject}, {gwObject2}}

	validations := MultiMatchChecker{
		GatewaysPerNamespace: gws,
	}.Check()

	assert.Equal(0, len(validations))
}

func TestSameHostPortConfigInDifferentNamespace(t *testing.T) {
	conf := config.NewConfig()
	config.Set(conf)

	assert := assert.New(t)

	gwObject := data.AddServerToGateway(data.CreateServer([]string{"valid"}, 80, "http", "http"),
		data.CreateEmptyGateway("validgateway", "test", map[string]string{
			"app": "real",
		}))

	// Another namespace
	gwObject2 := data.AddServerToGateway(data.CreateServer([]string{"valid"}, 80, "http", "http"),
		data.CreateEmptyGateway("stillvalid", "bookinfo", map[string]string{
			"app": "real",
		}))

	gws := [][]kubernetes.IstioObject{{gwObject}, {gwObject2}}

	validations := MultiMatchChecker{
		GatewaysPerNamespace: gws,
	}.Check()

	assert.NotEmpty(validations)
	assert.Equal(2, len(validations))
	validation, ok := validations[models.IstioValidationKey{ObjectType: "gateway", Namespace: "bookinfo", Name: "stillvalid"}]
	assert.True(ok)
	assert.True(validation.Valid)

	secValidation, ok := validations[models.IstioValidationKey{ObjectType: "gateway", Namespace: "test", Name: "validgateway"}]
	assert.True(ok)
	assert.True(secValidation.Valid)

	// Check references
	assert.Equal(1, len(validation.References))
	assert.Equal(1, len(secValidation.References))
}

func TestWildCardMatchingHost(t *testing.T) {
	conf := config.NewConfig()
	config.Set(conf)

	assert := assert.New(t)

	gwObject := data.AddServerToGateway(data.CreateServer([]string{"valid"}, 80, "http", "http"),
		data.CreateEmptyGateway("validgateway", "test", map[string]string{
			"istio": "istio-ingress",
		}))

	// Another namespace
	gwObject2 := data.AddServerToGateway(data.CreateServer([]string{"*"}, 80, "http", "http"),
		data.CreateEmptyGateway("stillvalid", "test", map[string]string{
			"istio": "istio-ingress",
		}))

	// Another namespace
	gwObject3 := data.AddServerToGateway(data.CreateServer([]string{"*.justhost.com"}, 80, "http", "http"),
		data.CreateEmptyGateway("keepsvalid", "test", map[string]string{
			"istio": "istio-ingress",
		}))

	gws := [][]kubernetes.IstioObject{{gwObject}, {gwObject2, gwObject3}}

	validations := MultiMatchChecker{
		GatewaysPerNamespace: gws,
	}.Check()

	assert.NotEmpty(validations)
	assert.Equal(3, len(validations))
	validation, ok := validations[models.IstioValidationKey{ObjectType: "gateway", Namespace: "test", Name: "stillvalid"}]
	assert.True(ok)
	assert.True(validation.Valid)

	// valid should have "*" as ref
	// "*" should have valid and *.just as ref
	// *.just should have "*" as ref
	for _, v := range validations {
		if v.Name == "stillvalid" {
			assert.Equal(2, len(v.References))
		} else {
			assert.Equal(1, len(v.References))
		}
	}
}

func TestAnotherSubdomainWildcardCombination(t *testing.T) {
	conf := config.NewConfig()
	config.Set(conf)

	assert := assert.New(t)

	gwObject := data.AddServerToGateway(data.CreateServer(
		[]string{
			"*.echidna.com",
			"tachyglossa.echidna.com",
		}, 80, "http", "http"),

		data.CreateEmptyGateway("shouldnotbevalid", "test", map[string]string{
			"app": "monotreme",
		}))

	gws := [][]kubernetes.IstioObject{{gwObject}}

	validations := MultiMatchChecker{
		GatewaysPerNamespace: gws,
	}.Check()

	assert.NotEmpty(validations)
	assert.Equal(1, len(validations))
	validation, ok := validations[models.IstioValidationKey{ObjectType: "gateway", Namespace: "test", Name: "shouldnotbevalid"}]
	assert.True(ok)
	assert.True(validation.Valid)
}

func TestNoMatchOnSubdomainHost(t *testing.T) {
	conf := config.NewConfig()
	config.Set(conf)

	assert := assert.New(t)

	gwObject := data.AddServerToGateway(data.CreateServer(
		[]string{
			"example.com",
			"thisisfine.example.com",
		}, 80, "http", "http"),

		data.CreateEmptyGateway("shouldbevalid", "test", map[string]string{
			"app": "someother",
		}))

	gws := [][]kubernetes.IstioObject{{gwObject}}

	validations := MultiMatchChecker{
		GatewaysPerNamespace: gws,
	}.Check()

	assert.Empty(validations)
}

func TestTwoWildCardsMatching(t *testing.T) {
	conf := config.NewConfig()
	config.Set(conf)

	assert := assert.New(t)

	gwObject := data.AddServerToGateway(data.CreateServer([]string{"*"}, 80, "http", "http"),
		data.CreateEmptyGateway("validgateway", "test", map[string]string{
			"istio": "istio-ingress",
		}))

	// Another namespace
	gwObject2 := data.AddServerToGateway(data.CreateServer([]string{"*"}, 80, "http", "http"),
		data.CreateEmptyGateway("stillvalid", "test", map[string]string{
			"istio": "istio-ingress",
		}))

	gws := [][]kubernetes.IstioObject{{gwObject}, {gwObject2}}

	validations := MultiMatchChecker{
		GatewaysPerNamespace: gws,
	}.Check()

	assert.NotEmpty(validations)
	assert.Equal(2, len(validations))
	validation, ok := validations[models.IstioValidationKey{ObjectType: "gateway", Namespace: "test", Name: "stillvalid"}]
	assert.True(ok)
	assert.True(validation.Valid)
	assert.Equal("spec/servers[0]/hosts[0]", validation.Checks[0].Path)
}

func TestDuplicateGatewaysErrorCount(t *testing.T) {
	conf := config.NewConfig()
	config.Set(conf)

	assert := assert.New(t)

	gwObject := data.AddServerToGateway(data.CreateServer([]string{"valid", "second.valid"}, 80, "http", "http"),
		data.CreateEmptyGateway("validgateway", "test", map[string]string{
			"app": "real",
		}))

	gwObjectIdentical := data.AddServerToGateway(data.CreateServer([]string{"valid", "second.valid"}, 80, "http", "http"),
		data.CreateEmptyGateway("duplicatevalidgateway", "test", map[string]string{
			"app": "real",
		}))

	gws := [][]kubernetes.IstioObject{{gwObject, gwObjectIdentical}}

	validations := MultiMatchChecker{
		GatewaysPerNamespace: gws,
	}.Check()

	assert.NotEmpty(validations)
	validgateway, ok := validations[models.IstioValidationKey{ObjectType: "gateway", Namespace: "test", Name: "validgateway"}]
	assert.True(ok)

	duplicatevalidgateway, ok := validations[models.IstioValidationKey{ObjectType: "gateway", Namespace: "test", Name: "duplicatevalidgateway"}]
	assert.True(ok)

	assert.Equal(2, len(validgateway.Checks))
	assert.Equal("spec/servers[0]/hosts[0]", validgateway.Checks[0].Path)
	assert.Equal("spec/servers[0]/hosts[1]", validgateway.Checks[1].Path)

	assert.Equal(2, len(duplicatevalidgateway.Checks))
	assert.Equal("spec/servers[0]/hosts[0]", duplicatevalidgateway.Checks[0].Path)
	assert.Equal("spec/servers[0]/hosts[1]", duplicatevalidgateway.Checks[1].Path)
}
