// Copyright 2020 MongoDB Inc
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package config

import (
	"fmt"
	"os"
	"path"
	"runtime"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v4"
	"github.com/mongodb/atlas-cli-plugin-kubernetes/internal/version"
	"github.com/spf13/afero"
	"github.com/spf13/viper"
	"go.mongodb.org/atlas/auth"
)

const (
	DefaultProfile           = "default"  // DefaultProfile default
	CloudService             = "cloud"    // CloudService setting when using Atlas API
	CloudGovService          = "cloudgov" // CloudGovService setting when using Atlas API for Government
	projectID                = "project_id"
	orgID                    = "org_id"
	service                  = "service"
	publicAPIKey             = "public_api_key"
	privateAPIKey            = "private_api_key"
	AccessTokenField         = "access_token"
	RefreshTokenField        = "refresh_token"
	OpsManagerURLField       = "ops_manager_url"
	output                   = "output"
	AtlasCLI                 = "atlascli"
	ContainerizedHostNameEnv = "MONGODB_ATLAS_IS_CONTAINERIZED"
	GitHubActionsHostNameEnv = "GITHUB_ACTIONS"
	AtlasActionHostNameEnv   = "ATLAS_GITHUB_ACTION"
	CLIUserTypeEnv           = "CLI_USER_TYPE" // CLIUserTypeEnv is used to separate MongoDB University users from default users
	DefaultUser              = "default"       // Users that do NOT use ATLAS CLI with MongoDB University
	NativeHostName           = "native"
	DockerContainerHostName  = "container"
	GitHubActionsHostName    = "all_github_actions"
	AtlasActionHostName      = "atlascli_github_action"
)

var (
	HostName       = getConfigHostnameFromEnvs()
	UserAgent      = fmt.Sprintf("%s/%s (%s;%s;%s)", AtlasCLI, version.Version, runtime.GOOS, runtime.GOARCH, HostName)
	CLIUserType    = newCLIUserTypeFromEnvs()
	defaultProfile = newProfile()
)

type Profile struct {
	name      string
	configDir string
	fs        afero.Fs
	err       error
}

func IsTrue(s string) bool {
	switch s {
	case "t", "T", "true", "True", "TRUE", "y", "Y", "yes", "Yes", "YES", "1":
		return true
	default:
		return false
	}
}

func Default() *Profile {
	return defaultProfile
}

// getConfigHostnameFromEnvs patches the agent hostname based on set env vars.
func getConfigHostnameFromEnvs() string {
	var builder strings.Builder

	envVars := []struct {
		envName  string
		hostName string
	}{
		{AtlasActionHostNameEnv, AtlasActionHostName},
		{GitHubActionsHostNameEnv, GitHubActionsHostName},
		{ContainerizedHostNameEnv, DockerContainerHostName},
	}

	for _, envVar := range envVars {
		if envIsTrue(envVar.envName) {
			appendToHostName(&builder, envVar.hostName)
		} else {
			appendToHostName(&builder, "-")
		}
	}
	configHostName := builder.String()

	if isDefaultHostName(configHostName) {
		return NativeHostName
	}
	return configHostName
}

// newCLIUserTypeFromEnvs patches the user type information based on set env vars.
func newCLIUserTypeFromEnvs() string {
	if value, ok := os.LookupEnv(CLIUserTypeEnv); ok {
		return value
	}

	return DefaultUser
}

func envIsTrue(env string) bool {
	return IsTrue(os.Getenv(env))
}

func appendToHostName(builder *strings.Builder, configVal string) {
	if builder.Len() > 0 {
		builder.WriteString("|")
	}
	builder.WriteString(configVal)
}

// isDefaultHostName checks if the hostname is the default placeholder.
func isDefaultHostName(hostname string) bool {
	// Using strings.Count for a more dynamic approach.
	return strings.Count(hostname, "-") == strings.Count(hostname, "|")+1
}

func newProfile() *Profile {
	configDir, err := CLIConfigHome()
	np := &Profile{
		name:      DefaultProfile,
		configDir: configDir,
		fs:        afero.NewOsFs(),
		err:       err,
	}
	return np
}

func Name() string { return Default().Name() }
func (p *Profile) Name() string {
	return p.name
}

func SetGlobal(name string, value any) { viper.Set(name, value) }
func (*Profile) SetGlobal(name string, value any) {
	SetGlobal(name, value)
}

func Get(name string) any { return Default().Get(name) }
func (p *Profile) Get(name string) any {
	if viper.IsSet(name) && viper.Get(name) != "" {
		return viper.Get(name)
	}
	settings := viper.GetStringMap(p.Name())
	return settings[name]
}

func GetString(name string) string { return Default().GetString(name) }
func (p *Profile) GetString(name string) string {
	value := p.Get(name)
	if value == nil {
		return ""
	}
	return value.(string)
}

// Service get configured service.
func Service() string { return Default().Service() }
func (p *Profile) Service() string {
	if viper.IsSet(service) {
		return viper.GetString(service)
	}

	settings := viper.GetStringMapString(p.Name())
	return settings[service]
}

// PublicAPIKey get configured public api key.
func PublicAPIKey() string { return Default().PublicAPIKey() }
func (p *Profile) PublicAPIKey() string {
	return p.GetString(publicAPIKey)
}

// PrivateAPIKey get configured private api key.
func PrivateAPIKey() string { return Default().PrivateAPIKey() }
func (p *Profile) PrivateAPIKey() string {
	return p.GetString(privateAPIKey)
}

// AccessToken get configured access token.
func AccessToken() string { return Default().AccessToken() }
func (p *Profile) AccessToken() string {
	return p.GetString(AccessTokenField)
}

// RefreshToken get configured refresh token.
func RefreshToken() string { return Default().RefreshToken() }
func (p *Profile) RefreshToken() string {
	return p.GetString(RefreshTokenField)
}

type AuthMechanism int

const (
	APIKeys AuthMechanism = iota

	OAuth
	NotLoggedIn
)

// AuthType returns the type of authentication used in the profile.
func AuthType() AuthMechanism { return Default().AuthType() }
func (p *Profile) AuthType() AuthMechanism {
	if p.PublicAPIKey() != "" && p.PrivateAPIKey() != "" {
		return APIKeys
	}
	if p.AccessToken() != "" {
		return OAuth
	}
	return NotLoggedIn
}

// Token gets configured auth.Token.
func Token() (*auth.Token, error) { return Default().Token() }
func (p *Profile) Token() (*auth.Token, error) {
	if p.AccessToken() == "" || p.RefreshToken() == "" {
		return nil, nil
	}
	c, err := p.tokenClaims()
	if err != nil {
		return nil, err
	}
	var e time.Time
	if c.ExpiresAt != nil {
		e = c.ExpiresAt.Time
	}
	t := &auth.Token{
		AccessToken:  p.AccessToken(),
		RefreshToken: p.RefreshToken(),
		TokenType:    "Bearer",
		Expiry:       e,
	}
	return t, nil
}

func (p *Profile) tokenClaims() (jwt.RegisteredClaims, error) {
	c := jwt.RegisteredClaims{}
	// ParseUnverified is ok here, only want to make sure is a JWT and to get the claims for a Subject
	_, _, err := new(jwt.Parser).ParseUnverified(p.AccessToken(), &c)
	return c, err
}

// OpsManagerURL get configured ops manager base url.
func OpsManagerURL() string { return Default().OpsManagerURL() }
func (p *Profile) OpsManagerURL() string {
	return p.GetString(OpsManagerURLField)
}

// ProjectID get configured project ID.
func ProjectID() string { return Default().ProjectID() }
func (p *Profile) ProjectID() string {
	return p.GetString(projectID)
}

// OrgID get configured organization ID.
func OrgID() string { return Default().OrgID() }
func (p *Profile) OrgID() string {
	return p.GetString(orgID)
}

// Output get configured output format.
func Output() string { return Default().Output() }
func (p *Profile) Output() string {
	return p.GetString(output)
}

// CLIConfigHome retrieves configHome path.
func CLIConfigHome() (string, error) {
	home, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}

	return path.Join(home, "atlascli"), nil
}
