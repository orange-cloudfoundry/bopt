package brd

import (
	"io"
	"archive/zip"
	"net/http"
	"time"
	"net"
	"crypto/tls"
	"sync"
	"github.com/go-errors/errors"
	"path/filepath"
	"fmt"
	"io/ioutil"
	"mime"
	"os"
	"crypto/sha1"
	"strings"
	"net/url"
)

type Option func(*Manager)

type Release struct {
	URL     string `yaml:"url"`
	SHA1    string `yaml:"sha1"`
	Name    string `yaml:"name"`
	Version string `yaml:"version"`
}

type Releases []Release

func (r Releases) String() string {
	names := make([]string, len(r))
	for i, rel := range r {
		names[i] = rel.Name
	}
	return strings.Join(names, ", ")
}

type OpDefinition struct {
	Type  string      `yaml:"type"`
	Path  string      `yaml:"path"`
	Value interface{} `yaml:"value"`
}

type Zipper struct {
	zipWriter *zip.Writer
	mux       *sync.Mutex
}
type Hook func(reader io.Reader) io.Reader

type Manager struct {
	zipWriter    *zip.Writer
	mux          *sync.Mutex
	client       *http.Client
	skipInsecure bool
}

var DirVarName string = "brd_repo_dir"

func NewManager(zipWriter *zip.Writer, mux *sync.Mutex, options ...Option) (*Manager, error) {
	brd := &Manager{
		zipWriter: zipWriter,
		mux:       mux,
	}
	for _, opt := range options {
		opt(brd)
	}
	brd.client = &http.Client{
		Transport: &http.Transport{
			Proxy: http.ProxyFromEnvironment,
			DialContext: (&net.Dialer{
				Timeout:   30 * time.Second,
				KeepAlive: 30 * time.Second,
				DualStack: true,
			}).DialContext,
			MaxIdleConns:          100,
			IdleConnTimeout:       90 * time.Second,
			TLSHandshakeTimeout:   10 * time.Second,
			ExpectContinueTimeout: 1 * time.Second,
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: brd.skipInsecure,
			},
		},
	}

	return brd, nil
}

func (b Manager) DownloadAndCompress(release Release, hooks func(size int64) (hookDownload Hook, hookCompress Hook)) (OpDefinition, error) {
	if !strings.HasPrefix(release.URL, "http") {
		return OpDefinition{}, nil
	}
	ft, err := ioutil.TempFile("", DirVarName)
	if err != nil {
		return OpDefinition{}, err
	}
	defer func() {
		ft.Close()
		os.Remove(ft.Name())
	}()

	resp, err := b.client.Get(release.URL)
	if err != nil {
		return OpDefinition{}, err
	}
	defer resp.Body.Close()

	if resp.StatusCode > 302 {
		err = fmt.Errorf("non valid status: %s", resp.Status)
		return OpDefinition{}, err
	}

	parsedUrl, _ := url.Parse(release.URL)

	filename := filepath.Base(parsedUrl.Path)
	_, params, err := mime.ParseMediaType(resp.Header.Get("Content-Disposition"))
	if err == nil {
		if _, ok := params["filename"]; ok {
			filename = params["filename"]
		}
	}
	if release.Name != "" && release.Version != "" {
		ext := filepath.Ext(filename)
		if ext == "" {
			ext = ".tgz"
		} else if ext == ".gz" && filepath.Ext(strings.TrimSuffix(filename, ext)) == ".tar" {
			ext = filepath.Ext(strings.TrimSuffix(filename, ext)) + ext
		}

		filename = fmt.Sprintf("%s-%s%s", release.Name, release.Version, ext)
	}

	size := resp.ContentLength

	hookDownload, hookCompress := hooks(size)

	var reader io.Reader
	reader = resp.Body
	if hookDownload != nil {
		reader = hookDownload(resp.Body)
	}

	_, err = io.Copy(ft, reader)
	if err != nil {
		return OpDefinition{}, err
	}

	err = checkSha1(ft, release.SHA1)
	if err != nil {
		return OpDefinition{}, err
	}

	_, err = ft.Seek(0, 0)
	if err != nil {
		return OpDefinition{}, err
	}

	b.mux.Lock()
	defer b.mux.Unlock()
	info, err := ft.Stat()

	if err != nil {
		return OpDefinition{}, err
	}

	header, err := zip.FileInfoHeader(info)
	if err != nil {
		return OpDefinition{}, err
	}
	header.Name = filename
	header.Method = zip.Deflate

	writer, err := b.zipWriter.CreateHeader(header)
	if err != nil {
		return OpDefinition{}, err
	}

	reader = ft
	if hookCompress != nil {
		reader = hookCompress(ft)
	}

	_, err = io.Copy(writer, reader)
	if err != nil {
		return OpDefinition{}, err
	}

	value := struct {
		Name    string `yaml:"name"`
		Version string `yaml:"version,omitempty"`
		Url     string `yaml:"url"`
	}{
		Name:    release.Name,
		Version: release.Version,
		Url:     fmt.Sprintf("file://((%s))/%s", DirVarName, filename),
	}

	return OpDefinition{
		Path:  fmt.Sprintf("/releases/%s", release.Name),
		Type:  "replace",
		Value: value,
	}, nil
}

func checkSha1(ft *os.File, fileSHA1 string) error {
	if fileSHA1 == "" {
		return nil
	}
	ft.Seek(0, 0)

	h := sha1.New()

	_, err := io.Copy(h, ft)
	if err != nil {
		return err
	}

	bs := h.Sum(nil)

	if fileSHA1 != fmt.Sprintf("%x", bs) {
		return errors.New("SHA1 mismatch from download")
	}
	return nil
}

func SkipInsecure(skipInsecure bool) Option {
	return func(b *Manager) {
		b.skipInsecure = skipInsecure
	}
}
