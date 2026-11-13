// Copyright 2026 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package report

import (
	"fmt"
	"io"

	"github.com/GoogleCloudPlatform/khi/scripts/codequality/internal/rule"
	"github.com/GoogleCloudPlatform/khi/scripts/gotestutil"
)

func renderSimilarityHeaderLine(w io.Writer, sim rule.SimilaritySummary, style gotestutil.TermStyle) {
	if sim.Skipped {
		fmt.Fprintf(w, "  Similarity: %s (%s)\n", style.Yellow("SKIPPED"), sim.SkipReason)
		return
	}
	if len(sim.Advisories) == 0 {
		msg := fmt.Sprintf("OK (%d new unit(s) checked against %d corpus units, 0 similar matches)",
			sim.NewUnitsEvaluated, sim.CorpusUnitsCount)
		fmt.Fprintf(w, "  Similarity: %s\n", style.BoldGreen(msg))
		return
	}
	msg := fmt.Sprintf("ADVISORY (%d/%d new unit(s) matched existing code across %d corpus units)",
		len(sim.Advisories), sim.NewUnitsEvaluated, sim.CorpusUnitsCount)
	fmt.Fprintf(w, "  Similarity: %s\n", style.Yellow(msg))
}

func renderSimilarityAdvisories(w io.Writer, lang string, advisories []rule.SimilarityAdvisory, style gotestutil.TermStyle) {
	header := fmt.Sprintf("── Similar Existing %s Functions (Advisory: %d new unit(s) matched) ──", lang, len(advisories))
	fmt.Fprintln(w)
	fmt.Fprintln(w, style.BoldYellow(header))
	renderSimilaritySummaryTable(w, advisories, style)
	renderSimilarityDetails(w, advisories, style)
}

func renderSimilaritySummaryTable(w io.Writer, advisories []rule.SimilarityAdvisory, style gotestutil.TermStyle) {
	headers := []string{"#", "SCORE", "NEW UNIT", "TOP EXISTING CANDIDATE"}
	rows := make([][]gotestutil.TableCell, 0, len(advisories))
	for i, adv := range advisories {
		top := adv.Candidates[0]
		tag := fmt.Sprintf("[S%d]", indexOrFallback(adv.Index, i+1))
		scoreTxt := fmt.Sprintf("%.3f", top.Score)
		newTarget := fmt.Sprintf("%s:%d · %s", adv.NewUnit.FilePath, adv.NewUnit.StartLine, adv.NewUnit.Name)
		candCell := formatSimilarityCandidateCell(top, style)
		rows = append(rows, []gotestutil.TableCell{
			{Plain: tag, Styled: style.BoldYellow(tag)},
			{Plain: scoreTxt, Styled: style.Yellow(scoreTxt), AlignRight: true},
			{Plain: newTarget, Styled: newTarget},
			candCell,
		})
	}
	gotestutil.RenderAlignedTable(w, "  ", headers, rows, style)
}

func formatSimilarityCandidateCell(cand rule.SimilarityCandidate, style gotestutil.TermStyle) gotestutil.TableCell {
	base := fmt.Sprintf("%s:%d · %s", cand.Unit.FilePath, cand.Unit.StartLine, cand.Unit.Name)
	if !cand.SharedModule {
		return gotestutil.TableCell{Plain: base, Styled: base}
	}
	styled := base + " " + style.Cyan("[shared]")
	return gotestutil.TableCell{Plain: styled, Styled: styled}
}

func renderSimilarityDetails(w io.Writer, advisories []rule.SimilarityAdvisory, style gotestutil.TermStyle) {
	fmt.Fprintln(w)
	for i, adv := range advisories {
		tag := fmt.Sprintf("[S%d]", indexOrFallback(adv.Index, i+1))
		fmt.Fprintf(w, "  %s %s — %s:%d-%d\n",
			style.BoldYellow(tag), style.Bold(adv.NewUnit.Name),
			adv.NewUnit.FilePath, adv.NewUnit.StartLine, adv.NewUnit.EndLine)
		for rank, cand := range adv.Candidates {
			renderSingleSimilarityCandidate(w, rank+1, cand, style)
		}
	}
}

func renderSingleSimilarityCandidate(w io.Writer, rank int, cand rule.SimilarityCandidate, style gotestutil.TermStyle) {
	badge := ""
	if cand.SharedModule {
		badge = " " + style.Cyan("[shared]")
	}
	fmt.Fprintf(w, "      Candidate #%d: %s (%s:%d-%d)%s [ast-norm cosine: %.3f]\n",
		rank, style.Bold(cand.Unit.Name), cand.Unit.FilePath, cand.Unit.StartLine, cand.Unit.EndLine,
		badge, cand.Score)
	fmt.Fprintf(w, "      Action:       %s\n", cand.Guidance)
	for _, ln := range cand.Unit.Preview {
		fmt.Fprintf(w, "        %s\n", style.Dim(fmt.Sprintf("%4d | %s", ln.Number, ln.Text)))
	}
}

func renderMarkdownSimilarityMetric(w io.Writer, sim rule.SimilaritySummary) {
	if sim.Skipped {
		fmt.Fprintf(w, "| Function Similarity | Skipped (%s) |\n", gotestutil.EscapeMarkdownTable(sim.SkipReason))
		return
	}
	if len(sim.Advisories) == 0 {
		fmt.Fprintf(w, "| Function Similarity | OK (%d new unit(s), 0 matches across %d corpus units) |\n",
			sim.NewUnitsEvaluated, sim.CorpusUnitsCount)
		return
	}
	fmt.Fprintf(w, "| Function Similarity | Advisory (%d/%d new unit(s) matched across %d corpus units) |\n",
		len(sim.Advisories), sim.NewUnitsEvaluated, sim.CorpusUnitsCount)
}

func renderMarkdownSimilarityAdvisories(w io.Writer, lang string, advisories []rule.SimilarityAdvisory) {
	fmt.Fprintf(w, "### Similar Existing %s Functions (Advisory)\n\n", lang)
	fmt.Fprintln(w, "| New Unit | Top Existing Candidate | AST-Norm Score | Action |")
	fmt.Fprintln(w, "| :--- | :--- | :---: | :--- |")
	for _, adv := range advisories {
		top := adv.Candidates[0]
		newCell := fmt.Sprintf("`%s` (`%s:%d`)", gotestutil.EscapeMarkdownTable(adv.NewUnit.Name), adv.NewUnit.FilePath, adv.NewUnit.StartLine)
		candCell := fmt.Sprintf("`%s` (`%s:%d`)", gotestutil.EscapeMarkdownTable(top.Unit.Name), top.Unit.FilePath, top.Unit.StartLine)
		fmt.Fprintf(w, "| %s | %s | `%.3f` | %s |\n",
			newCell, candCell, top.Score, gotestutil.EscapeMarkdownTable(top.Guidance))
	}
	fmt.Fprintln(w)
}
