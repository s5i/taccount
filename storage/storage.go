package storage

import (
	"encoding/base64"
	"os"

	"gopkg.in/yaml.v3"
)

type String string

func (b String) MarshalYAML() (interface{}, error) {
	return base64.StdEncoding.EncodeToString([]byte(b)), nil
}

func (b *String) UnmarshalYAML(node *yaml.Node) error {
	data, err := base64.StdEncoding.DecodeString(node.Value)
	if err != nil {
		return err
	}
	*b = String(data)
	return nil
}

type Bytes []byte

func (b Bytes) MarshalYAML() (interface{}, error) {
	return base64.StdEncoding.EncodeToString(b), nil
}

func (b *Bytes) UnmarshalYAML(node *yaml.Node) error {
	data, err := base64.StdEncoding.DecodeString(node.Value)
	if err != nil {
		return err
	}
	*b = data
	return nil
}

// Entry represents a single account entry with registry key values.
type Entry struct {
	ID        string `yaml:"id"`
	HumanName string `yaml:"human_name"`
	A         Bytes  `yaml:"a"`
	B         Bytes  `yaml:"b"`
	C         String `yaml:"c"`
}

// Load reads entries from a YAML file. Returns nil entries if the file
// does not exist.
func Load(path string) ([]Entry, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	var entries []Entry
	if err := yaml.Unmarshal(data, &entries); err != nil {
		return nil, err
	}
	return entries, nil
}

// Save writes entries to a YAML file.
func Save(path string, entries []Entry) error {
	data, err := yaml.Marshal(entries)
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0644)
}

// AddEntry appends a new entry to the slice.
func AddEntry(entries *[]Entry, e Entry) {
	*entries = append(*entries, e)
}

// DeleteEntry removes an entry by ID.
func DeleteEntry(entries *[]Entry, id string) {
	for i, e := range *entries {
		if e.ID == id {
			*entries = append((*entries)[:i], (*entries)[i+1:]...)
			return
		}
	}
}

// RenameEntry updates the HumanName of an entry identified by ID.
func RenameEntry(entries *[]Entry, id, newName string) {
	for i := range *entries {
		if (*entries)[i].ID == id {
			(*entries)[i].HumanName = newName
			return
		}
	}
}
