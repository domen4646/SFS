package main

import (
	"os"
	"encoding/json"
	"fmt"
	"errors"
)

type Settings struct {
	FolderPath string
	SizeLimit uint64
	SingleFileSizeLimit uint64
	ReadOnly bool
	ForbiddenExtensions []string
}

// Load settings from file
func (s *Settings) Load(filename string) error {
	data, err := os.ReadFile(filename)
	if err != nil {
		return err
	}
	err = json.Unmarshal(data, &s)
	if err != nil {
		return err
	}
	return nil
}

// Save settings to file
func (s Settings) Save(filename string) error {
	json, err := json.Marshal(s)
	if err != nil {
		return err
	}
	err = os.WriteFile(filename, json, os.ModePerm)
	if err != nil {
		return err
	}
	return nil
}

// Get default settings
func DefaultSettings() Settings {
	return Settings {
		FolderPath: "./uploads",
		SizeLimit: 128,
		SingleFileSizeLimit: 8,
		ReadOnly: false,
		ForbiddenExtensions: []string{".html"},
	}
}

// Attempt to load settings. if file does not exist write default settings and return those
// If there is failure at any point, panic
func LoadSettingsOrPanic(filename string) Settings {
	var settings Settings
	err := settings.Load(filename)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			// Attempt to write default settings
			fmt.Printf("File %s does not exist. Writing default settings ...\n", filename)
			settings = DefaultSettings()
			err = settings.Save(filename)
			if err != nil {
				panic("Failed to save default settings!")
			}
			fmt.Printf("Written default settings to %s.\n", filename)
			os.Exit(0)
			return Settings{}
		}
		panic("Failed to load settings from " + filename + "!")
	}
	return settings
}
