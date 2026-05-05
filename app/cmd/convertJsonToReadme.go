package cmd

import (
	"encoding/json"
	"html/template"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/spf13/cobra"

	"github.com/EngineeringKiosk/awesome-software-engineering-movies/io"
	"github.com/EngineeringKiosk/awesome-software-engineering-movies/utils"
)

const defaultDescriptionMaxLength = 600

// convertJsonToReadmeCmd represents the convertJsonToReadme command
var convertJsonToReadmeCmd = &cobra.Command{
	Use:   "convertJsonToReadme",
	Short: "Converts generated movie JSON files into a repository README.md",
	Long: `The source of truth are our YAML files in movies/.
Those will be converted and enriched into JSON with the convertYamlToJson and
collectMovieData commands. To make it human readable, we generate a README.md
based on this JSON data.

This command converts the generated JSON information into a human readable README.`,
	RunE: cmdConvertJsonToReadme,
}

func init() {
	rootCmd.AddCommand(convertJsonToReadmeCmd)

	convertJsonToReadmeCmd.Flags().String("json-directory", "", "Directory on where to store the json files")
	convertJsonToReadmeCmd.Flags().String("readme-template", "", "Path to the README template")
	convertJsonToReadmeCmd.Flags().String("readme-output", "", "Path to the README file that will be written")
	convertJsonToReadmeCmd.Flags().Int("description-max-length", defaultDescriptionMaxLength, "Maximum length of the rendered description in runes; longer descriptions are cut at the next word or sentence boundary")

	for _, name := range []string{"json-directory", "readme-template", "readme-output"} {
		if err := convertJsonToReadmeCmd.MarkFlagRequired(name); err != nil {
			log.Fatalf("Error marking flag %q as required: %v", name, err)
		}
	}
	convertJsonToReadmeCmd.MarkFlagsRequiredTogether("json-directory", "readme-template", "readme-output")
}

func cmdConvertJsonToReadme(cmd *cobra.Command, args []string) error {
	readmeOutput, err := cmd.Flags().GetString("readme-output")
	if err != nil {
		return err
	}
	readmeTemplate, err := cmd.Flags().GetString("readme-template")
	if err != nil {
		return err
	}
	jsonDir, err := cmd.Flags().GetString("json-directory")
	if err != nil {
		return err
	}
	descriptionMaxLength, err := cmd.Flags().GetInt("description-max-length")
	if err != nil {
		return err
	}

	log.Printf("Reading files with extension %s from directory %s", io.JSONExtension, jsonDir)
	jsonFiles, err := io.GetAllFilesFromDirectory(jsonDir, io.JSONExtension)
	if err != nil {
		return err
	}
	log.Printf("%d files found with extension %s in directory %s", len(jsonFiles), io.JSONExtension, jsonDir)

	movies := make([]*MovieInformation, 0, len(jsonFiles))
	for _, f := range jsonFiles {
		absJsonFilePath := filepath.Join(jsonDir, f.Name())
		jsonFileContent, err := os.ReadFile(absJsonFilePath)
		if err != nil {
			return err
		}
		movieInfo := &MovieInformation{}
		if err := json.Unmarshal(jsonFileContent, movieInfo); err != nil {
			return err
		}
		movies = append(movies, movieInfo)
	}

	log.Printf("Sorting %d movies by name", len(movies))
	sort.Slice(movies, func(i, j int) bool {
		return strings.ToLower(movies[i].Name) < strings.ToLower(movies[j].Name)
	})

	grouped := groupMoviesByType(movies)

	log.Printf("Read template file %s from disk", readmeTemplate)
	readmeTemplateContent, err := os.ReadFile(readmeTemplate)
	if err != nil {
		return err
	}

	log.Printf("Create target file %s", readmeOutput)
	readmeTarget, err := os.Create(readmeOutput)
	if err != nil {
		return err
	}
	defer func() { _ = readmeTarget.Close() }()

	log.Printf("Render template and write it into %s (description max length %d runes) ...", readmeOutput, descriptionMaxLength)
	funcs := template.FuncMap{
		"truncateDescription": func(s string) string {
			return utils.TruncateTextRespectWords(s, descriptionMaxLength)
		},
	}
	t := template.Must(template.New("readme-template").Funcs(funcs).Parse(string(readmeTemplateContent)))
	if err := t.Execute(readmeTarget, grouped); err != nil {
		return err
	}
	log.Printf("Render template and write it into %s ... successful", readmeOutput)

	return nil
}

// groupedMovies is the value handed to the README template. Each
// bucket holds entries whose Type matches the bucket; unknown or
// empty types fall into Documentaries (the dominant catalogue type),
// which keeps an under-curated entry visible while
// validateCategoryAndType warns about it at YAML→JSON time.
type groupedMovies struct {
	TVSeries      []*MovieInformation
	Documentaries []*MovieInformation
	Movies        []*MovieInformation
	// Total is convenient for "(N entries)" copy in the intro.
	Total int
}

func groupMoviesByType(movies []*MovieInformation) groupedMovies {
	var g groupedMovies
	for _, m := range movies {
		switch m.Type {
		case "TV Series":
			g.TVSeries = append(g.TVSeries, m)
		case "Movie":
			g.Movies = append(g.Movies, m)
		default:
			g.Documentaries = append(g.Documentaries, m)
		}
	}
	g.Total = len(movies)
	return g
}
