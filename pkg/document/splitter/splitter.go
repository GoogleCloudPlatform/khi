package splitter

import (
	"crypto/sha256"
	"fmt"
	"strings"
)

// Generated section comments are expected to use a line. Document contents must not be on the same line.
const beginGeneratedSectionPrefix = "<!-- BEGIN GENERATED PART:"
const endGeneratedSectionPrefix = "<!-- END GENERATED PART:"
const generatedSectionSuffix = "-->"

type SectionType int

const (
	// SectionTypeGenerated is the section type indicating the section is generated automatically.
	SectionTypeGenerated = 0
	// SectionTypeAmend is the section type indicating the section is added after the generation.
	SectionTypeAmend = 1
)

// DocumentSection represents a section inside the document.
// The splitter read given text and split them in multiple DocumentSection.
type DocumentSection struct {
	Type SectionType
	Id   string
	Body string
}

// SplitToDocumentSections splits text to array of DocumentSection
func SplitToDocumentSections(text string) ([]*DocumentSection, error) {
	lines := strings.Split(text, "\n")
	var sections []*DocumentSection
	var currentSection *DocumentSection

	for lineIndex, line := range lines {
		lineWithoutSpace := strings.TrimSpace(line)
		if strings.HasPrefix(lineWithoutSpace, beginGeneratedSectionPrefix) && strings.HasSuffix(lineWithoutSpace, generatedSectionSuffix) {
			if currentSection != nil {
				if currentSection.Type == SectionTypeGenerated {
					return nil, fmt.Errorf("invalid begin of section. section began twice. line %d", lineIndex+1)
				}
				sections = append(sections, currentSection)
			}
			id := readIdFromGeneratedSectionComment(lineWithoutSpace)
			currentSection = &DocumentSection{
				Type: SectionTypeGenerated,
				Id:   id,
				Body: line,
			}
			continue
		}
		if strings.HasPrefix(lineWithoutSpace, endGeneratedSectionPrefix) && strings.HasSuffix(lineWithoutSpace, generatedSectionSuffix) {
			id := readIdFromGeneratedSectionComment(lineWithoutSpace)
			if currentSection == nil {
				return nil, fmt.Errorf("invalid end of section. section id %s ended but not began. line %d", id, lineIndex+1)
			}
			if currentSection.Id != id {
				return nil, fmt.Errorf("invalid end of section. section id %s ended but the id is not matching with the previous section id %s. line %d", id, currentSection.Id, lineIndex+1)
			}
			currentSection.Body += "\n" + line
			sections = append(sections, currentSection)
			currentSection = nil
			continue
		}

		if currentSection == nil {
			currentSection = &DocumentSection{
				Type: SectionTypeAmend,
				Id:   "",
				Body: line,
			}
			continue
		}

		currentSection.Body += "\n" + line
	}

	if currentSection != nil {
		if currentSection.Type == SectionTypeGenerated {
			return nil, fmt.Errorf("invalid end of section. section id %s began but not ended", currentSection.Id)
		}
		if currentSection.Body != "" {
			sections = append(sections, currentSection)
		}
	}

	for _, section := range sections {
		// generate section id for ammended section. This uses hash just because I don't want to use random string to improve testability.
		if section.Id == "" {
			section.Id = getHashFromText(section.Body)
		}
	}
	return sections, nil
}

// readIdFromGeneratedSectionComment extract the id part of the comment of generated section.
// Example: input: <!-- BEGIN GENERATED PART:my-id--> returns "my-id"
func readIdFromGeneratedSectionComment(line string) string {
	return strings.TrimSpace(strings.TrimSuffix(strings.TrimPrefix(strings.TrimPrefix(strings.TrimSpace(line), beginGeneratedSectionPrefix), endGeneratedSectionPrefix), generatedSectionSuffix))
}

func getHashFromText(text string) string {
	h := sha256.New()
	h.Write([]byte(text))
	return fmt.Sprintf("%x", h.Sum(nil))
}
