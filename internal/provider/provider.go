package provider

type MagnetProvider interface {
	Initiate()
	RunGet()
}
