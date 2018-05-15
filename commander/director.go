package commander

import (
	boshdir "github.com/cloudfoundry/bosh-cli/director"
	boshuaa "github.com/cloudfoundry/bosh-cli/uaa"
	boshlog "github.com/cloudfoundry/bosh-utils/logger"
	"fmt"
)

func GenerateDirector(boshDirector BoshDirector, loggerBosh boshlog.Logger) (boshdir.Director, error) {
	directorNoAuth, err := buildDirector(boshDirector, nil, loggerBosh)
	if err != nil {
		return nil, err
	}

	info, err := directorNoAuth.Info()
	if err != nil {
		return nil, err
	}

	var uaa boshuaa.UAA
	if info.Auth.Type == "uaa" {
		uaaUrl := fmt.Sprintf("%s", info.Auth.Options["url"])
		uaa, err = buildUAA(boshDirector, uaaUrl, loggerBosh)
		if err != nil {
			return nil, err
		}
	}

	director, err := buildDirector(boshDirector, uaa, loggerBosh)
	if err != nil {
		return nil, err
	}
	return director, nil
}

func buildUAA(boshDirector BoshDirector, uaaUrl string, loggerBosh boshlog.Logger) (boshuaa.UAA, error) {

	if uaaUrl == "" {
		return nil, nil
	}
	factory := boshuaa.NewFactory(loggerBosh)

	// Build a UAA config from a URL.
	// HTTPS is required and certificates are always verified.
	config, err := boshuaa.NewConfigFromURL(uaaUrl)
	if err != nil {
		return nil, err
	}
	// Set client credentials for authentication.
	// Machine level access should typically use a client instead of a particular user.
	config.Client = boshDirector.ClientId
	config.ClientSecret = boshDirector.ClientSecret

	if config.Client == "" {
		config.Client = "bosh_cli"
		config.ClientSecret = ""

	}

	// Configure trusted CA certificates.
	// If nothing is provided default system certificates are used.
	config.CACert = boshDirector.CACert

	return factory.New(config)
}

func buildDirector(boshDirector BoshDirector, uaa boshuaa.UAA, loggerBosh boshlog.Logger) (boshdir.Director, error) {
	factory := boshdir.NewFactory(loggerBosh)

	// Build a Director config from address-like string.
	// HTTPS is required and certificates are always verified.
	config, err := boshdir.NewConfigFromURL(boshDirector.DirectorUrl)
	if err != nil {
		return nil, err
	}
	// Configure custom trusted CA certificates.
	// If nothing is provided default system certificates are used.
	config.CACert = boshDirector.CACert

	if uaa == nil {
		config.Client = boshDirector.Username
		config.ClientSecret = boshDirector.Password
		return factory.New(config, boshdir.NewNoopTaskReporter(), boshdir.NewNoopFileReporter())
	}
	// Allow Director to fetch UAA tokens when necessary.
	var token boshuaa.StaleAccessToken

	if boshDirector.Username != "" {
		token, err = uaa.OwnerPasswordCredentialsGrant([]boshuaa.PromptAnswer{
			{
				Key:   "username",
				Value: boshDirector.Username,
			},
			{
				Key:   "password",
				Value: boshDirector.Password,
			},
		})
		if err != nil {
			return nil, err
		}
	} else {
		token = uaa.NewStaleAccessToken(boshDirector.RefreshToken)
	}
	config.TokenFunc = boshuaa.NewAccessTokenSession(token).TokenFunc
	uaa.ClientCredentialsGrant()

	return factory.New(config, boshdir.NewNoopTaskReporter(), boshdir.NewNoopFileReporter())
}
