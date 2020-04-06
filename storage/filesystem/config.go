package filesystem

import (
	"fmt"
	stdioutil "io/ioutil"
	"os"

	"gopkg.in/src-d/go-git.v4/config"
	"gopkg.in/src-d/go-git.v4/storage/filesystem/dotgit"
	"gopkg.in/src-d/go-git.v4/utils/ioutil"
)

type ConfigStorage struct {
	dir *dotgit.DotGit
}

func (c *ConfigStorage) Config() (conf *config.Config, err error) {
	cfg := config.NewConfig()
	var b []byte
	data := []byte("\n")

	if b, err = c.systemConfig(); err != nil {
		return nil, err
	}
	data = append(data, b...)
	data = append(data, []byte("\n")...)

	if b, err = c.userConfig(); err != nil {
		return nil, err
	}
	data = append(data, b...)
	data = append(data, []byte("\n")...)

	if b, err = c.localConfig(); err != nil {
		return nil, err
	}
	data = append(data, b...)
	data = append(data, []byte("\n")...)

	if err = cfg.Unmarshal(data); err != nil {
		return nil, err
	}

	return cfg, err
}

func (c *ConfigStorage) systemConfig() (b []byte, err error) {
	if b, err = c.dir.SystemConfig(); err != nil {
		if os.IsNotExist(err) {
			return []byte{}, nil
		}

		return nil, err
	}

	return b, nil
}

func (c *ConfigStorage) userConfig() (b []byte, err error) {
	if b, err = c.dir.UserConfig(); err != nil {
		fmt.Println("USER ERROR: ", err)
		if os.IsNotExist(err) {
			return []byte{}, nil
		}

		return nil, err
	}

	return b, nil
}

func (c *ConfigStorage) localConfig() (b []byte, err error) {
	f, err := c.dir.Config()
	if err != nil {
		if os.IsNotExist(err) {
			return []byte{}, nil
		}

		return nil, err
	}

	defer ioutil.CheckClose(f, &err)

	b, err = stdioutil.ReadAll(f)
	if err != nil {
		return nil, err
	}

	return b, nil
}

func (c *ConfigStorage) SetConfig(cfg *config.Config) (err error) {
	if err = cfg.Validate(); err != nil {
		return err
	}

	f, err := c.dir.ConfigWriter()
	if err != nil {
		return err
	}

	defer ioutil.CheckClose(f, &err)

	b, err := cfg.Marshal()
	if err != nil {
		return err
	}

	_, err = f.Write(b)
	return err
}
