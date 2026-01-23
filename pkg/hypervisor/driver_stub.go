//go:build !darwin && !linux

package hypervisor

// NewDriver returns an error on unsupported platforms.
func NewDriver() (Driver, error) {
	return nil, ErrUnsupportedPlatform
}
