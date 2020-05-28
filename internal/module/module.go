package module

import (
	"crypto/md5"
	"encoding/base64"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"path"
	"regexp"
	"sync"

	"github.com/commitdev/zero/configs"
	"github.com/commitdev/zero/internal/config"
	"github.com/commitdev/zero/internal/util"
	"github.com/hashicorp/go-getter"
	"github.com/manifoldco/promptui"
)

// TemplateModule merges a module instance params with the static configs
type TemplateModule struct {
	config.ModuleInstance
	Config config.ModuleConfig
}

type ProgressTracking struct {
	sync.Mutex
	downloaded map[string]int
}

// Init downloads the remote template files and parses the module config yaml
func NewTemplateModule(moduleCfg config.ModuleInstance) (*TemplateModule, error) {
	var templateModule TemplateModule
	templateModule.Source = moduleCfg.Source
	templateModule.Params = moduleCfg.Params
	templateModule.Overwrite = moduleCfg.Overwrite
	templateModule.Output = moduleCfg.Output

	p := &ProgressTracking{}
	sourcePath := GetSourceDir(templateModule.Source)

	if !IsLocal(templateModule.Source) {
		err := getter.Get(sourcePath, templateModule.Source, getter.WithProgress(p))
		if err != nil {
			return nil, err
		}
	}

	configPath := path.Join(sourcePath, "zero-module.yml")
	moduleConfig := config.LoadModuleConfig(configPath)
	templateModule.Config = *moduleConfig

	return &templateModule, nil
}

func appendProjectEnvToCmdEnv(envMap map[string]string, envList []string) []string {
	for key, val := range envMap {
		if val != "" {
			envList = append(envList, fmt.Sprintf("%s=%s", key, val))
		}
	}
	return envList
}

// aws cli prints output with linebreak in them
func sanitizePromptResult(str string) string {
	re := regexp.MustCompile("\\n")
	return re.ReplaceAllString(str, "")
}

// PromptParams renders series of prompt UI based on the config
func (m *TemplateModule) PromptParams(projectContext map[string]string) error {
	for _, promptConfig := range m.Config.Prompts {

		label := promptConfig.Label
		if promptConfig.Label == "" {
			label = promptConfig.Field
		}

		// deduplicate fields already prompted and received
		if _, isAlreadySet := projectContext[promptConfig.Field]; isAlreadySet {
			continue
		}

		var err error
		var result string
		if len(promptConfig.Options) > 0 {
			prompt := promptui.Select{
				Label: label,
				Items: promptConfig.Options,
			}
			_, result, err = prompt.Run()

		} else if promptConfig.Execute != "" {
			// TODO: this could perhaps be set as a default for part of regular prompt
			cmd := exec.Command("bash", "-c", promptConfig.Execute)
			cmd.Env = appendProjectEnvToCmdEnv(projectContext, os.Environ())
			out, err := cmd.Output()

			if err != nil {
				log.Fatalf("Failed to execute  %v\n", err)
				panic(err)
			}
			result = string(out)
		} else {
			prompt := promptui.Prompt{
				Label: label,
			}
			result, err = prompt.Run()
		}
		if err != nil {
			return err
		}

		result = sanitizePromptResult(result)
		if m.Params == nil {
			m.Params = make(map[string]string)
		}
		m.Params[promptConfig.Field] = result
		projectContext[promptConfig.Field] = result
	}

	return nil
}

// GetSourcePath gets a unique local source directory name. For local modules, it use the local directory
func GetSourceDir(source string) string {
	if !IsLocal(source) {
		h := md5.New()
		io.WriteString(h, source)
		source = base64.StdEncoding.EncodeToString(h.Sum(nil))
		return path.Join(configs.TemplatesDir, source)
	} else {
		return source
	}
}

// IsLocal uses the go-getter FileDetector to check if source is a file
func IsLocal(source string) bool {
	pwd := util.GetCwd()

	// ref: https://github.com/hashicorp/go-getter/blob/master/detect_test.go
	out, err := getter.Detect(source, pwd, getter.Detectors)

	match, err := regexp.MatchString("^file://.*", out)
	if err != nil {
		log.Panicf("invalid source format %s", err)
	}

	return match
}

func withPWD(pwd string) func(*getter.Client) error {
	return func(c *getter.Client) error {
		c.Pwd = pwd
		return nil
	}
}

func (p *ProgressTracking) TrackProgress(src string, currentSize, totalSize int64, stream io.ReadCloser) (body io.ReadCloser) {
	p.Lock()
	defer p.Unlock()

	if p.downloaded == nil {
		p.downloaded = map[string]int{}
	}

	v, _ := p.downloaded[src]
	p.downloaded[src] = v + 1
	return stream
}
