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
	target.Tags = source.Tags
	return target
}
