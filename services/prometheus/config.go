package prometheus

import (
	"fmt"
	"github.com/percona/promconfig"
	"gopkg.in/yaml.v2"
	"io/ioutil"
	"path/filepath"
	"time"
)

var (
	// DefaultConfig is the default top-level configuration.
	DefaultConfig = promconfig.Config{
		GlobalConfig: DefaultGlobalConfig,
	}

	// DefaultGlobalConfig is the default global configuration.
	DefaultGlobalConfig = promconfig.GlobalConfig{
		ScrapeInterval:     promconfig.Duration(1 * time.Minute),
		ScrapeTimeout:      promconfig.Duration(10 * time.Second),
		EvaluationInterval: promconfig.Duration(1 * time.Minute),
	}
	// DefaultScrapeConfig is the default scrape configuration.
	DefaultScrapeConfig = ScrapeConfig{
		// ScrapeTimeout and ScrapeInterval default to the
		// configured globals.
		MetricsPath: "/metrics",
		Scheme:      "http",
		HonorLabels: false,
	}
)

func LoadFile(filename string) (*promconfig.Config, error) {
	content, err := ioutil.ReadFile(filename)
	if err != nil {
		return nil, err
	}
	cfg, err := Load(string(content))
	if err != nil {
		return nil, fmt.Errorf("parsing YAML file %s: %v", filename, err)
	}
	resolveFilepaths(filepath.Dir(filename), cfg)
	return cfg, nil
}

// Load parses the YAML input s into a Config.
func Load(s string) (*promconfig.Config, error) {
	cfg := &promconfig.Config{}
	// If the entire config body is empty the UnmarshalYAML method is
	// never called. We thus have to set the DefaultConfig at the entry
	// point as well.
	*cfg = DefaultConfig

	err := yaml.Unmarshal([]byte(s), cfg)
	if err != nil {
		return nil, err
	}
	return cfg, nil
}

// resolveFilepaths joins all relative paths in a configuration
// with a given base directory.
func resolveFilepaths(baseDir string, cfg *promconfig.Config) {
	join := func(fp string) string {
		if len(fp) > 0 && !filepath.IsAbs(fp) {
			fp = filepath.Join(baseDir, fp)
		}
		return fp
	}

	for i, rf := range cfg.RuleFiles {
		cfg.RuleFiles[i] = join(rf)
	}

	clientPaths := func(scfg *promconfig.HTTPClientConfig) {
		scfg.BearerTokenFile = join(scfg.BearerTokenFile)
		scfg.TLSConfig.CAFile = join(scfg.TLSConfig.CAFile)
		scfg.TLSConfig.CertFile = join(scfg.TLSConfig.CertFile)
		scfg.TLSConfig.KeyFile = join(scfg.TLSConfig.KeyFile)
	}
	sdPaths := func(cfg *promconfig.ServiceDiscoveryConfig) {
		for _, kcfg := range cfg.KubernetesSDConfigs {
			kcfg.HTTPClientConfig = promconfig.HTTPClientConfig{
				BearerTokenFile: join(kcfg.HTTPClientConfig.BearerTokenFile),
				TLSConfig: promconfig.TLSConfig{
					CAFile:   join(kcfg.HTTPClientConfig.TLSConfig.CAFile),
					CertFile: join(kcfg.HTTPClientConfig.TLSConfig.CertFile),
					KeyFile:  join(kcfg.HTTPClientConfig.TLSConfig.KeyFile),
				},
			}
		}
		// @TODO
		//for _, mcfg := range cfg.MarathonSDConfigs {
		//	mcfg.BearerTokenFile = join(mcfg.BearerTokenFile)
		//	mcfg.TLSConfig.CAFile = join(mcfg.TLSConfig.CAFile)
		//	mcfg.TLSConfig.CertFile = join(mcfg.TLSConfig.CertFile)
		//	mcfg.TLSConfig.KeyFile = join(mcfg.TLSConfig.KeyFile)
		//}
		//for _, consulcfg := range cfg.ConsulSDConfigs {
		//	consulcfg.TLSConfig.CAFile = join(consulcfg.TLSConfig.CAFile)
		//	consulcfg.TLSConfig.CertFile = join(consulcfg.TLSConfig.CertFile)
		//	consulcfg.TLSConfig.KeyFile = join(consulcfg.TLSConfig.KeyFile)
		//}
		for _, filecfg := range cfg.FileSDConfigs {
			for i, fn := range filecfg.Files {
				filecfg.Files[i] = join(fn)
			}
		}
	}

	for _, cfg := range cfg.ScrapeConfigs {
		clientPaths(&cfg.HTTPClientConfig)
		sdPaths(&cfg.ServiceDiscoveryConfig)
	}
	for _, cfg := range cfg.AlertingConfig.AlertmanagerConfigs {
		clientPaths(&cfg.HTTPClientConfig)
		sdPaths(&cfg.ServiceDiscoveryConfig)
	}
}
