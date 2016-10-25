package filesystem

import (
	"fmt"

	"gopkg.in/src-d/go-git.v4/config"
	gitconfig "gopkg.in/src-d/go-git.v4/formats/config"
	"gopkg.in/src-d/go-git.v4/storage/filesystem/internal/dotgit"
)

const (
	remoteSection = "remote"
	fetchKey      = "fetch"
	urlKey        = "url"
)

type ConfigStorage struct {
	dir *dotgit.DotGit
}

func (c *ConfigStorage) Remote(name string) (*config.RemoteConfig, error) {
	cfg, err := c.read()
	if err != nil {
		return nil, err
	}

	s := cfg.Section(remoteSection).Subsection(name)
	if s == nil {
		return nil, config.ErrRemoteConfigNotFound
	}

	return parseRemote(s), nil
}

func (c *ConfigStorage) Remotes() ([]*config.RemoteConfig, error) {
	cfg, err := c.read()
	if err != nil {
		return nil, err
	}

	remotes := []*config.RemoteConfig{}
	sect := cfg.Section(remoteSection)
	for _, s := range sect.Subsections {
		remotes = append(remotes, parseRemote(s))
	}

	return remotes, nil
}

func (c *ConfigStorage) SetRemote(r *config.RemoteConfig) error {
	return nil
	return fmt.Errorf("set remote - not implemented yet")
}

func (c *ConfigStorage) DeleteRemote(name string) error {
	return fmt.Errorf("delete - remote not implemented yet")
}

func (c *ConfigStorage) read() (*gitconfig.Config, error) {
	f, err := c.dir.Config()
	if err != nil {
		return nil, err
	}

	defer f.Close()

	cfg := gitconfig.New()
	d := gitconfig.NewDecoder(f)
	err = d.Decode(cfg)
	if err != nil {
		return nil, err
	}

	return cfg, nil
}

func parseRemote(s *gitconfig.Subsection) *config.RemoteConfig {
	fetch := []config.RefSpec{}
	for _, o := range s.Options {
		if o.IsKey(fetchKey) {
			rs := config.RefSpec(o.Value)
			if rs.IsValid() {
				fetch = append(fetch, rs)
			}
		}
	}

	return &config.RemoteConfig{
		Name:  s.Name,
		URL:   s.Option(urlKey),
		Fetch: fetch,
	}
}
