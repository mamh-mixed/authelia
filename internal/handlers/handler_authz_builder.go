package handlers

import (
	"fmt"
	"time"

	"github.com/authelia/authelia/v4/internal/configuration/schema"
	"github.com/authelia/authelia/v4/internal/logging"
	"github.com/authelia/authelia/v4/internal/utils"
)

// NewAuthzBuilder creates a new AuthzBuilder.
func NewAuthzBuilder() *AuthzBuilder {
	return &AuthzBuilder{
		config: AuthzConfig{RefreshInterval: time.Second * -1},
	}
}

// WithStrategies replaces all strategies in this builder with the provided value.
func (b *AuthzBuilder) WithStrategies(strategies ...AuthnStrategy) *AuthzBuilder {
	b.strategies = strategies

	return b
}

// WithStrategyCookie adds the Cookie header strategy to the strategies in this builder.
func (b *AuthzBuilder) WithStrategyCookie(refreshInterval time.Duration) *AuthzBuilder {
	b.strategies = append(b.strategies, NewCookieSessionAuthnStrategy(refreshInterval))

	return b
}

// WithStrategyAuthorization adds the Authorization header strategy to the strategies in this builder.
func (b *AuthzBuilder) WithStrategyAuthorization() *AuthzBuilder {
	b.strategies = append(b.strategies, NewHeaderAuthorizationAuthnStrategy())

	return b
}

// WithStrategyProxyAuthorization adds the Proxy-Authorization header strategy to the strategies in this builder.
func (b *AuthzBuilder) WithStrategyProxyAuthorization() *AuthzBuilder {
	b.strategies = append(b.strategies, NewHeaderProxyAuthorizationAuthnStrategy())

	return b
}

// WithImplementationLegacy configures this builder to output an Authz which is used with the Legacy
// implementation which is a mix of the other implementations and usually works with most proxies.
func (b *AuthzBuilder) WithImplementationLegacy() *AuthzBuilder {
	b.impl = AuthzImplLegacy

	return b
}

// WithImplementationForwardAuth configures this builder to output an Authz which is used with the ForwardAuth
// implementation traditionally used by Traefik, Caddy, and Skipper.
func (b *AuthzBuilder) WithImplementationForwardAuth() *AuthzBuilder {
	b.impl = AuthzImplForwardAuth

	return b
}

// WithImplementationAuthRequest configures this builder to output an Authz which is used with the AuthRequest
// implementation traditionally used by NGINX.
func (b *AuthzBuilder) WithImplementationAuthRequest() *AuthzBuilder {
	b.impl = AuthzImplAuthRequest

	return b
}

// WithConfig allows configuring the Authz config by providing a *schema.Configuration. This function converts it to
// an AuthzConfig and assigns it to the builder.
func (b *AuthzBuilder) WithConfig(config *schema.Configuration) *AuthzBuilder {
	if config == nil {
		return b
	}

	var refreshInterval time.Duration

	switch config.AuthenticationBackend.RefreshInterval {
	case schema.ProfileRefreshDisabled:
		refreshInterval = time.Second * -1
	case schema.ProfileRefreshAlways:
		refreshInterval = time.Second * 0
	default:
		refreshInterval, _ = utils.ParseDurationString(config.AuthenticationBackend.RefreshInterval)
	}

	b.config = AuthzConfig{
		RefreshInterval: refreshInterval,
		Domains: []AuthzDomain{
			{
				Name:      fmt.Sprintf(".%s", config.Session.Domain),
				PortalURL: nil,
			},
		},
	}

	return b
}

// WithEndpointConfig configures the AuthzBuilder with a *schema.ServerAuthzEndpointConfig. Should be called AFTER
// WithConfig or WithAuthzConfig.
func (b *AuthzBuilder) WithEndpointConfig(config schema.ServerAuthzEndpointConfig) *AuthzBuilder {
	logger := logging.Logger()

	switch config.Implementation {
	case AuthzImplForwardAuth.String():
		logger.Debugf("adding endpoint as ForwardAuth")

		b.WithImplementationForwardAuth()
	case AuthzImplAuthRequest.String():
		logger.Debugf("adding endpoint as AuthRequest")

		b.WithImplementationAuthRequest()
	default:
		logger.Debugf("adding endpoint as Legacy")

		b.WithImplementationLegacy()
	}

	b.WithStrategies()

	for _, strategy := range config.AuthnStrategies {
		switch strategy.Name {
		case AuthnStrategyCookieSession:
			logger.Debugf("adding strategy CookieSession")

			b.strategies = append(b.strategies, NewCookieSessionAuthnStrategy(b.config.RefreshInterval))
		case AuthnStrategyHeaderAuthorization:
			logger.Debugf("adding strategy HeaderAuthorization")

			b.strategies = append(b.strategies, NewHeaderAuthorizationAuthnStrategy())
		case AuthnStrategyHeaderProxyAuthorization:
			logger.Debugf("adding strategy HeaderProxyAuthorization")

			b.strategies = append(b.strategies, NewHeaderProxyAuthorizationAuthnStrategy())
		case AuthnStrategyHeaderAuthRequestProxyAuthorization:
			logger.Debugf("adding strategy HeaderAuthRequestProxyAuthorization")

			b.strategies = append(b.strategies, NewHeaderAuthRequestProxyAuthorizationAuthnStrategy())
		case AuthnStrategyHeaderLegacy:
			logger.Debugf("adding strategy HeaderLegacy")

			b.strategies = append(b.strategies, NewHeaderLegacyAuthnStrategy())
		}
	}

	return b
}

// WithAuthzConfig allows configuring the Authz config by providing a AuthzConfig directly. Recommended this is only
// used in testing and WithConfig is used instead.
func (b *AuthzBuilder) WithAuthzConfig(config AuthzConfig) *AuthzBuilder {
	b.config = config

	return b
}

// Build returns a new Authz from the currently configured options in this builder.
func (b *AuthzBuilder) Build() (authz *Authz) {
	logger := logging.Logger()

	authz = &Authz{
		config:            b.config,
		strategies:        b.strategies,
		fObjectVerify:     authzObjectVerifyStandard,
		fHandleAuthorized: authzHandleAuthorizedStandard,
	}

	logger.Debugf("building endpoint with config %+v", b.config)

	for _, strat := range b.strategies {
		switch s := strat.(type) {
		case *CookieSessionAuthnStrategy:
			logger.Debugf("has strategy CookieSession")
		case *HeaderAuthnStrategy:
			logger.Debugf("has strategy Header type %s", s.headerAuthorize)
		case *HeaderLegacyAuthnStrategy:
			logger.Debugf("has strategy LegacyHeader")
		}
	}

	if len(authz.strategies) == 0 {
		logger.Debugf("building endpoint with default straegies for impl %s", b.impl)

		switch b.impl {
		case AuthzImplLegacy:
			authz.strategies = []AuthnStrategy{NewHeaderLegacyAuthnStrategy(), NewCookieSessionAuthnStrategy(b.config.RefreshInterval)}
		case AuthzImplAuthRequest:
			authz.strategies = []AuthnStrategy{NewHeaderAuthRequestProxyAuthorizationAuthnStrategy(), NewCookieSessionAuthnStrategy(b.config.RefreshInterval)}
		default:
			authz.strategies = []AuthnStrategy{NewHeaderProxyAuthorizationAuthnStrategy(), NewCookieSessionAuthnStrategy(b.config.RefreshInterval)}
		}
	}

	switch b.impl {
	case AuthzImplLegacy:
		authz.fObjectGet = authzGetObjectImplLegacy
		authz.fHandleUnauthorized = authzHandleUnauthorizedImplLegacy
	case AuthzImplForwardAuth:
		authz.fObjectGet = authzGetObjectImplForwardAuth
		authz.fHandleUnauthorized = authzHandleUnauthorizedImplForwardAuth
	case AuthzImplAuthRequest:
		authz.fObjectGet = authzGetObjectImplAuthRequest
		authz.fHandleUnauthorized = authzHandleUnauthorizedImplAuthRequest
	}

	return authz
}
