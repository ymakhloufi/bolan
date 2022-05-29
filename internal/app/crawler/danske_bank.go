package crawler

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	"cloud.google.com/go/civil"
	"github.com/antchfx/htmlquery"
	"github.com/ymakhloufi/bolan-compare/internal/pkg/model"
	"go.uber.org/zap"
	"golang.org/x/net/html"
)

const (
	danskeBankUrl             = "https://danskebank.se/privat/produkter/bolan/relaterat/aktuella-bolanerantor"
	DanskeBankName model.Bank = "Danske Bank"
)

var _ SiteCrawler = &DanskeBankCrawler{}

type DanskeBankCrawler struct {
	logger *zap.Logger
}

func NewDanskeBankCrawler(logger *zap.Logger) *DanskeBankCrawler {
	return &DanskeBankCrawler{logger: logger}
}

func (d DanskeBankCrawler) Crawl(channel chan<- model.InterestSet) {
	crawlTime := time.Now()
	doc, err := htmlquery.LoadURL(danskeBankUrl)
	if err != nil {
		d.logger.Error("failed reading Danske Bank website", zap.Error(err))
		return
	}
	d.logger.Debug("parsed root nodes")

	// this is the shaky part. If they change anything on their website structure, this is most likely gonna fail here
	nodes, err := htmlquery.QueryAll(doc, "//table/tbody/tr/td/b[contains(text(), 'Belåningsgrad')]]")
	if err != nil {
		d.logger.Error("failed to xpath table with interest rates", zap.Error(err))
		return
	}
	if len(nodes) != 2 { // regular and union-subsidized
		d.logger.Error("failed to find both tables with interest rates", zap.Error(err))
		return
	}
	d.logger.Debug("found relevant tables")

	// todo: get list prices
	// todo: get historic average prices
	// todo: get historic list prices

	interestSets, err := d.parseTable(nodes[0].Parent.Parent.Parent.Parent, crawlTime)
	if err != nil {
		d.logger.Error("failed to parse TableNodes", zap.Error(err))
	}
	d.logger.Debug("parsed interestSets", zap.Any("rows", interestSets))

	unionOfferTable, err := d.parseTable(nodes[1].Parent.Parent.Parent.Parent, crawlTime)
	if err != nil {
		d.logger.Error("failed to parse TableNodes", zap.Error(err))
	}
	d.logger.Debug("parsed interestSets", zap.Any("rows", unionOfferTable))

	for _, set := range append(interestSets, unionOfferTable...) {
		channel <- set
	}

}

func (d DanskeBankCrawler) parseTable(table *html.Node, crawlTime time.Time) ([]model.InterestSet, error) {
	rowNodes, err := htmlquery.QueryAll(table, "//tr")
	if err != nil {
		return nil, fmt.Errorf("failed to xpath rows: %w", err)
	}

	rows, err := parseRows(rowNodes)
	if err != nil {
		return nil, fmt.Errorf("failed to parse rows: %w", err)
	}

	if !strings.Contains(rows[0].title, "Belåningsgrad") {
		return nil, fmt.Errorf("source table structure seems to have changed... fix parser?")
	}

	discountBoundaries, err := parseDiscountBoundaries(rows[0].fields)
	if err != nil {
		return nil, fmt.Errorf("failed to extract categories from first rowStruct: %w", err)
	}

	sets := make([]model.InterestSet, 0, len(rows)-1)
	for _, row := range rows[1:] {
		term, err := parseTerm(row.title)
		if err != nil {
			return nil, fmt.Errorf("failed to parse term for row %v: %w", row, err)
		}
		for i, cell := range row.fields {
			nominalRate, effectiveRate, err := parseInterestRatesFromCellText(cell)
			if err != nil {
				return nil, fmt.Errorf("failed to parse interest rates for row %v: %w", row, err)
			}
			sets = append(sets, model.InterestSet{
				Bank:                    DanskeBankName,
				NominalRate:             nominalRate,
				EffectiveRate:           effectiveRate,
				Term:                    term,
				Type:                    model.TypeRatioDiscounted,
				RatioDiscountBoundaries: &discountBoundaries[i],
				UnionDiscount:           false,
				ChangedOn:               civil.DateOf(time.Now()), //todo: read from list-price-table[term]
				LastCrawledAt:           crawlTime,
			})
		}
	}

	return sets, nil
}

func parseInterestRatesFromCellText(cell string) (nominal float32, effective float32, err error) {
	sanitized := regexp.MustCompile(`\s+`).ReplaceAllString(cell, " ")
	sanitized = strings.ReplaceAll(sanitized, "%", "")
	sanitized = strings.ReplaceAll(sanitized, "*", "")
	sanitized = strings.ReplaceAll(sanitized, "(", "")
	sanitized = strings.ReplaceAll(sanitized, ")", "")
	sanitized = strings.TrimSpace(sanitized)

	parts := strings.Split(sanitized, " ")
	nominalStr := strings.Replace(parts[0], ",", ".", -1)
	nominal64, err := strconv.ParseFloat(nominalStr, 32)
	if err != nil {
		return 0, 0, fmt.Errorf("failed to parse nominal float for string '%s' in cell '%s' (sanitized: '%s'): %w", nominalStr, cell, sanitized, err)
	}

	effectiveStr := strings.Replace(parts[0], ",", ".", -1)
	effective64, err := strconv.ParseFloat(effectiveStr, 32)
	if err != nil {
		return 0, 0, fmt.Errorf("failed to parse effective float for string '%s' in cell '%s' (sanitized: '%s'): %w", effectiveStr, cell, sanitized, err)
	}

	return float32(nominal64), float32(effective64), nil
}

func parseDiscountBoundaries(fields []string) ([]model.RatioDiscountBoundary, error) {
	out := make([]model.RatioDiscountBoundary, 0, len(fields))
	for _, cat := range fields {
		cleanedCat := regexp.MustCompile(`\s+`).ReplaceAllString(cat, "") // remove spaces
		switch cleanedCat {
		case "60%":
			out = append(out, model.RatioDiscountBoundary{MinRatio: 0, MaxRatio: 0.6})
		case "61-74%":
			out = append(out, model.RatioDiscountBoundary{MinRatio: 0.6, MaxRatio: 0.75})
		case "75-79%":
			out = append(out, model.RatioDiscountBoundary{MinRatio: 0.75, MaxRatio: 0.8})
		case "80-85%":
			out = append(out, model.RatioDiscountBoundary{MinRatio: 0.8, MaxRatio: 0.85})
		default:
			return nil, fmt.Errorf("failed to parse Discount Boundary from string '%s' (sanitized: '%s')", cat, cleanedCat)
		}
	}
	return out, nil
}

func parseTerm(title string) (model.Term, error) {
	cleanedTitle := regexp.MustCompile(`\s+`).ReplaceAllString(title, "") // remove spaces
	switch cleanedTitle {
	case "3mån":
		return model.Term3months, nil
	case "1år":
		return model.Term1year, nil
	case "2år":
		return model.Term2years, nil
	case "3år":
		return model.Term3years, nil
	case "4år":
		return model.Term4years, nil
	case "5år":
		return model.Term5years, nil
	case "6år":
		return model.Term6years, nil
	case "7år":
		return model.Term7years, nil
	case "8år":
		return model.Term8years, nil
	case "9år":
		return model.Term9years, nil
	case "10år":
		return model.Term10years, nil
	default:
		return "", fmt.Errorf("failed to parse term from string '%s' (sanitized: '%s')", title, cleanedTitle)
	}
}

func parseRows(rows []*html.Node) ([]rowStruct, error) {
	rowStructs := make([]rowStruct, 0, len(rows))
	for _, rowNode := range rows {
		cells, err := htmlquery.QueryAll(rowNode, "//td")
		if err != nil {
			return nil, fmt.Errorf("failed to xpath cells: %w", err)
		}

		titleCellText := getAllTextFromNode(cells[0])
		fieldTexts := make([]string, 0, len(cells)-1)
		for _, cell := range cells[1:] {
			fieldTexts = append(fieldTexts, getAllTextFromNode(cell))
		}

		rowStructs = append(rowStructs, rowStruct{
			title:  titleCellText,
			fields: fieldTexts,
		})
	}

	return rowStructs, nil
}

func getAllTextFromNode(node *html.Node) string {
	out := ""
	if node != nil {
		if node.Type == html.TextNode {
			out += " " + node.Data
		}

		// iterate over children
		nextNode := node.FirstChild
		for nextNode != nil {
			out += " " + getAllTextFromNode(nextNode)
			nextNode = nextNode.NextSibling
		}
	}

	// sanitize text
	out = strings.ReplaceAll(out, " ", " ")                    // weird invisible space that's not a space
	out = regexp.MustCompile(`\s+`).ReplaceAllString(out, " ") // merge multi-spaces
	out = strings.Trim(out, " ")                               // trim spaces left and right
	return out
}

type rowStruct struct {
	title  string
	fields []string
}
