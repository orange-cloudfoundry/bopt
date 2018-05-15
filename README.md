# Bopt

Bopt is the short name for bosh operator tools.

It provides some useful tool to exploit bosh, exemple:

- `brd` -- *Bosh Release Downloader* command: Download all releases from a bosh manifest in a single zip
 and add an ops-file to be able to use offline release after downloading locally. (Useful when no access to internet to 
 deposit in an accessible repository before deployment)
- `cmder` -- *Commander* command: Run a set of commands on multiple vms found by deployments, jobs name and bosh directors.


## Installation

### On *nix system

You can install this via the command-line with either `curl` or `wget`.

#### via curl

```bash
$ sh -c "$(curl -fsSL https://raw.github.com/orange-cloudfoundry/bopt/master/bin/install.sh)"
```

#### via wget

```bash
$ sh -c "$(wget https://raw.github.com/orange-cloudfoundry/bopt/master/bin/install.sh -O -)"
```

### On windows

You can install it by downloading the `.exe` corresponding to your cpu from releases page: https://github.com/orange-cloudfoundry/bopt/releases .
Alternatively, if you have terminal interpreting shell you can also use command line script above, it will download file in your current working dir.

### From go command line

Simply run in terminal:

```bash
$ go get github.com/orange-cloudfoundry/bopt
```

## Usage

```
bopt: Usage:
  bopt [OPTIONS] <brd | cmder>

Application Options:
  -v, --verbose  Verbose output

Help Options:
  -h, --help     Show this help message

Available commands:
  brd    Download all releases from a manifest, package them and add an ops-file in order to use local-file as releases
  cmder  Run a set of commands on multiple vms found by deployments, jobs name and bosh directors
```

### BRD

```
bopt: Usage:
  bopt [OPTIONS] brd [brd-OPTIONS] PATH

Download all releases from a manifest, package them and add an ops-file in order to use local-file as releases.
This will package all of this (including ops-file) in a zip.
Decompress zip and use local-release.yml to patch your existing manifest in order to use downloaded release.

Application Options:
  -v, --verbose                  Verbose output

Help Options:
  -h, --help                     Show this help message

[brd command options]
      -v, --var=VAR=VALUE        Set variable
          --var-file=VAR=PATH    Set variable to file contents
      -l, --vars-file=PATH       Load variables from a YAML file
          --output=OUTPUT        Place zip file to this path (use - to write to stdout)
          --vars-env=PREFIX      Load variables from environment variables (e.g.: 'MY' to load MY_var=value)
      -o, --ops-file=PATH        Load manifest operations from a YAML file
      -p, --parallel=PARALLEL    Concurrent download at same time
          --path=OP-PATH         Extract value out of template (e.g.: /private_key)
          --var-errs             Expect all variables to be found, otherwise error
      -k, --skip-insecure        Skip insecure ssl
          --var-errs-unused      Expect all variables to be used, otherwise error

[brd command arguments]
  PATH:                          Path to a template which could be interpolated (use - to load manifest from stdin)
```

### Cmder

```
bopt: Usage:
  bopt [OPTIONS] cmder [cmder-OPTIONS]

Run a set of commands on multiple vms found by deployments, jobs name and bosh directors

Application Options:
  -v, --verbose                      Verbose output

Help Options:
  -h, --help                         Show this help message

[cmder command options]
      -j, --job-match=JOB_MATCH      Job to target
      -d, --deployment=DEPLOYMENT    If set it will looking only deployments which match regex given
      -n, --non-privileged           Run scripts not in privileged mode
      -s, --script=SCRIPT            Scripts to run
      -a, --after-all=               Det of commands to run after all commands in script have been ran in all vms
          --config=                  Config file path (default: ~/.bosh/config) [$BOSH_CONFIG]
      -f, --file=PATH                Path to a script in yml format (use - to load file from stdin)
      -e, --environment=             Director environment name or URL
      -u, --username=                Username to use to connect to director
      -p, --password=                Password to use to connect to director
          --gw-disable               Disable usage of gateway connection [$BOSH_GW_DISABLE]
          --gw-user=                 Username for gateway connection [$BOSH_GW_USER]
          --gw-host=                 Host for gateway connection [$BOSH_GW_HOST]
          --gw-private-key=          Private key path for gateway connection [$BOSH_GW_PRIVATE_KEY]
          --store=PATH               Store script in yml format at a path (- write it to stdout)
```