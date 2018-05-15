package cmd

import (
	"github.com/ArthurHlt/yint"
	"io/ioutil"
	"strings"
	"fmt"
	"github.com/orange-cloudfoundry/bopt/brd"
	"gopkg.in/yaml.v2"
	"sync"
	"os"
	"io"
	"archive/zip"
	"github.com/vbauerster/mpb"
	"github.com/vbauerster/mpb/decor"
	"time"
	"github.com/go-errors/errors"
)

const defaultOpsFile = "local-release.yml"

type Manifest struct {
	Bytes []byte
}

func (a *Manifest) UnmarshalFlag(data string) error {
	if len(data) == 0 {
		return errors.New("Expected file path to be non-empty")
	}

	if data == "-" {
		bs, err := ioutil.ReadAll(os.Stdin)
		if err != nil {
			return fmt.Errorf("Reading from stdin: %s", err.Error())
		}

		a.Bytes = bs

		return nil
	}

	absPath, err := expandPath(data)
	if err != nil {
		return fmt.Errorf("Getting absolute path '%s': %s", data, err.Error())
	}

	bytes, err := ioutil.ReadFile(absPath)
	if err != nil {
		return err
	}

	a.Bytes = bytes

	return nil
}

type Job struct {
	release brd.Release
	waitBar *mpb.Bar
}

type ManifestArg struct {
	Manifest Manifest `positional-arg-name:"PATH" description:"Path to a template which could be interpolated (use - to load manifest from stdin)"`
}

type BrdCommand struct {
	ManifestArg     ManifestArg `positional-args:"true" required:"1"`
	VarKVs          []string    `long:"var"             short:"v"       value-name:"VAR=VALUE"     description:"Set variable"`
	VarFiles        []string    `long:"var-file"                        value-name:"VAR=PATH"      description:"Set variable to file contents"`
	VarsFiles       []string    `long:"vars-file"       short:"l"       value-name:"PATH"          description:"Load variables from a YAML file"`
	Output          string      `long:"output"                          value-name:"OUTPUT"        description:"Place zip file to this path (use - to write to stdout)"`
	VarsEnvs        []string    `long:"vars-env"                        value-name:"PREFIX"        description:"Load variables from environment variables (e.g.: 'MY' to load MY_var=value)"`
	OpsFiles        []string    `long:"ops-file"        short:"o"       value-name:"PATH"          description:"Load manifest operations from a YAML file"`
	Parallel        int         `long:"parallel"        short:"p"       value-name:"PARALLEL"      description:"Concurrent download at same time"`
	Path            string      `long:"path"                            value-name:"OP-PATH"       description:"Extract value out of template (e.g.: /private_key)"`
	VarErrors       bool        `long:"var-errs"                                                   description:"Expect all variables to be found, otherwise error"`
	SkipInsecure    bool        `long:"skip-insecure"   short:"k"                                  description:"Skip insecure ssl"`
	VarErrorsUnused bool        `long:"var-errs-unused"                                            description:"Expect all variables to be used, otherwise error"`
	errChan         chan error
}

var brdCommand BrdCommand

func (c *BrdCommand) Execute(_ []string) error {
	if c.Parallel <= 0 {
		c.Parallel = 3
	}
	c.errChan = make(chan error, 1)
	varsKv, err := convertVarsKV(c.VarKVs)
	if err != nil {
		return err
	}
	b := c.ManifestArg.Manifest.Bytes
	if len(c.OpsFiles) > 0 {
		b, err = yint.Apply(yint.ApplyOpts{
			YamlContent:     b,
			OpsFiles:        c.OpsFiles,
			VarErrorsUnused: c.VarErrorsUnused,
			VarErrors:       c.VarErrors,
			OpPath:          c.Path,
			VarsEnv:         c.VarsEnvs,
			VarsFiles:       c.VarsFiles,
			VarFiles:        c.VarFiles,
			VarsKV:          varsKv,
		})
	}

	ymlReleases := struct {
		Releases []brd.Release `yaml:"releases"`
	}{}
	err = yaml.Unmarshal(b, &ymlReleases)
	if err != nil {
		return err
	}

	releases := ymlReleases.Releases
	f, err := c.createFile(b, releases)
	if err != nil {
		return err
	}
	defer f.Close()
	err = c.runDownload(f, releases)
	if err != nil {
		return err
	}
	fmt.Fprintf(LogWriter, "Done: zip file writen to %s", f.Name())
	return nil
}

func (c *BrdCommand) createFile(manByte []byte, releases []brd.Release) (*os.File, error) {
	if c.Output != "" {
		if c.Output == "-" {
			return os.Stdout, nil
		}
		path, err := expandPath(c.Output)
		if err != nil {
			return nil, err
		}
		return os.Create(path)
	}
	manInfo := struct {
		Name            string `yaml:"name"`
		ManifestVersion string `yaml:"manifest_version"`
	}{}

	err := yaml.Unmarshal(manByte, &manInfo)
	if err != nil {
		return nil, err
	}

	if manInfo.ManifestVersion == "" {
		for _, rel := range releases {
			if rel.Name == manInfo.Name {
				manInfo.ManifestVersion = rel.Version
				break
			}
		}
	}

	return os.Create(fmt.Sprintf("%s-%s.zip", manInfo.Name, manInfo.ManifestVersion))
}

func (c *BrdCommand) worker(brdDl *brd.Manager, p *mpb.Progress, jobs chan Job, opDefChan chan brd.OpDefinition) {
	for job := range jobs {
		rel := job.release
		hooks := func(size int64) (brd.Hook, brd.Hook) {
			nameDl := fmt.Sprintf("Downloading %s", rel.Name)
			dlBar := p.AddBar(size, mpb.BarReplaceOnComplete(job.waitBar), mpb.BarRemoveOnComplete(), mpb.PrependDecorators(
				decor.StaticName(nameDl, len(nameDl)+1, decor.DidentRight),
				decor.CountersKibiByte("%6.1f / %6.1f", 0, decor.DwidthSync),
			), mpb.AppendDecorators(decor.Percentage(5, 0)))
			job.waitBar.IncrBy(int(job.waitBar.Total()))

			nameCompress := fmt.Sprintf("Compressing %s", rel.Name)
			compressBar := p.AddBar(size, mpb.BarReplaceOnComplete(dlBar), mpb.BarClearOnComplete(), mpb.PrependDecorators(
				decor.StaticName(nameCompress, len(nameCompress)+1, decor.DidentRight),
				decor.OnComplete(decor.CountersKibiByte("%6.1f / %6.1f", 0, decor.DwidthSync), "done!", 0, decor.DSyncSpaceR),
			), mpb.AppendDecorators(decor.Percentage(5, 0)))
			return func(r io.Reader) io.Reader { return dlBar.ProxyReader(r) }, func(r io.Reader) io.Reader { return compressBar.ProxyReader(r) }
		}

		opDef, err := brdDl.DownloadAndCompress(rel, hooks)
		if err != nil {
			c.errChan <- fmt.Errorf("Errors on release '%s': %s", rel.Name, err.Error())
			break
		}
		opDefChan <- opDef
	}

}

func (c *BrdCommand) runDownload(writer io.Writer, releases []brd.Release) error {
	zipWriter := zip.NewWriter(writer)
	defer zipWriter.Close()
	opDefChan := make(chan brd.OpDefinition, 10)
	jobsChan := make(chan Job, 10)
	mux := &sync.Mutex{}

	brdDl, err := brd.NewManager(zipWriter, mux, brd.SkipInsecure(c.SkipInsecure))
	if err != nil {
		return err
	}

	var wg sync.WaitGroup
	wg.Add(len(releases))
	p := mpb.New(mpb.WithWidth(64), mpb.WithWaitGroup(&wg), mpb.WithOutput(LogWriter))

	for w := 1; w <= c.Parallel; w++ {
		go c.worker(brdDl, p, jobsChan, opDefChan)
	}

	jobs := make([]Job, 0)

	for _, rel := range releases {
		nameWait := "Waiting " + rel.Name
		waitBar := p.AddBar(1, mpb.BarRemoveOnComplete(), mpb.PrependDecorators(
			decor.StaticName(nameWait, len(nameWait)+1, decor.DidentRight),
		))
		jobs = append(jobs, Job{
			release: rel,
			waitBar: waitBar,
		})
	}

	for _, job := range jobs {
		jobsChan <- job
	}
	close(jobsChan)

	opDefs := make([]brd.OpDefinition, 0)
	for i := 0; i < len(releases); i++ {
		select {
		case opDef := <-opDefChan:
			wg.Done()
			opDefs = append(opDefs, opDef)
		case err = <-c.errChan:
			if err != nil {
				return err
			}
		}
	}

	wg.Wait()

	opDefsYml, err := yaml.Marshal(opDefs)
	if err != nil {
		return err
	}

	fh := &zip.FileHeader{
		Name:               defaultOpsFile,
		Method:             zip.Deflate,
		UncompressedSize64: uint64(len(opDefsYml)),
		Modified:           time.Now(),
	}
	fh.SetMode(0666)
	if fh.UncompressedSize64 > (1<<32)-1 {
		fh.UncompressedSize = (1 << 32) - 1
	} else {
		fh.UncompressedSize = uint32(fh.UncompressedSize64)
	}

	opsWriter, err := zipWriter.CreateHeader(fh)
	if err != nil {
		return err
	}

	_, err = opsWriter.Write(opDefsYml)

	return err
}

func convertVarsKV(kvRawSlice []string) (map[string]interface{}, error) {
	varsKv := make(map[string]interface{})

	for _, kvRaw := range kvRawSlice {
		pieces := strings.SplitN(kvRaw, "=", 2)
		if len(pieces) != 2 {
			return varsKv, fmt.Errorf("Expected var '%s' to be in format 'name=path'", kvRaw)
		}
		if len(pieces[0]) == 0 {
			return varsKv, fmt.Errorf("Expected var '%s' to specify non-empty name", kvRaw)
		}

		if len(pieces[1]) == 0 {
			return varsKv, fmt.Errorf("Expected var '%s' to specify non-empty path", kvRaw)
		}
		varsKv[pieces[0]] = pieces[1]
	}
	return varsKv, nil
}

func init() {
	desc := fmt.Sprintf(`Download all releases from a manifest, package them and add an ops-file in order to use local-file as releases.
This will package all of this (including ops-file) in a zip.
Decompress zip and use %s to patch your existing manifest in order to use downloaded release.`, defaultOpsFile)
	parser.AddCommand(
		"brd",
		"Download all releases from a manifest, package them and add an ops-file in order to use local-file as releases",
		desc,
		&brdCommand)
}
