package settings

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

type Storage struct {
	file     string
	preset   string
	settings *Settings
}

type StoredSettings struct {
	Preset string `yaml:"preset"`
}

func New(dir string) (*Storage, error) {
	s := &Storage{
		file: filepath.Join(dir, "settings.yaml"),
	}

	data, err := os.ReadFile(s.file)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			s.preset = Ancestra
			s.settings = Presets[Ancestra]
			return s, nil
		}
		return nil, err
	}

	stored := &StoredSettings{}
	if err := yaml.Unmarshal(data, stored); err != nil {
		return nil, err
	}

	p, ok := Presets[stored.Preset]
	if !ok {
		return nil, fmt.Errorf("unknown preset %q", stored.Preset)
	}
	s.preset = stored.Preset
	s.settings = p

	return s, nil
}

func (s *Storage) Preset() string {
	return s.preset
}

func (s *Storage) Get() *Settings {
	return s.settings
}

func (s *Storage) SwitchPreset(id string) error {
	p, ok := Presets[id]
	if !ok {
		return fmt.Errorf("unknown preset %q", id)
	}

	data, err := yaml.Marshal(&StoredSettings{
		Preset: id,
	})
	if err != nil {
		return err
	}

	if err := os.MkdirAll(filepath.Dir(s.file), 0755); err != nil {
		return err
	}

	if err := os.WriteFile(s.file, data, 0644); err != nil {
		return err
	}

	s.settings = p

	return nil
}
