package cri

// Config defines the CRI configuration.
type Config struct {
	// Listen is the listening address which servers CRI.
	Listen string
	// NetworkPluginBinDir is the directory in which the binaries for the plugin is kept.
	NetworkPluginBinDir string
	// NetworkPluginConfDir is the directory in which the admin places a CNI conf.
	NetworkPluginConfDir string
}
