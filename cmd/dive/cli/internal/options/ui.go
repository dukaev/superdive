package options

// UI combines all UI configuration elements
type UI struct {
	Version    string        `yaml:"version" mapstructure:"version"` // "v1" or "v2"
	Keybinding UIKeybindings `yaml:"keybinding" mapstructure:"keybinding"`
	Diff       UIDiff        `yaml:"diff" mapstructure:"diff"`
	Filetree   UIFiletree    `yaml:"filetree" mapstructure:"filetree"`
	Layer      UILayers      `yaml:"layer" mapstructure:"layer"`
}

func DefaultUI() UI {
	return UI{
		Version:    "v2",
		Keybinding: DefaultUIKeybinding(),
		Diff:       DefaultUIDiff(),
		Filetree:   DefaultUIFiletree(),
		Layer:      DefaultUILayers(),
	}
}
