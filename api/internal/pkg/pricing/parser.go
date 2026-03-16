package pricing

import (
	"context"
	"fmt"
	"io"
	"regexp"
	"strconv"
	"strings"

	"golang.org/x/net/html"

	"thing-index/internal/domain"
)

var blockElements = map[string]struct{}{
	"div": {}, "section": {}, "article": {}, "p": {}, "br": {}, "tr": {}, "li": {},
	"h1": {}, "h2": {}, "h3": {}, "h4": {}, "h5": {}, "h6": {}, "table": {},
}

type Registry struct {
	parsers map[string]domain.PricingParser
}

func NewRegistry(parsers ...domain.PricingParser) *Registry {
	items := make(map[string]domain.PricingParser, len(parsers))
	for _, parser := range parsers {
		items[strings.ToUpper(strings.TrimSpace(parser.ProviderType()))] = parser
	}
	return &Registry{parsers: items}
}

func (r *Registry) ParserFor(providerType string) (domain.PricingParser, bool) {
	parser, ok := r.parsers[strings.ToUpper(strings.TrimSpace(providerType))]
	return parser, ok
}

type linePatternParser struct {
	providerType  string
	modelMatchers []*regexp.Regexp
}

type openAIParser struct{}

type ollamaPlanParser struct{}

func NewLinePatternParser(providerType string, modelMatchers ...*regexp.Regexp) domain.PricingParser {
	return &linePatternParser{
		providerType:  providerType,
		modelMatchers: modelMatchers,
	}
}

func (p *linePatternParser) ProviderType() string {
	return p.providerType
}

func (p *linePatternParser) Parse(_ context.Context, _ string, body io.Reader) ([]domain.ParsedModelPrice, error) {
	lines, err := extractTextLines(body)
	if err != nil {
		return nil, err
	}

	results := make([]domain.ParsedModelPrice, 0)
	for idx := 0; idx < len(lines); idx++ {
		modelName := p.matchModelLine(lines[idx])
		if modelName == "" {
			continue
		}

		parsed := domain.ParsedModelPrice{
			ModelKey:  slugify(modelName),
			ModelName: modelName,
			Unit:      "1M tokens",
			Currency:  "USD",
		}

		for scan := idx + 1; scan < len(lines) && scan <= idx+8; scan++ {
			line := lines[scan]
			lower := strings.ToLower(line)
			switch {
			case strings.Contains(lower, "cached input"):
				parsed.CachedInputCostPerMillionUSD = parsePriceUSD(line)
			case strings.Contains(lower, "input"):
				parsed.InputCostPerMillionUSD = parsePriceUSD(line)
			case strings.Contains(lower, "output"):
				parsed.OutputCostPerMillionUSD = parsePriceUSD(line)
			case strings.Contains(lower, "context"):
				parsed.ContextCostPerMillionUSD = parsePriceUSD(line)
			}

			if parsed.PriceNotes == "" && strings.Contains(lower, "token") {
				parsed.PriceNotes = line
			}
		}

		if parsed.InputCostPerMillionUSD == 0 &&
			parsed.OutputCostPerMillionUSD == 0 &&
			parsed.CachedInputCostPerMillionUSD == 0 &&
			parsed.ContextCostPerMillionUSD == 0 {
			continue
		}

		results = append(results, parsed)
	}

	if len(results) == 0 {
		return nil, fmt.Errorf("no pricing rows parsed for provider %s", p.providerType)
	}

	return dedupeByModel(results), nil
}

func (p *linePatternParser) matchModelLine(line string) string {
	for _, matcher := range p.modelMatchers {
		if matcher.MatchString(line) {
			return strings.TrimSpace(line)
		}
	}
	return ""
}

func DefaultParsers() *Registry {
	return NewRegistry(
		&openAIParser{},
		NewLinePatternParser(domain.ProviderDeepSeek,
			regexp.MustCompile(`(?i)^deepseek[- ][a-z0-9 ._-]+$`),
		),
		NewLinePatternParser(domain.ProviderGemini,
			regexp.MustCompile(`(?i)^gemini[ a-z0-9._-]+$`),
		),
		&ollamaPlanParser{},
	)
}

func (p *openAIParser) ProviderType() string {
	return domain.ProviderOpenAI
}

func (p *openAIParser) Parse(_ context.Context, _ string, body io.Reader) ([]domain.ParsedModelPrice, error) {
	lines, err := extractTextLines(body)
	if err != nil {
		return nil, err
	}

	rowPattern := regexp.MustCompile(`(?i)^([a-z0-9][a-z0-9._-]+)\$`)
	results := make([]domain.ParsedModelPrice, 0)
	for _, line := range lines {
		match := rowPattern.FindStringSubmatch(line)
		if len(match) != 2 {
			continue
		}
		modelName := match[1]
		values := parseAllPricesUSD(line)
		if len(values) < 2 {
			continue
		}

		item := domain.ParsedModelPrice{
			ModelKey:  slugify(modelName),
			ModelName: modelName,
			Unit:      "1M tokens",
			Currency:  "USD",
		}

		switch {
		case len(values) >= 4:
			item.InputCostPerMillionUSD = values[len(values)-3]
			item.CachedInputCostPerMillionUSD = values[len(values)-2]
			item.OutputCostPerMillionUSD = values[len(values)-1]
		case len(values) == 3:
			item.InputCostPerMillionUSD = values[0]
			item.CachedInputCostPerMillionUSD = values[1]
			item.OutputCostPerMillionUSD = values[2]
		case len(values) == 2:
			item.InputCostPerMillionUSD = values[0]
			item.OutputCostPerMillionUSD = values[1]
		}

		if len(values) > 3 {
			item.PriceNotes = "Linha com colunas adicionais; mantidos os tres ultimos valores como input/cached/output."
		}
		results = append(results, item)
	}

	if len(results) == 0 {
		return nil, fmt.Errorf("no pricing rows parsed for provider %s", domain.ProviderOpenAI)
	}
	return dedupeByModel(results), nil
}

func (p *ollamaPlanParser) ProviderType() string {
	return domain.ProviderOllama
}

func (p *ollamaPlanParser) Parse(_ context.Context, _ string, body io.Reader) ([]domain.ParsedModelPrice, error) {
	lines, err := extractTextLines(body)
	if err != nil {
		return nil, err
	}

	plans := []string{"Free", "Pro", "Max"}
	results := make([]domain.ParsedModelPrice, 0, len(plans))
	for idx := 0; idx < len(lines); idx++ {
		line := lines[idx]
		if !containsExact(plans, line) {
			continue
		}

		for scan := idx + 1; scan < len(lines) && scan <= idx+4; scan++ {
			price := parsePriceUSD(lines[scan])
			if price == 0 && !strings.Contains(lines[scan], "$0") {
				continue
			}
			results = append(results, domain.ParsedModelPrice{
				ModelKey:     slugify(line),
				ModelName:    line,
				Unit:         "monthly plan",
				Currency:     "USD",
				FlatPriceUSD: price,
				PriceNotes:   "Ollama pricing page currently exposes subscription plans instead of per-model token prices.",
			})
			break
		}
	}

	if len(results) == 0 {
		return nil, fmt.Errorf("no pricing rows parsed for provider %s", domain.ProviderOllama)
	}
	return results, nil
}

func extractTextLines(body io.Reader) ([]string, error) {
	doc, err := html.Parse(body)
	if err != nil {
		return nil, err
	}

	var builder strings.Builder
	var walk func(*html.Node)
	walk = func(node *html.Node) {
		if node.Type == html.TextNode {
			text := strings.TrimSpace(node.Data)
			if text != "" {
				builder.WriteString(text)
				builder.WriteByte('\n')
			}
		}
		if node.Type == html.ElementNode {
			if _, ok := blockElements[node.Data]; ok {
				builder.WriteByte('\n')
			}
		}
		for child := node.FirstChild; child != nil; child = child.NextSibling {
			walk(child)
		}
	}
	walk(doc)

	rawLines := strings.Split(builder.String(), "\n")
	lines := make([]string, 0, len(rawLines))
	for _, line := range rawLines {
		cleaned := strings.Join(strings.Fields(strings.TrimSpace(line)), " ")
		if cleaned != "" {
			lines = append(lines, cleaned)
		}
	}
	return lines, nil
}

var usdPattern = regexp.MustCompile(`\$?\s*([0-9]+(?:\.[0-9]+)?)`)

func parsePriceUSD(line string) float64 {
	match := usdPattern.FindStringSubmatch(line)
	if len(match) != 2 {
		return 0
	}
	value, err := strconv.ParseFloat(match[1], 64)
	if err != nil {
		return 0
	}
	return value
}

func parseAllPricesUSD(line string) []float64 {
	matches := usdPattern.FindAllStringSubmatch(line, -1)
	values := make([]float64, 0, len(matches))
	for _, match := range matches {
		if len(match) != 2 {
			continue
		}
		value, err := strconv.ParseFloat(match[1], 64)
		if err != nil {
			continue
		}
		values = append(values, value)
	}
	return values
}

func slugify(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	replacer := strings.NewReplacer(" ", "-", "/", "-", "_", "-", ".", "-", ":", "-")
	value = replacer.Replace(value)
	value = strings.Trim(value, "-")
	return value
}

func dedupeByModel(items []domain.ParsedModelPrice) []domain.ParsedModelPrice {
	seen := make(map[string]struct{}, len(items))
	result := make([]domain.ParsedModelPrice, 0, len(items))
	for _, item := range items {
		key := strings.TrimSpace(item.ModelKey)
		if key == "" {
			continue
		}
		if _, exists := seen[key]; exists {
			continue
		}
		seen[key] = struct{}{}
		result = append(result, item)
	}
	return result
}

func containsExact(values []string, candidate string) bool {
	for _, value := range values {
		if value == candidate {
			return true
		}
	}
	return false
}
