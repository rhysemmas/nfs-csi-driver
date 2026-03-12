package driver

// Config holds driver configuration.
type Config struct {
	DriverName   string
	NodeID       string
	NFSServer    string
	NFSRootPath  string
	NFSRootMount string
}
