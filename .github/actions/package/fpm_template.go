package main

import (
	"os"
	"path/filepath"
	"strings"
	"text/template"
)

type NfpmInput struct {
	Name        string
	Arch        string
	Version     string
	Maintainer  string
	Homepage    string
	License     string
	Depends     []string
	Binary      string
	Preinstall  string
	Postinstall string
	Preremove   string
	Postremove  string

	ConfigFiles []*ConfigFile
}

type ConfigFile struct {
	LocalPath string
	DestPath  string
}

func findConfigs(dir string) []*ConfigFile {
	var configs []*ConfigFile

	wd, err := os.Getwd()
	if err != nil {
		return []*ConfigFile{}
	}

	fullDir := filepath.Join(wd, dir)

	err = filepath.Walk(fullDir,
		func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			if !info.IsDir() {
				relativePath := path[len(fullDir):]
				cf := &ConfigFile{
					LocalPath: path,
					DestPath:  relativePath,
				}
				configs = append(configs, cf)
			}
			return nil
		})
	if err != nil {
		return []*ConfigFile{}
	}

	return configs
}

func main() {
	inputName := os.Getenv("INPUT_NAME")
	inputArch := os.Getenv("INPUT_ARCH")
	inputVersion := os.Getenv("INPUT_VERSION")
	inputMaintainer := os.Getenv("INPUT_MAINTAINER")
	inputHomepage := os.Getenv("INPUT_HOMEPAGE")
	inputLicense := os.Getenv("INPUT_LICENSE")
	inputDepends := os.Getenv("INPUT_DEPENDS")
	inputBinary := os.Getenv("INPUT_BINARY")
	inputConfigDir := os.Getenv("INPUT_CONFIG_DIR")
	inputPreinstall := os.Getenv("INPUT_PREINSTALL")
	inputPostinstall := os.Getenv("INPUT_POSTINSTALL")
	inputPreremove := os.Getenv("INPUT_PREREMOVE")
	inputPostremove := os.Getenv("INPUT_POSTREMOVE")

	depends := strings.Split(inputDepends, ",")

	input := &NfpmInput{
		Name:        inputName,
		Arch:        inputArch,
		Version:     inputVersion,
		Maintainer:  inputMaintainer,
		Homepage:    inputHomepage,
		License:     inputLicense,
		Depends:     depends,
		Binary:      inputBinary,
		Preinstall:  inputPreinstall,
		Postinstall: inputPostinstall,
		Preremove:   inputPreremove,
		Postremove:  inputPostremove,
	}

	input.ConfigFiles = findConfigs(inputConfigDir)

	var t *template.Template
	t = template.Must(template.New("nfpm").Parse(nfpmTemplate))

	t.Execute(os.Stdout, input)
}

const nfpmTemplate = `
name: {{ .Name }}
arch: {{ .Arch }}
platform: linux
version: {{ .Version }}
maintainer: {{ .Maintainer }}
homepage: {{ .Homepage }}
license: {{ .License }}
depends:
{{- with .Depends }}
{{- range $index, $element := . }}
  - {{ . }}
{{- end }}
{{- end }}
contents:
  - src: {{ .Binary }}
    dst: /usr/bin/{{ .Name }}
{{- with .ConfigFiles }}
{{- range $index, $element := . }}
  - src: {{ .LocalPath }}
    dst: {{ .DestPath }}
    type: config|noreplace
{{- end }}
{{- end }}
scripts:
  preinstall: {{ .Preinstall }}
  postinstall: {{ .Postinstall }}
  preremove: {{ .Preremove }}
  postremove: {{ .Postremove }}
`
