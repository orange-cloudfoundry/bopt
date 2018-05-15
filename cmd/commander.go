package cmd

import (
	"github.com/orange-cloudfoundry/bopt/commander"
	"errors"
	"io/ioutil"
	"fmt"
	"os"
	"gopkg.in/yaml.v2"
	boshlog "github.com/cloudfoundry/bosh-utils/logger"
	"github.com/sirupsen/logrus"
	"log"
	"io"
)

type ScriptYml struct {
	Script *commander.Script `no-flag:"true"`
}

func (a *ScriptYml) UnmarshalFlag(data string) error {
	if len(data) == 0 {
		return nil
	}

	var b []byte
	var err error

	if data == "-" {
		b, err = ioutil.ReadAll(os.Stdin)
		if err != nil {
			return fmt.Errorf("Reading from stdin: %s", err.Error())
		}
	} else {
		absPath, err := expandPath(data)
		if err != nil {
			return fmt.Errorf("Getting absolute path '%s': %s", data, err.Error())
		}

		b, err = ioutil.ReadFile(absPath)
		if err != nil {
			return err
		}
	}

	var script commander.Script
	err = yaml.Unmarshal(b, &script)
	if err != nil {
		return err
	}

	a.Script = &script

	return nil
}

type BoshConfig struct {
	Config commander.Config
}

func (a *BoshConfig) UnmarshalFlag(data string) error {
	if len(data) == 0 {
		return errors.New("Expected file path to be non-empty")
	}

	var b []byte
	var err error

	absPath, err := expandPath(data)
	if err != nil {
		return fmt.Errorf("Getting absolute path '%s': %s", data, err.Error())
	}

	b, err = ioutil.ReadFile(absPath)
	if err != nil {
		return err
	}

	var config commander.Config
	err = yaml.Unmarshal(b, &config)
	if err != nil {
		return err
	}

	a.Config = config

	return nil
}

type ScriptYmlArg struct {
	ScriptYml ScriptYml ` description:"Path to a script in yml format (use - to load file from stdin)"`
}

type CommanderCommand struct {
	*commander.Script
	BoshConfig   BoshConfig `long:"config"                                  description:"Config file path" env:"BOSH_CONFIG" default:"~/.bosh/config"`
	ScriptYml    ScriptYml  `long:"file" short:"f"        value-name:"PATH" description:"Path to a script in yml format (use - to load file from stdin)"`
	Environments []string   `long:"environment" short:"e" required:"true"   description:"Director environment name or URL"`
	Username     string     `long:"username" short:"u"                      description:"Username to use to connect to director"`
	Password     string     `long:"password" short:"p"                      description:"Password to use to connect to director"`

	GwDisable        bool   `long:"gw-disable"                              description:"Disable usage of gateway connection" env:"BOSH_GW_DISABLE"`
	GwUsername       string `long:"gw-user"                                 description:"Username for gateway connection" env:"BOSH_GW_USER"`
	GwHost           string `long:"gw-host"                                 description:"Host for gateway connection" env:"BOSH_GW_HOST"`
	GwPrivateKeyPath string `long:"gw-private-key"                          description:"Private key path for gateway connection" env:"BOSH_GW_PRIVATE_KEY"`

	Store string `long:"store"                 value-name:"PATH" description:"Store script in yml format at a path (- write it to stdout)"`
}

var commanderCommand CommanderCommand

func (c *CommanderCommand) Execute(_ []string) error {
	script := c.Script

	if c.ScriptYml.Script != nil {
		script = c.ScriptYml.Script
	} else if script != nil {
		script.Sudo = !script.Sudo
	}

	if script == nil || len(script.Script) == 0 {
		return fmt.Errorf("You must give a script")
	}
	if script == nil || script.JobMatch.Regexp == nil {
		return fmt.Errorf("You must give a job")
	}

	err := c.storeScript(*script)
	if err != nil {
		return err
	}

	logLevel := boshlog.LevelInfo
	if logrus.GetLevel() == logrus.DebugLevel {
		logLevel = boshlog.LevelDebug
	}
	logWriter := log.New(os.Stderr, "", log.Ldate|log.Ltime)
	boshLogger := boshlog.New(logLevel, logWriter, logWriter)
	cmdRunner := commander.NewCommandRunner(script, boshLogger)

	for _, envRaw := range c.Environments {
		err := c.runOnEnv(cmdRunner, envRaw)
		if err != nil {
			logrus.WithField("environment", envRaw).Error(err.Error())
		}
	}
	return nil
}

func (c CommanderCommand) storeScript(script commander.Script) error {
	if c.Store == "" {
		return nil
	}

	var writer io.Writer = os.Stdout

	if c.Store != "-" {
		absPath, err := expandPath(c.Store)
		if err != nil {
			return fmt.Errorf("Getting absolute path '%s': %s", c.Store, err.Error())
		}
		f, err := os.Create(absPath)
		if err != nil {
			return fmt.Errorf("Error creating file '%s': %s", c.Store, err.Error())
		}
		writer = f
		defer f.Close()
	}

	b, _ := yaml.Marshal(script)
	_, err := writer.Write(b)
	if err != nil {
		return err
	}
	logrus.WithField("path", c.Store).Info("Script has been stored.")
	return nil
}

func (c CommanderCommand) runOnEnv(cmdRunner *commander.CommandRunner, envRaw string) error {
	envs := c.BoshConfig.Config.Environments

	env, err := envs.FindByNameOrUrl(envRaw)
	if err != nil {
		return err
	}

	director := env.ToDirector()
	if director.Name == "" {
		director.Name = director.DirectorUrl
	}
	if c.Username != "" {
		director.Username = c.Username
		director.Password = c.Password
	}

	if !c.GwDisable && c.GwHost != "" {
		gw := commander.Gateway{
			Username:       c.GwUsername,
			Host:           c.GwHost,
			PrivateKeyPath: c.GwPrivateKeyPath,
		}
		director.Gateway = gw
	}

	return cmdRunner.Run(director)
}

func init() {
	desc := `Run a set of commands on multiple vms found by deployments, jobs name and bosh directors`
	parser.AddCommand(
		"cmder",
		desc,
		desc,
		&commanderCommand)
}
