// Package pdf renders owner statements to PDF with fpdf — pure Go, no
// headless browser or external binary, so it runs unchanged in the
// distroless production image.
package pdf

import (
	"bytes"
	"fmt"
	"strings"
	"time"

	"github.com/go-pdf/fpdf"

	"propertymanagement/internal/domain"
)

// StatementRenderer implements domain.StatementRenderer.
type StatementRenderer struct{}

var _ domain.StatementRenderer = StatementRenderer{}

func NewStatementRenderer() StatementRenderer { return StatementRenderer{} }

const (
	pageMargin = 15.0
	colDate    = 28.0
	colAmount  = 34.0
)

var (
	ink   = [3]int{20, 24, 31}
	muted = [3]int{107, 118, 136}
	green = [3]int{47, 122, 85}
	line  = [3]int{221, 225, 232}
)

func formatCents(cents int64) string {
	sign := ""
	if cents < 0 {
		sign, cents = "-", -cents
	}
	whole := fmt.Sprintf("%d", cents/100)
	var b strings.Builder
	for i, c := range whole {
		if i > 0 && (len(whole)-i)%3 == 0 {
			b.WriteByte(',')
		}
		b.WriteRune(c)
	}
	return fmt.Sprintf("%s$%s.%02d", sign, b.String(), cents%100)
}

func (StatementRenderer) RenderOwnerStatement(s *domain.StatementSnapshot) ([]byte, error) {
	doc := fpdf.New("P", "mm", "A4", "")
	doc.SetMargins(pageMargin, pageMargin, pageMargin)
	doc.SetAutoPageBreak(true, pageMargin)
	// The built-in fonts are cp1252; this maps UTF-8 text (names with
	// accents, em dashes) onto it instead of printing garbage.
	tr := doc.UnicodeTranslatorFromDescriptor("")
	doc.SetTitle("Owner Statement — "+s.PropertyName, true)
	doc.AddPage()

	contentWidth := 210 - 2*pageMargin
	descWidth := contentWidth - colDate - colAmount

	doc.SetFont("Helvetica", "B", 20)
	doc.SetTextColor(green[0], green[1], green[2])
	doc.CellFormat(0, 10, "Owner Statement", "", 1, "L", false, 0, "")

	doc.SetFont("Helvetica", "", 10)
	doc.SetTextColor(muted[0], muted[1], muted[2])
	doc.CellFormat(0, 5, tr(fmt.Sprintf("%s to %s", s.PeriodStart, s.PeriodEnd)), "", 1, "L", false, 0, "")
	doc.Ln(4)

	pair := func(label, value string) {
		doc.SetFont("Helvetica", "", 9)
		doc.SetTextColor(muted[0], muted[1], muted[2])
		doc.CellFormat(28, 5, label, "", 0, "L", false, 0, "")
		doc.SetFont("Helvetica", "B", 10)
		doc.SetTextColor(ink[0], ink[1], ink[2])
		doc.CellFormat(0, 5, tr(value), "", 1, "L", false, 0, "")
	}
	pair("Property", s.PropertyName)
	if s.PropertyAddress != "" {
		pair("Address", s.PropertyAddress)
	}
	pair("Owner", s.OwnerName)
	pair("Generated", s.GeneratedAt.Format(time.DateOnly))
	doc.Ln(6)

	section := func(title string, items []domain.StatementLineItem, total int64) {
		doc.SetFont("Helvetica", "B", 12)
		doc.SetTextColor(ink[0], ink[1], ink[2])
		doc.CellFormat(0, 7, title, "", 1, "L", false, 0, "")

		doc.SetFillColor(247, 248, 250)
		doc.SetFont("Helvetica", "B", 8)
		doc.SetTextColor(muted[0], muted[1], muted[2])
		doc.CellFormat(colDate, 6, "DATE", "", 0, "L", true, 0, "")
		doc.CellFormat(descWidth, 6, "DESCRIPTION", "", 0, "L", true, 0, "")
		doc.CellFormat(colAmount, 6, "AMOUNT", "", 1, "R", true, 0, "")

		doc.SetDrawColor(line[0], line[1], line[2])
		doc.SetFont("Helvetica", "", 9)
		doc.SetTextColor(ink[0], ink[1], ink[2])
		if len(items) == 0 {
			doc.SetTextColor(muted[0], muted[1], muted[2])
			doc.CellFormat(contentWidth, 7, "None in this period", "B", 1, "L", false, 0, "")
		}
		for _, it := range items {
			doc.CellFormat(colDate, 6.5, it.Date, "B", 0, "L", false, 0, "")
			desc := tr(it.Description)
			for doc.GetStringWidth(desc) > descWidth-2 && len(desc) > 4 {
				desc = desc[:len(desc)-4] + "..."
			}
			doc.CellFormat(descWidth, 6.5, desc, "B", 0, "L", false, 0, "")
			doc.CellFormat(colAmount, 6.5, formatCents(it.AmountCents), "B", 1, "R", false, 0, "")
		}
		doc.SetFont("Helvetica", "B", 10)
		doc.CellFormat(colDate+descWidth, 7, "Total", "", 0, "R", false, 0, "")
		doc.CellFormat(colAmount, 7, formatCents(total), "", 1, "R", false, 0, "")
		doc.Ln(4)
	}
	section("Income collected", s.Income, s.IncomeCents)
	section("Expenses paid", s.Expenses, s.ExpensesCents)

	summary := func(label string, cents int64, bold bool) {
		style := ""
		if bold {
			style = "B"
			doc.SetDrawColor(ink[0], ink[1], ink[2])
		}
		doc.SetFont("Helvetica", style, 10)
		doc.SetTextColor(ink[0], ink[1], ink[2])
		border := ""
		if bold {
			border = "T"
		}
		doc.CellFormat(colDate+descWidth, 7, label, border, 0, "R", false, 0, "")
		doc.CellFormat(colAmount, 7, formatCents(cents), border, 1, "R", false, 0, "")
	}
	summary("Income collected", s.IncomeCents, false)
	summary("Expenses paid", -s.ExpensesCents, false)
	summary(fmt.Sprintf("Management fee (%.2f%% of income)", float64(s.FeeBps)/100), -s.FeeCents, false)
	summary("Net payout to owner", s.NetPayoutCents, true)

	var buf bytes.Buffer
	if err := doc.Output(&buf); err != nil {
		return nil, fmt.Errorf("render statement pdf: %w", err)
	}
	return buf.Bytes(), nil
}
