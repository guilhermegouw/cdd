package styles

// NewDefaultTheme creates an ocean-inspired dark theme for CDD.
func NewDefaultTheme() *Theme {
	return &Theme{
		Name:   "default",
		IsDark: true,

		// Ocean blue tones
		Primary:   ParseHex("#5eb5f7"), // Clear ocean blue
		Secondary: ParseHex("#7ec8e8"), // Light sky blue
		Tertiary:  ParseHex("#2d4a5e"), // Deep ocean
		Accent:    ParseHex("#8fd4f4"), // Bright water

		// Dark backgrounds
		BgBase:    ParseHex("#0f1419"), // Deep sea dark
		BgSubtle:  ParseHex("#1a2028"), // Slightly lighter
		BgOverlay: ParseHex("#232a32"), // Overlay background

		// Light foregrounds
		FgBase:   ParseHex("#c5d1de"), // Soft white-blue
		FgMuted:  ParseHex("#7a8b99"), // Muted blue-gray
		FgSubtle: ParseHex("#4d5b66"), // Subtle blue-gray

		// Borders
		Border:      ParseHex("#2d4a5e"),
		BorderFocus: ParseHex("#5eb5f7"),

		// Status colors
		Success: ParseHex("#7ec8e8"), // Light blue (calm success)
		Error:   ParseHex("#f4726d"), // Coral red
		Warning: ParseHex("#f4c56d"), // Sandy amber
		Info:    ParseHex("#5eb5f7"), // Ocean blue
	}
}
