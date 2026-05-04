package cmd

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path"
	"path/filepath"

	"github.com/gosimple/slug"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"

	"github.com/EngineeringKiosk/awesome-software-engineering-movies/io"
	"github.com/EngineeringKiosk/awesome-software-engineering-movies/platform"
)

// convertYamlToJsonCmd represents the convertYamlToJson command
var convertYamlToJsonCmd = &cobra.Command{
	Use:   "convertYamlToJson",
	Short: "Converts movie YAML files into JSON files",
	Long: `The YAML representation of the basic movie info is more for humans.
For machines we have a JSON format with more information about the movie available.

This command converts the basic YAML information into JSON format.`,
	RunE: cmdConvertYamlToJson,
}

func init() {
	rootCmd.AddCommand(convertYamlToJsonCmd)

	convertYamlToJsonCmd.Flags().String("yaml-directory", "", "Directory on where to find the yaml files")
	convertYamlToJsonCmd.Flags().String("json-directory", "", "Directory on where to store the json files")

	err := convertYamlToJsonCmd.MarkFlagRequired("yaml-directory")
	if err != nil {
		log.Fatalf("Error marking flag as required: %v", err)
	}
	err = convertYamlToJsonCmd.MarkFlagRequired("json-directory")
	if err != nil {
		log.Fatalf("Error marking flag as required: %v", err)
	}
	convertYamlToJsonCmd.MarkFlagsRequiredTogether("yaml-directory", "json-directory")
}

func cmdConvertYamlToJson(cmd *cobra.Command, args []string) error {
	yamlDir, err := cmd.Flags().GetString("yaml-directory")
	if err != nil {
		return err
	}

	jsonDir, err := cmd.Flags().GetString("json-directory")
	if err != nil {
		return err
	}

	yamlExts := io.GetYAMLExtensions()
	log.Printf("Reading files with extensions %v from directory %s", yamlExts, yamlDir)
	yamlFiles, err := io.GetAllFilesFromDirectoryWithExtensions(yamlDir, yamlExts)
	if err != nil {
		return err
	}
	log.Printf("%d files found with extensions %v in directory %s", len(yamlFiles), yamlExts, yamlDir)

	log.Printf("Reading files with extension %s from directory %s", io.JSONExtension, jsonDir)
	jsonFiles, err := io.GetAllFilesFromDirectory(jsonDir, io.JSONExtension)
	if err != nil {
		return err
	}
	log.Printf("%d files found with extension %s in directory %s", len(jsonFiles), io.JSONExtension, jsonDir)

	// Process every YAML file found and dump it into a JSON file with
	// the same name. If the JSON file already exists, the YAML is the
	// source of truth for the manually-curated fields and the API-
	// enriched fields are preserved on top.
	for _, f := range yamlFiles {
		absYamlFilePath := filepath.Join(yamlDir, f.Name())
		log.Printf("Processing file %s", absYamlFilePath)
		yamlFileContent, err := os.ReadFile(absYamlFilePath)
		if err != nil {
			return err
		}

		movieInfo := &MovieInformation{}
		err = yaml.Unmarshal(yamlFileContent, movieInfo)
		if err != nil {
			return err
		}

		currentFileExtension := path.Ext(f.Name())
		jsonFileName := f.Name()[0:len(f.Name())-len(currentFileExtension)] + io.JSONExtension
		absJsonFilePath := filepath.Join(jsonDir, jsonFileName)

		log.Printf("Converting %s to %s", absYamlFilePath, absJsonFilePath)

		if _, ok := jsonFiles[jsonFileName]; ok {
			// JSON file exists — read it, then overwrite the YAML-sourced
			// fields while keeping enriched fields intact.
			jsonFileContent, err := os.ReadFile(absJsonFilePath)
			if err != nil {
				return err
			}

			movieJsonInfo := &MovieInformation{}
			err = json.Unmarshal(jsonFileContent, movieJsonInfo)
			if err != nil {
				return err
			}

			movieInfo = mergeMovieInformation(movieInfo, movieJsonInfo)
		}

		// Generated fields
		movieInfo.Slug = slug.Make(movieInfo.Name)
		resolvePlatform(movieInfo, absYamlFilePath)
		resolveLocalizedPlatforms(movieInfo, absYamlFilePath)
		validateLocalized(movieInfo, absYamlFilePath)

		log.Printf("Write %s to disk ...", absJsonFilePath)
		err = io.WriteJSONFile(absJsonFilePath, movieInfo)
		if err != nil {
			return err
		}
		log.Printf("Write %s to disk ... successful", absJsonFilePath)
	}

	log.Printf("Converting of YAML to JSON ... successful")
	return nil
}

// mergeMovieInformation overwrites the YAML-sourced fields on target
// with the values from source while preserving the API-enriched
// fields already present in target. If the YAML schema gains new
// curated fields, they must be added here as well.
//
// Language, Subtitles and Description are special: all are optional
// in YAML. If the YAML omits them we keep whatever target already has
// (which may be a value previously derived from the YouTube API in
// collectMovieData), so the YAML acts as an override rather than a
// forced reset. The union with API-supplied codes is performed later
// in collectMovieData; here we only handle the YAML-vs-cached merge.
func mergeMovieInformation(source, target *MovieInformation) *MovieInformation {
	target.Name = source.Name
	target.Link = source.Link
	if len(source.Language) > 0 {
		target.Language = source.Language
	}
	if len(source.Subtitles) > 0 {
		target.Subtitles = source.Subtitles
	}
	if len(source.Description) > 0 {
		target.Description = source.Description
	}
	if len(source.IMDbID) > 0 {
		target.IMDbID = source.IMDbID
	}
	if len(source.Platform) > 0 {
		target.Platform = source.Platform
	}
	if len(source.Localized) > 0 {
		target.Localized = source.Localized
	}
	target.Tags = source.Tags
	return target
}

// validateLocalized inspects info.Localized for the obvious
// mistakes: a key that does not look like an ISO 639-1 code, or an
// entry with no override fields set at all (in which case the key
// adds nothing to the file). Each problem is logged as a warning so
// the maintainer notices, but the conversion proceeds — the data
// shape is still valid YAML.
//
// fileLabel is the human-friendly file path for the warning text.
func validateLocalized(info *MovieInformation, fileLabel string) {
	for code, v := range info.Localized {
		if !isISO639_1Like(code) {
			log.Printf("WARNING: %s: localized language key %q is not a 2-letter lowercase code; expected ISO 639-1",
				fileLabel, code)
		}
		if v.Title == "" && v.Link == "" && v.Description == "" {
			log.Printf("WARNING: %s: localized.%s has no overrides set; remove the key or fill in at least one of title/link/description",
				fileLabel, code)
		}
	}
}

// isISO639_1Like is a cheap shape check, not a registry lookup —
// catching typos like "DE" or "deu" matters; being authoritative
// about which two-letter codes are real does not.
func isISO639_1Like(s string) bool {
	if len(s) != 2 {
		return false
	}
	return s[0] >= 'a' && s[0] <= 'z' && s[1] >= 'a' && s[1] <= 'z'
}

// resolvePlatformValue centralises the four-warning-branch decision
// so both the top-level entry and per-localized entries reuse it
// without duplication. Returns the resolved platform value (which
// may be empty) and emits warnings as a side effect.
//
// The YAML value always wins — when currentPlatform is non-empty it
// flows through unchanged.
func resolvePlatformValue(currentPlatform, link, label string) string {
	detected, detectedOK := platform.Detect(link)
	switch {
	case currentPlatform == "" && detectedOK:
		return detected
	case currentPlatform == "" && !detectedOK:
		log.Printf("WARNING: %s has no platform in YAML and link %q matches no known platform; leaving empty",
			label, link)
		return ""
	case currentPlatform != "" && detectedOK && currentPlatform != detected:
		log.Printf("WARNING: %s YAML platform %q disagrees with link-detected platform %q; keeping YAML value",
			label, currentPlatform, detected)
		return currentPlatform
	case currentPlatform != "" && !detectedOK:
		log.Printf("WARNING: %s YAML platform %q set but link %q matches no known platform; keeping YAML value",
			label, currentPlatform, link)
		return currentPlatform
	}
	// YAML value matches the detected value, or both are empty —
	// nothing to report.
	return currentPlatform
}

// resolvePlatform fills in info.Platform from the top-level link
// when YAML did not set it, surfacing the four warning paths from
// resolvePlatformValue with a fileLabel-only label.
func resolvePlatform(info *MovieInformation, fileLabel string) {
	info.Platform = resolvePlatformValue(info.Platform, info.Link, fileLabel)
}

// resolveLocalizedPlatforms fills in Platform on each localized
// entry whose link is set. Description-only or title-only overrides
// have no link to detect against and are left untouched.
func resolveLocalizedPlatforms(info *MovieInformation, fileLabel string) {
	for code, v := range info.Localized {
		if v.Link == "" {
			continue
		}
		v.Platform = resolvePlatformValue(v.Platform, v.Link, fmt.Sprintf("%s localized.%s", fileLabel, code))
		info.Localized[code] = v
	}
}
