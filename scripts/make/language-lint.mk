# Makefile for linting Markdown files in Japanese using LanguageTool

# Set the desired LanguageTool version
LANGUAGETOOL_VERSION = 6.5
LANGUAGETOOL_ZIP = LanguageTool-stable.zip
LANGUAGETOOL_DIR = languagetool-6.5
LANGUAGETOOL_JAR = $(LANGUAGETOOL_DIR)/languageTool-commandline.jar

.PHONY: lint download-lt

# Download and extract LanguageTool if the jar doesn't exist
$(LANGUAGETOOL_JAR):
	@echo "Downloading LanguageTool version $(LANGUAGETOOL_VERSION)..."
	@wget -q https://languagetool.org/download/LanguageTool-stable.zip -O $(LANGUAGETOOL_ZIP)
	@echo "Extracting LanguageTool..."
	@unzip -q $(LANGUAGETOOL_ZIP)
	@rm $(LANGUAGETOOL_ZIP)
	@if [ ! -f $(LANGUAGETOOL_JAR) ]; then \
	  echo "Error: $(LANGUAGETOOL_JAR) not found after extraction."; \
	  exit 1; \
	fi

# Default target: lint all markdown files
languagelint: $(LANGUAGETOOL_JAR)
	@echo "Linting all Markdown files with LanguageTool (Japanese)..."
	@# Find all .md files and lint them one-by-one.
	@for file in $(shell find . -type f -name "*.md"); do \
	  echo "Linting $$file..."; \
	  java -jar $(LANGUAGETOOL_JAR) --language ja "$$file" || exit 1; \
	done
	@echo "Linting complete."

