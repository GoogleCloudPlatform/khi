package generator

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"text/template"

	"github.com/GoogleCloudPlatform/khi/pkg/document/splitter"
)

// DocumentGenerator generates a document from a template.
// If there is already text added other than the parts automatically generated by the template, the text added after the immediately preceding automatically generated part will be kept after the corresponding generated text section.
type DocumentGenerator struct {
	template *template.Template
}

func NewDocumentGeneratorFromTemplateFileGlob(templateFileGlob string) (*DocumentGenerator, error) {
	template, err := template.ParseGlob(templateFileGlob)
	if err != nil {
		return nil, err
	}
	return &DocumentGenerator{
		template: template,
	}, nil
}

func newDocumentGeneratorFromStringTemplate(templateStr string) (*DocumentGenerator, error) {
	return &DocumentGenerator{
		template: template.Must(template.New("").Parse(templateStr)),
	}, nil
}

// GenerateDocument creates or update document at the specified path with the specified template and parameter.
// When the ignoreNonMatchingGeneratedSection is false, this function returns error when it can't find the associated generated section preceding added text not to lose the edit.
func (g *DocumentGenerator) GenerateDocument(destination string, templateName string, parameters any, ignoreNonMatchingGeneratedSection bool) error {
	exists, err := checkFileExists(destination)
	if err != nil {
		return err
	}
	currentDestinationContent := ""
	if exists {
		currentDestinationContent, err = readFromFile(destination)
		if err != nil {
			return err
		}
	}

	outputString, err := g.generateDocumentString(currentDestinationContent, templateName, parameters, ignoreNonMatchingGeneratedSection)
	if err != nil {
		return err
	}

	return writeToFile(destination, outputString)
}

func (g *DocumentGenerator) generateDocumentString(destinationString string, templateName string, parameters any, ignoreNonMatchingGeneratedSection bool) (string, error) {
	outputBuffer := new(bytes.Buffer)
	err := g.template.ExecuteTemplate(outputBuffer, templateName, parameters)
	if err != nil {
		return "", err
	}
	outputString := outputBuffer.String()
	autoGeneratedSections, err := splitter.SplitToDocumentSections(outputString)
	if err != nil {
		return "", err
	}

	prevGeneratedSections, err := splitter.SplitToDocumentSections(destinationString)
	if err != nil {
		return "", err
	}

	outputString, err = concatAmendedContents(autoGeneratedSections, prevGeneratedSections, ignoreNonMatchingGeneratedSection)
	if err != nil {
		return "", err
	}
	return outputString, nil
}

func concatAmendedContents(generated []*splitter.DocumentSection, prev []*splitter.DocumentSection, ignoreNonMatchingGeneratedSection bool) (string, error) {
	var resultSections []*splitter.DocumentSection
	prevToNextMap := make(map[string]*splitter.DocumentSection)
	usedPrevIds := make(map[string]interface{})
	for index, section := range prev {
		if section.Type == splitter.SectionTypeAmend {
			if index == 0 {
				resultSections = append(resultSections, section)
			} else {
				prevToNextMap[prev[index-1].Id] = section
				usedPrevIds[prev[index-1].Id] = struct{}{}
			}
		}
	}

	for _, section := range generated {
		if section.Type != splitter.SectionTypeGenerated {
			continue
		}
		resultSections = append(resultSections, section)
		if next, ok := prevToNextMap[section.Id]; ok {
			resultSections = append(resultSections, next)
			delete(usedPrevIds, section.Id)
		}
	}

	if len(usedPrevIds) > 0 && !ignoreNonMatchingGeneratedSection {
		var nonUsedIds []string
		for k := range usedPrevIds {
			nonUsedIds = append(nonUsedIds, k)
		}
		return "", fmt.Errorf("previous amended sections belongs to other generated sections is not used. Unused ids %v", nonUsedIds)
	}

	result := ""
	for _, section := range resultSections {
		result += section.Body + "\n"
	}
	return result, nil
}

func checkFileExists(path string) (bool, error) {
	_, err := os.Stat(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return false, nil
		} else {
			return false, err
		}
	}
	return true, nil
}

func writeToFile(path string, content string) error {
	return os.WriteFile(path, []byte(content), 0644)
}

func readFromFile(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	return string(data), nil
}
