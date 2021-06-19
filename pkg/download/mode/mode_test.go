package mode

import (
	"testing"

	"github.com/gomods/athens/pkg/errors"
)

var testCases = []struct {
	name         string
	file         *DownloadFile
	input        string
	expectedMode Mode
	expectedURL  string
}{
	{
		name:         "sync",
		file:         &DownloadFile{Mode: Sync},
		input:        "github.com/gomods/athens",
		expectedMode: Sync,
	},
	{
		name:         "redirect",
		file:         &DownloadFile{Mode: Redirect, DownloadURL: "gomods.io"},
		input:        "github.com/gomods/athens",
		expectedMode: Redirect,
		expectedURL:  "gomods.io",
	},
	{
		name:         "redirect with download url suffix",
		file:         &DownloadFile{Mode: Redirect, DownloadURL: "internal.domain/repository/gonexus"},
		input:        "github.com/gomods/athens",
		expectedMode: Redirect,
		expectedURL:  "internal.domain/repository/gonexus",
	},
	{
		name: "pattern match",
		file: &DownloadFile{
			Mode: Sync,
			Paths: []*DownloadPath{
				{Pattern: "github.com/gomods/*", Mode: None},
			},
		},
		input:        "github.com/gomods/athens",
		expectedMode: None,
	},
	{
		name: "multiple depth pattern match",
		file: &DownloadFile{
			Mode: Sync,
			Paths: []*DownloadPath{
				{Pattern: "github.com/*", Mode: None},
			},
		},
		input:        "github.com/gomods/athens/pkg/mode",
		expectedMode: None,
	},
	{
		name: "subdomain pattern match",
		file: &DownloadFile{
			Mode: Sync,
			Paths: []*DownloadPath{
				{Pattern: "*.github.com/gomods/*", Mode: None},
			},
		},
		input:        "athens.github.com/gomods/pkg/mode",
		expectedMode: None,
	},
	{
		name: "pattern fallback",
		file: &DownloadFile{
			Mode: Sync,
			Paths: []*DownloadPath{
				{Pattern: "github.com/gomods/*", Mode: None},
			},
		},
		input:        "github.com/athens-artifacts/maturelib",
		expectedMode: Sync,
	},
	{
		name: "pattern redirect",
		file: &DownloadFile{
			Mode: Sync,
			Paths: []*DownloadPath{
				{
					Pattern:     "github.com/gomods/*",
					Mode:        AsyncRedirect,
					DownloadURL: "gomods.io"},
			},
		},
		input:        "github.com/gomods/athens",
		expectedMode: AsyncRedirect,
		expectedURL:  "gomods.io",
	},
	{
		name: "redirect fallback",
		file: &DownloadFile{
			Mode:        Redirect,
			DownloadURL: "proxy.golang.org",
			Paths: []*DownloadPath{
				{
					Pattern:     "github.com/gomods/*",
					Mode:        AsyncRedirect,
					DownloadURL: "gomods.io",
				},
			},
		},
		input:        "github.com/athens-artifacts/maturelib",
		expectedMode: Redirect,
		expectedURL:  "proxy.golang.org",
	},
}

func TestMode(t *testing.T) {
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			givenMode := tc.file.Match(tc.input)
			if givenMode != tc.expectedMode {
				t.Fatalf("expected matched mode to be %q but got %q", tc.expectedMode, givenMode)
			}
			givenURL := tc.file.URL(tc.input)
			if givenURL != tc.expectedURL {
				t.Fatalf("expected matched DownloadURL to be %q but got %q", tc.expectedURL, givenURL)
			}
		})
	}
}

func TestNewFile_err(t *testing.T) {
	const op = errors.Op("downloadMode.NewFile")
	tc := []struct {
		name   string
		mode   Mode
		hasErr bool
	}{
		{
			name:   "empty mode",
			mode:   "",
			hasErr: true,
		},
		{
			name:   "invalid mode",
			mode:   "invalidMode",
			hasErr: true,
		},
	}
	for _, c := range tc {
		t.Run(c.name, func(subT *testing.T) {
			_, err := NewFile(c.mode, "github.com/gomods/athens")
			if c.hasErr && err == nil {
				t.Errorf(
					"Expected error for mode %s, but got none",
					c.mode,
				)
			}
			if !c.hasErr && err != nil {
				t.Errorf(
					"Expected no error for mode %s, but got %s",
					c.mode,
					err,
				)
			}
		})
	}
	// loop through all of the valid modes
	modeStrings := []string{
		"sync",
		"async",
		"redirect",
		"async_redirect",
		"none",
	}
	for _, modeString := range modeStrings {
		_, err := NewFile(Mode(modeString), "github.com/gomods/athens")
		if err != nil {
			t.Errorf(
				"Expected no error for mode %s, got %s",
				modeString,
				err,
			)
		}
	}
}
