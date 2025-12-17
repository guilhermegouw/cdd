package styles

// NewDefaultTheme creates a clean dark theme for CDD.
func NewDefaultTheme() *Theme {
	return &Theme{
		Name:   "default",
		IsDark: true,

		// Clean blue/cyan tones
		Primary:   ParseHex("#61afef"), // Soft blue
		Secondary: ParseHex("#56b6c2"), // Cyan
		Tertiary:  ParseHex("#3e4451"), // Dark gray-blue
		Accent:    ParseHex("#c678dd"), // Purple accent

		// Dark backgrounds
		BgBase:    ParseHex("#1e1e1e"), // Dark background
		BgSubtle:  ParseHex("#252526"), // Slightly lighter
		BgOverlay: ParseHex("#2d2d30"), // Overlay background

		// Light foregrounds
		FgBase:   ParseHex("#abb2bf"), // Light gray text
		FgMuted:  ParseHex("#7f848e"), // Muted gray
		FgSubtle: ParseHex("#5c6370"), // Subtle gray

		// Borders
		Border:      ParseHex("#3e4451"),
		BorderFocus: ParseHex("#61afef"),

		// Status colors
		Success: ParseHex("#98c379"), // Green
		Error:   ParseHex("#e06c75"), // Red
		Warning: ParseHex("#e5c07b"), // Yellow
		Info:    ParseHex("#61afef"), // Blue
	}
}
