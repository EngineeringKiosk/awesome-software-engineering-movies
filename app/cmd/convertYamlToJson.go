package cmd

import (
	"encoding/json"
	"log"
	"os"
	"path"
	"path/filepath"

	"github.com/gosimple/slug"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"

	"github.com/EngineeringKiosk/awesome-software-engineering-movies/io"
	"github.com/EngineeringKiosk/awesome-software-engineering-movies/platform"
	"github.com/EngineeringKiosk/awesome-software-engineering-movies/youtube"
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
		if id, ok := youtube.ParseVideoID(movieInfo.Link); ok {
			movieInfo.VideoID = id
		} else {
			log.Printf("WARNING: could not parse YouTube video ID from %q in %s", movieInfo.Link, absYamlFilePath)
		}
		resolvePlatform(movieInfo, absYamlFilePath)
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
// entry whose title and link are both empty (in which case the key
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
		if v.Title == "" && v.Link == "" {
			log.Printf("WARNING: %s: localized.%s has neither title nor link set; remove the key or fill in at least one field",
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

// resolvePlatform fills in info.Platform from the link when YAML did
// not set it, and warns on the cases the maintainer should know
// about: YAML disagrees with the link, YAML names a platform whose
// link the tooling cannot recognise, or neither YAML nor the link
// yields a platform.
//
// The YAML value always wins — this function never overwrites a
// non-empty Platform.
//
// fileLabel is the human-friendly file path used only in the warning
// text; passing it in keeps the function pure-ish (no globals) and
// trivially testable.
func resolvePlatform(info *MovieInformation, fileLabel string) {
	detected, detectedOK := platform.Detect(info.Link)
	switch {
	case info.Platform == "" && detectedOK:
		info.Platform = detected
	case info.Platform == "" && !detectedOK:
		log.Printf("WARNING: %s has no platform in YAML and link %q matches no known platform; leaving empty",
			fileLabel, info.Link)
	case info.Platform != "" && detectedOK && info.Platform != detected:
		log.Printf("WARNING: %s YAML platform %q disagrees with link-detected platform %q; keeping YAML value",
			fileLabel, info.Platform, detected)
	case info.Platform != "" && !detectedOK:
		log.Printf("WARNING: %s YAML platform %q set but link %q matches no known platform; keeping YAML value",
			fileLabel, info.Platform, info.Link)
	}
}
