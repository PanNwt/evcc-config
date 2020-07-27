package main

import (
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"text/template"

	"gopkg.in/yaml.v2"

	"github.com/andig/evcc-config/registry"
	"github.com/andig/evcc/meter"
	"github.com/andig/evcc/util"
	flag "github.com/spf13/pflag"
)

const (
	ext     = ".yaml"
	summary = "template.md"
)

var (
	confYaml       string
	confGo         bool
	confOutGo      string
	confSummary    bool
	confOutSummary string
	confHelp       bool
	tmpl           *template.Template
)

func init() {
	flag.StringVarP(&confYaml, "yaml", "y", "yaml", "yaml path")
	flag.StringVarP(&confOutGo, "output-go", "o", "", "output go files path")
	flag.StringVarP(&confOutSummary, "output-summary", "f", "", "output summary file")
	flag.BoolVarP(&confGo, "go", "g", false, "generate go files")
	flag.BoolVarP(&confSummary, "summary", "s", false, "generate summary")
	flag.BoolVarP(&confHelp, "help", "h", false, "help")
	flag.Parse()
}

var sourceTemplate = `package templates {{/* Define backtick variable */}}{{$tick := "` + "`" + `"}}

import (
	"github.com/andig/evcc-config/registry"
)

func init() {
	template := registry.Template{
		Class:       "{{.Class}}",
		Type:        "{{.Type}}",
		Name:        "{{.Name}}",
		Sample:      {{$tick}}{{escape .Sample}}{{$tick}},
	}

	registry.Registry.Add(template)
}
`

func scanFolder(root string) []string {
	var files []string
	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if filepath.Ext(info.Name()) == ext {
			files = append(files, path)
		}

		return nil
	})

	if err != nil {
		panic(err)
	}

	return files
}

func configureMeter(sample registry.Template) {
	var conf map[string]interface{}
	if err := yaml.Unmarshal([]byte(sample.Sample), &conf); err != nil {
		panic(err)
	}

	log := util.NewLogger("foo")
	// log.SetLogThreshold(jww.Th)

	meter.NewConfigurableFromConfig(log, conf)
}

func parseSample(file string) registry.Template {
	src, err := ioutil.ReadFile(file)
	if err != nil {
		panic(err)
	}

	var sample registry.Template
	if err := yaml.Unmarshal(src, &sample); err != nil {
		panic(err)
	}

	// trim trailing linebreaks
	sample.Sample = strings.TrimRight(sample.Sample, "\r\n")

	return sample
}

func render(wr io.Writer, sample registry.Template) {
	if tmpl == nil {
		var err error
		tmpl, err = template.New("test").Funcs(template.FuncMap{
			// escape backticks in raw strings
			"escape": func(s string) string {
				return strings.ReplaceAll(s, "`", "`+\"`\"+`")
			},
		}).Parse(sourceTemplate)

		if err != nil {
			panic(err)
		}
	}

	tmpl.Execute(wr, sample)
}

func renderSummary(wr io.Writer, samples []registry.Template) {
	summaryTemplate, err := ioutil.ReadFile(summary)
	if err != nil {
		panic(err)
	}

	tmpl, err := template.New("test").Funcs(template.FuncMap{
		// filter samples by class
		"filter": func(class string, samples []registry.Template) (reg []registry.Template) {
			for _, sample := range samples {
				if sample.Class == class {
					reg = append(reg, sample)
				}
			}
			return
		},
		// https://github.com/Masterminds/sprig/blob/48e6b77026913419ba1a4694dde186dc9c4ad74d/strings.go#L109
		"indent": func(spaces int, v string) string {
			pad := strings.Repeat(" ", spaces)
			return pad + strings.Replace(v, "\n", "\n"+pad, -1)
		},
	}).Parse(string(summaryTemplate))

	if err != nil {
		panic(err)
	}

	tmpl.Execute(wr, samples)
}

func output(file string, fun func(io.Writer)) {
	wr := os.Stdout
	if file != "" {
		var err error
		wr, err = os.Create(file)
		if err != nil {
			panic(err)
		}
	}

	fun(wr)
	wr.Close()
}

func main() {
	if confHelp {
		flag.PrintDefaults()
		os.Exit(0)
	}

	var samples []registry.Template

	files := scanFolder(confYaml)
	for _, file := range files {
		sample := parseSample(file)

		// example type
		dir := filepath.Dir(file)
		typ := filepath.Base(dir)
		typ = strings.TrimRight(typ, "s") // de-pluralize

		sample.Class = typ
		samples = append(samples, sample)

		if confGo {
			var out string
			if confOutGo != "" {
				name := strings.TrimSuffix(filepath.Base(file), filepath.Ext(file))
				out = fmt.Sprintf("%s/%s-%s.go", confOutGo, typ, name)
			}

			println(out)

			output(out, func(wr io.Writer) {
				render(wr, sample)
			})
		}
	}

	if confSummary {
		sort.Sort(registry.Templates(samples))
		output(confOutSummary, func(wr io.Writer) {
			renderSummary(wr, samples)
		})
	}
}
