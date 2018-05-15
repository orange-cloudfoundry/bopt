package commander

import (
	"fmt"
	boshdir "github.com/cloudfoundry/bosh-cli/director"
	"io/ioutil"
	"regexp"
)

type Environment struct {
	URL    string `yaml:"url"`
	CACert string `yaml:"ca_cert,omitempty"`

	Alias string `yaml:"alias,omitempty"`

	// Auth
	Username     string `yaml:"username,omitempty"`
	Password     string `yaml:"password,omitempty"`
	RefreshToken string `yaml:"refresh_token,omitempty"`
}

func (e Environment) ToDirector() BoshDirector {
	return BoshDirector{
		Name:         e.Alias,
		DirectorUrl:  e.URL,
		CACert:       e.CACert,
		Username:     e.Username,
		Password:     e.Password,
		RefreshToken: e.RefreshToken,
	}
}

type Environments []Environment

func (envs Environments) FindByNameOrUrl(ident string) (Environment, error) {
	for _, env := range envs {
		if ident == env.Alias || ident == env.URL {
			return env, nil
		}
	}
	return Environment{}, fmt.Errorf("Environment '%s' not found", ident)
}

type Config struct {
	Environments Environments `yaml:"environments"`
}

func (c *Config) UnmarshalYAML(unmarshal func(interface{}) error) error {
	type plain Config
	if err := unmarshal((*plain)(c)); err != nil {
		return err
	}
	return nil
}

type BoshDirector struct {
	Name         string  `yaml:"name"`
	ClientId     string  `yaml:"client_id"`
	ClientSecret string  `yaml:"client_secret"`
	Username     string  `yaml:"username"`
	Password     string  `yaml:"password"`
	DirectorUrl  string  `yaml:"director_url"`
	CACert       string  `yaml:"ca_cert"`
	CACertFile   string  `yaml:"ca_cert_file"`
	RefreshToken string  `yaml:"refresh_token,omitempty"`
	Gateway      Gateway `yaml:"gateway"`
}

type Gateway struct {
	Username       string `yaml:"username"`
	Host           string `yaml:"host"`
	PrivateKeyPath string `yaml:"private_key_path"`
}

func (c *BoshDirector) UnmarshalYAML(unmarshal func(interface{}) error) error {
	type plain BoshDirector
	if err := unmarshal((*plain)(c)); err != nil {
		return err
	}
	if c.Name == "" {
		return fmt.Errorf("You must set an name to your bosh director.")
	}
	if c.DirectorUrl == "" {
		return fmt.Errorf("You must set the url to your director.")
	}

	return nil
}

type BoshDirectors []BoshDirector

func (boshDirs BoshDirectors) FindDirector(name string) BoshDirector {
	for _, boshDir := range boshDirs {
		if boshDir.Name == name {
			return boshDir
		}
	}
	return BoshDirector{}
}

func (boshDir *BoshDirector) LoadCaCertFile() error {
	if boshDir.CACertFile == "" {
		return nil
	}
	b, err := ioutil.ReadFile(boshDir.CACertFile)
	if err != nil {
		return fmt.Errorf("Error when loading CACert file for bosh director '%s': %s", boshDir.Name, err.Error())
	}
	if boshDir.CACert != "" {
		boshDir.CACert += "\n" + string(b)
	} else {
		boshDir.CACert = string(b)
	}
	return nil
}

type Script struct {
	JobMatch    Regexp   `yaml:"job_match"   long:"job-match"      short:"j"                 value-name:"JOB_MATCH"  description:"Job to target"`
	Deployments []Regexp `yaml:"deployments" long:"deployment"     short:"d"                 value-name:"DEPLOYMENT" description:"If set it will looking only deployments which match regex given"`
	Sudo        bool     `yaml:"sudo"        long:"non-privileged" short:"n"                                         description:"Run scripts not in privileged mode"`
	Script      []string `yaml:"script"      long:"script"         short:"s"                 value-name:"SCRIPT"     description:"Scripts to run"`
	AfterAll    []string `yaml:"after_all"   long:"after-all"      short:"a"                                         description:"Det of commands to run after all commands in script have been ran in all vms"`
}

func (c *Script) UnmarshalYAML(unmarshal func(interface{}) error) error {
	type plain Script
	if err := unmarshal((*plain)(c)); err != nil {
		return err
	}
	return c.Check()
}

func (c *Script) Check() error {
	if c.JobMatch.String() == "" {
		return fmt.Errorf("You must set a job matcher (this can be a regex).")
	}
	if len(c.Script) == 0 {
		return fmt.Errorf("You must set an user name to connect to the supervision")
	}

	return nil
}

type Regexps []Regexp

func (re Regexps) MatchString(match string) bool {
	for _, regex := range re {
		if regex.MatchString(match) {
			return true
		}
	}
	return false
}

type Regexp struct {
	*regexp.Regexp
	Raw string
}

func (re *Regexp) UnmarshalYAML(unmarshal func(interface{}) error) error {
	var s string
	if err := unmarshal(&s); err != nil {
		return err
	}

	return re.UnmarshalFlag(s)
}

func (re *Regexp) UnmarshalFlag(data string) error {
	regex, err := regexp.Compile("^(?:" + data + ")$")
	if err != nil {
		return err
	}
	re.Regexp = regex
	re.Raw = data

	return nil
}

func (re Regexp) MarshalYAML() (interface{}, error) {
	return re.Raw, nil
}

type BoshSshInstance struct {
	Deployment boshdir.Deployment
	Instance   boshdir.VMInfo
}

func (i BoshSshInstance) String() string {
	indexJob := *i.Instance.Index
	return fmt.Sprintf("instance %s/%d in deployment %s", i.Instance.JobName, indexJob, i.Deployment.Name())
}
