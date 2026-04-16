//go:build windows

package registry

import (
	"fmt"

	"golang.org/x/sys/windows/registry"
)

const regPath = "SOFTWARE\tibiantis\\Credentials"

// Snapshot reads the current values of keys A, B, C from the Tibiantis
// credentials registry path.
func Snapshot() (a, b []byte, c string, err error) {
	k, err := registry.OpenKey(registry.CURRENT_USER, regPath, registry.QUERY_VALUE)
	if err != nil {
		return nil, nil, "", fmt.Errorf("open registry key: %w", err)
	}
	defer k.Close()

	a, _, err = k.GetBinaryValue("A")
	if err != nil {
		return nil, nil, "", fmt.Errorf("read A: %w", err)
	}
	b, _, err = k.GetBinaryValue("B")
	if err != nil {
		return nil, nil, "", fmt.Errorf("read B: %w", err)
	}
	c, _, err = k.GetStringValue("C")
	if err != nil {
		return nil, nil, "", fmt.Errorf("read C: %w", err)
	}
	return a, b, c, nil
}

// Restore writes the given values back into the Tibiantis credentials
// registry path.
func Restore(a, b []byte, c string) error {
	k, err := registry.OpenKey(registry.CURRENT_USER, regPath, registry.SET_VALUE)
	if err != nil {
		return fmt.Errorf("open registry key: %w", err)
	}
	defer k.Close()

	if err := k.SetBinaryValue("A", a); err != nil {
		return fmt.Errorf("write A: %w", err)
	}
	if err := k.SetBinaryValue("B", b); err != nil {
		return fmt.Errorf("write B: %w", err)
	}
	if err := k.SetStringValue("C", c); err != nil {
		return fmt.Errorf("write C: %w", err)
	}
	return nil
}
