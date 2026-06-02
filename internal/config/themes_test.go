package config

import (
	"testing"
)

func TestValidateColor(t *testing.T) {
	tests := []struct {
		color    string
		field    string
		wantErr  bool
		errMsg   string
	}{
		{"#FF0000", "primary", false, ""},
		{"#FFF", "primary", false, ""},
		{"255", "primary", false, ""},
		{"red", "primary", false, ""},
		{"bright-blue", "primary", false, ""},
		{"", "primary", true, "color field 'primary' is required but empty"},
		{"#GGGGGG", "primary", true, "invalid color format for 'primary': #GGGGGG"},
		{"badcolor", "primary", true, "unknown named color for 'primary': badcolor"},
		{"999", "primary", true, "invalid color format for 'primary': 999 (ANSI codes must be 0-255)"},
	}

	for _, tt := range tests {
		t.Run(tt.color+"_"+tt.field, func(t *testing.T) {
			err := validateColor(tt.color, tt.field)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateColor(%s, %s) error = %v, wantErr %v", tt.color, tt.field, err, tt.wantErr)
				return
			}
			if err != nil && tt.errMsg != "" {
				got := err.Error()
				if len(got) < len(tt.errMsg) || got[:len(tt.errMsg)] != tt.errMsg {
					t.Errorf("validateColor(%s, %s) error = %v, expected to start with %s", tt.color, tt.field, err, tt.errMsg)
				}
			}
		})
	}
}

func TestValidateTheme(t *testing.T) {
	validTheme := &Theme{
		Name: "Test Theme",
		Colors: struct {
			Primary       string `yaml:"primary"`
			Secondary     string `yaml:"secondary"`
			Success       string `yaml:"success"`
			Warning       string `yaml:"warning"`
			Error         string `yaml:"error"`
			TextPrimary   string `yaml:"text_primary"`
			TextSecondary string `yaml:"text_secondary"`
			TextMuted     string `yaml:"text_muted"`
			Prompt        string `yaml:"prompt"`
			Separator     string `yaml:"separator"`
			Activity      string `yaml:"activity"`
			Background    string `yaml:"background"`
			Surface       string `yaml:"surface"`
		}{
			Primary:       "#FF0000",
			Secondary:     "#00FF00",
			Success:       "#0000FF",
			Warning:       "#FFFF00",
			Error:         "#FF00FF",
			TextPrimary:   "#FFFFFF",
			TextSecondary: "#CCCCCC",
			TextMuted:     "#888888",
			Prompt:        "#00AAFF",
			Separator:     "#00AAFF",
			Activity:      "#FFFFFF",
			Background:    "#000000",
			Surface:       "#111111",
		},
	}

	// Test valid theme
	if err := validateTheme(validTheme); err != nil {
		t.Errorf("validateTheme() with valid theme failed: %v", err)
	}

	// Test theme without name
	noNameTheme := *validTheme
	noNameTheme.Name = ""
	if err := validateTheme(&noNameTheme); err == nil {
		t.Error("validateTheme() should fail for theme without name")
	}

	// Test theme with invalid color
	invalidColorTheme := *validTheme
	invalidColorTheme.Colors.Primary = "#GGGGGG"
	if err := validateTheme(&invalidColorTheme); err == nil {
		t.Error("validateTheme() should fail for theme with invalid color")
	}
}

func TestLoadThemeGracefulFallback(t *testing.T) {
	// Test loading nonexistent theme falls back gracefully
	theme, err := LoadTheme("nonexistent-theme")
	if err != nil {
		t.Errorf("LoadTheme() should not return error for nonexistent theme, got: %v", err)
	}
	if theme == nil {
		t.Error("LoadTheme() should return a valid theme even for nonexistent theme")
	}
	if theme.Name != "Default" {
		t.Errorf("LoadTheme() should fall back to default theme, got: %s", theme.Name)
	}
}

func TestLoadThemeInvalidName(t *testing.T) {
	invalidNames := []string{
		"../../etc/passwd",
		"../secret",
		"theme/subdir",
		"theme name",
		"theme@name",
		"theme.yaml",
	}
	for _, name := range invalidNames {
		t.Run(name, func(t *testing.T) {
			theme, err := LoadTheme(name)
			if err != nil {
				t.Errorf("LoadTheme(%q) should not return error, got: %v", name, err)
			}
			if theme == nil {
				t.Errorf("LoadTheme(%q) should return built-in default theme for invalid name", name)
			}
			if theme.Name != "Default" {
				t.Errorf("LoadTheme(%q) should fall back to built-in theme, got: %s", name, theme.Name)
			}
		})
	}
}

func TestGetDefaultTheme(t *testing.T) {
	theme := getDefaultTheme()
	if theme.Name != "Default" {
		t.Errorf("getDefaultTheme() name = %s, want Default", theme.Name)
	}
	if err := validateTheme(theme); err != nil {
		t.Errorf("getDefaultTheme() should return valid theme: %v", err)
	}
}