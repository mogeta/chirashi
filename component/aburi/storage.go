package aburi

// ParticleStorage defines the interface for saving and loading particle configurations
type ParticleStorage interface {
	// Save saves a particle configuration to the specified path
	Save(path string, config *ParticleConfig) error

	// Load loads a particle configuration from the specified path
	Load(path string) (*ParticleConfig, error)

	// List returns a list of file paths matching the pattern
	List(pattern string) ([]string, error)
}
