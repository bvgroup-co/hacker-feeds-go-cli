package feeds

import (
	"encoding/json"
	"encoding/xml"
	"fmt"
	"html"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"
)

const (
	defaultProductBase      = "https://www.producthunt.com/feed"
	defaultProductWebBase   = "https://www.producthunt.com"
	defaultProductUserAgent = "hacker-feeds-go-cli/dev (+https://github.com/bvgroup-co/hacker-feeds-go-cli)"
	productHuntFeedSource   = "producthunt-feed"
	productHuntPageSource   = "producthunt-public-page"
)

var (
	productHTMLTagPattern = regexp.MustCompile(`<[^>]*>`)
	productParagraphRegex = regexp.MustCompile(`(?is)<p\b[^>]*>(.*?)</p>`)
	productLinkRegex      = regexp.MustCompile(`(?is)<a\b([^>]*)>(.*?)</a>`)
	productJSONLDRegex    = regexp.MustCompile(`(?is)<script\b[^>]*type\s*=\s*["']application/ld\+json["'][^>]*>(.*?)</script>`)
	productTitleRegex     = regexp.MustCompile(`(?is)<title\b[^>]*>(.*?)</title>`)
	productMetaRegex      = regexp.MustCompile(`(?is)<meta\b([^>]*)>`)
	productLinkTagRegex   = regexp.MustCompile(`(?is)<link\b([^>]*)>`)
	productURLSlugPattern = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9-]*$`)
	productDataKeyPattern = regexp.MustCompile(`"(product|launch|makers|topics|media|latestScore|launchDayScore|hideVotesCount|commentsCount|followersCount|reviewsCount|reviewsRating|scheduledAt|featuredAt)"\s*:`)
)

type productAtomFeed struct {
	XMLName xml.Name           `xml:"feed"`
	Entries []productAtomEntry `xml:"entry"`
}

type productAtomEntry struct {
	Title     string            `xml:"title"`
	Published string            `xml:"published"`
	Links     []productAtomLink `xml:"link"`
	Content   struct {
		Body string `xml:",innerxml"`
	} `xml:"content"`
}

type productAtomLink struct {
	Rel  string `xml:"rel,attr"`
	Href string `xml:"href,attr"`
}

type productJSONLD struct {
	Name            string          `json:"name"`
	Description     string          `json:"description"`
	URL             string          `json:"url"`
	Image           productLDImages `json:"image"`
	Screenshot      productLDImages `json:"screenshot"`
	Author          productLDPeople `json:"author"`
	AggregateRating productLDRating `json:"aggregateRating"`
}

type productLDImages []string

type productLDPeople []ProductMaker

type productLDRating struct {
	RatingValue string `json:"ratingValue"`
	ReviewCount string `json:"reviewCount"`
	RatingCount string `json:"ratingCount"`
}

type productMeta struct {
	Title       string
	Description string
	Canonical   string
	Image       string
}

func (client Client) FetchProducts(count int, past int, now time.Time) ([]Product, error) {
	if count <= 0 {
		return nil, fmt.Errorf("count must be greater than 0")
	}
	if past < 0 {
		return nil, fmt.Errorf("past must be non-negative")
	}
	req, err := http.NewRequest(http.MethodGet, client.ProductBase, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/atom+xml, application/xml, text/xml")
	req.Header.Set("User-Agent", client.productUserAgent())
	body, err := client.do(req)
	if err != nil {
		return nil, err
	}
	products, err := parseProductAtom(body, past, now)
	if err != nil {
		return nil, err
	}
	if len(products) > count {
		return products[:count], nil
	}
	return products, nil
}

func (client Client) FetchProductDetails(input ProductDetailsInput) (ProductDetails, error) {
	path, slug, err := productDetailsPath(input)
	if err != nil {
		return ProductDetails{}, err
	}
	base := strings.TrimRight(valueOrDefault(client.ProductWebBase, defaultProductWebBase), "/")
	req, err := http.NewRequest(http.MethodGet, base+path, nil)
	if err != nil {
		return ProductDetails{}, err
	}
	req.Header.Set("Accept", "text/html, application/xhtml+xml")
	req.Header.Set("User-Agent", client.productUserAgent())
	body, err := client.do(req)
	if err != nil {
		return ProductDetails{}, err
	}
	details := parseProductDetailsPage(body, base, slug)
	if details.ProductURL == "" {
		details.ProductURL = base + path
	}
	if details.Slug == "" {
		details.Slug = slug
	}
	return details, nil
}

func NormalizeProductDetailsSlug(slug string) (string, error) {
	trimmed := strings.TrimSpace(slug)
	if trimmed == "" {
		return "", fmt.Errorf("--slug is required")
	}
	if strings.Contains(trimmed, "/") || strings.Contains(trimmed, "?") || strings.Contains(trimmed, "#") {
		return "", fmt.Errorf("--slug accepts only a Product Hunt slug")
	}
	if !productURLSlugPattern.MatchString(trimmed) {
		return "", fmt.Errorf("--slug accepts only letters, numbers, and hyphens")
	}
	return trimmed, nil
}

func NormalizeProductDetailsURL(value string) (string, string, error) {
	parsed, err := url.Parse(strings.TrimSpace(value))
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return "", "", fmt.Errorf("--url must be a Product Hunt product or post URL")
	}
	if parsed.Scheme != "https" && parsed.Scheme != "http" {
		return "", "", fmt.Errorf("--url must be an http or https Product Hunt URL")
	}
	host := strings.ToLower(parsed.Hostname())
	if host != "producthunt.com" && host != "www.producthunt.com" {
		return "", "", fmt.Errorf("--url must be a Product Hunt URL")
	}
	segments := pathSegments(parsed.Path)
	if len(segments) < 2 {
		return "", "", fmt.Errorf("--url must include /products/{slug} or /posts/{slug}")
	}
	if segments[0] != "products" && segments[0] != "posts" {
		return "", "", fmt.Errorf("--url must be a Product Hunt product or post URL")
	}
	slug, err := NormalizeProductDetailsSlug(segments[1])
	if err != nil {
		return "", "", err
	}
	return "/" + segments[0] + "/" + slug, slug, nil
}

func parseProductAtom(body []byte, past int, now time.Time) ([]Product, error) {
	var feed productAtomFeed
	if err := xml.Unmarshal(body, &feed); err != nil {
		return nil, err
	}
	products := make([]Product, 0, len(feed.Entries))
	for _, entry := range feed.Entries {
		if !productPublishedWithinPast(entry.Published, past, now) {
			continue
		}
		products = append(products, Product{
			Name:        strings.TrimSpace(html.UnescapeString(entry.Title)),
			Description: firstProductParagraph(entry.Content.Body),
			URL:         stripQuery(productAlternateURL(entry.Links)),
			Website:     stripQuery(productWebsiteURL(entry.Content.Body)),
			VotesKnown:  false,
			Source:      productHuntFeedSource,
		})
	}
	return products, nil
}

func productDetailsPath(input ProductDetailsInput) (string, string, error) {
	hasURL := strings.TrimSpace(input.URL) != ""
	hasSlug := strings.TrimSpace(input.Slug) != ""
	if hasURL && hasSlug {
		return "", "", fmt.Errorf("--url and --slug are mutually exclusive")
	}
	if !hasURL && !hasSlug {
		return "", "", fmt.Errorf("one of --url or --slug is required")
	}
	if hasSlug {
		slug, err := NormalizeProductDetailsSlug(input.Slug)
		if err != nil {
			return "", "", err
		}
		return "/products/" + slug, slug, nil
	}
	return NormalizeProductDetailsURL(input.URL)
}

func parseProductDetailsPage(body []byte, base string, requestedSlug string) ProductDetails {
	content := string(body)
	details := ProductDetails{Slug: requestedSlug, Source: productHuntPageSource}
	applyProductJSONLD(&details, content)
	applyProductEmbeddedData(&details, content)
	applyProductMeta(&details, content)
	finalizeProductDetails(&details, base)
	return details
}

func applyProductJSONLD(details *ProductDetails, content string) {
	for _, match := range productJSONLDRegex.FindAllStringSubmatch(content, -1) {
		var raw any
		if json.Unmarshal([]byte(html.UnescapeString(strings.TrimSpace(match[1]))), &raw) != nil {
			continue
		}
		for _, object := range flattenProductJSONLD(raw) {
			var item productJSONLD
			encoded, err := json.Marshal(object)
			if err != nil || json.Unmarshal(encoded, &item) != nil {
				continue
			}
			fillString(&details.Name, item.Name)
			fillString(&details.Description, item.Description)
			fillString(&details.ProductURL, item.URL)
			for _, image := range append(item.Image, item.Screenshot...) {
				addProductMedia(details, image, "image")
			}
			for _, maker := range item.Author {
				addProductMaker(details, maker)
			}
			if details.ReviewsCount == 0 {
				details.ReviewsCount = firstInt(item.AggregateRating.ReviewCount, item.AggregateRating.RatingCount)
			}
			if details.ReviewsRating == 0 {
				details.ReviewsRating = parseFloatString(item.AggregateRating.RatingValue)
			}
		}
	}
}

func applyProductEmbeddedData(details *ProductDetails, content string) {
	for _, start := range productDataObjectStarts(content) {
		object, ok := balancedJSONObject(content, start)
		if !ok {
			continue
		}
		var raw any
		if json.Unmarshal([]byte(object), &raw) != nil {
			continue
		}
		applyProductDataValue(details, raw)
	}
}

func applyProductMeta(details *ProductDetails, content string) {
	meta := parseProductMeta(content)
	fillString(&details.Name, productNameFromTitle(meta.Title))
	fillString(&details.Tagline, meta.Description)
	fillString(&details.Description, meta.Description)
	fillString(&details.ProductURL, meta.Canonical)
	addProductMedia(details, meta.Image, "image")
}

func applyProductDataValue(details *ProductDetails, value any) {
	switch typed := value.(type) {
	case map[string]any:
		applyProductDataMap(details, typed)
		for _, nested := range typed {
			applyProductDataValue(details, nested)
		}
	case []any:
		for _, nested := range typed {
			applyProductDataValue(details, nested)
		}
	}
}

func applyProductDataMap(details *ProductDetails, data map[string]any) {
	if product, ok := data["product"].(map[string]any); ok {
		applyProductObject(details, product)
	}
	if launch, ok := data["launch"].(map[string]any); ok {
		applyProductLaunch(details, launch)
	}
	if makers, ok := data["makers"].([]any); ok {
		applyProductMakers(details, makers)
	}
	if topics, ok := data["topics"].(map[string]any); ok {
		applyProductTopics(details, topics)
	}
	if media, ok := data["media"].([]any); ok {
		applyProductMediaList(details, media)
	}
	fillInt(&details.CommentsCount, data["commentsCount"])
	fillInt(&details.FollowersCount, data["followersCount"])
	fillInt(&details.ReviewsCount, data["reviewsCount"])
	fillFloat(&details.ReviewsRating, data["reviewsRating"])
	fillString(&details.PublishedAt, stringValue(data["featuredAt"]))
	fillString(&details.PublishedAt, stringValue(data["scheduledAt"]))
	fillString(&details.UpdatedAt, stringValue(data["updatedAt"]))
	if boolValue(data["hideVotesCount"]) {
		details.VotesKnown = false
		details.Votes = 0
		return
	}
	votes := 0
	if intValue(data["latestScore"], &votes) || intValue(data["launchDayScore"], &votes) {
		details.Votes = votes
		details.VotesKnown = true
	}
}

func applyProductObject(details *ProductDetails, product map[string]any) {
	fillString(&details.Name, stringValue(product["name"]))
	fillString(&details.Slug, stringValue(product["slug"]))
	fillString(&details.WebsiteURL, stringValue(product["websiteUrl"]))
	fillString(&details.CleanDomain, stringValue(product["cleanUrl"]))
	fillString(&details.CleanDomain, stringValue(product["cleanDomain"]))
}

func applyProductLaunch(details *ProductDetails, launch map[string]any) {
	fillString(&details.LaunchName, stringValue(launch["name"]))
	fillString(&details.Tagline, stringValue(launch["tagline"]))
	if description := stringValue(launch["description"]); description != "" {
		details.Description = description
	}
	fillString(&details.PublishedAt, stringValue(launch["featuredAt"]))
	fillString(&details.PublishedAt, stringValue(launch["scheduledAt"]))
	if boolValue(launch["hideVotesCount"]) {
		details.VotesKnown = false
		details.Votes = 0
		return
	}
	votes := 0
	if intValue(launch["latestScore"], &votes) || intValue(launch["launchDayScore"], &votes) {
		details.Votes = votes
		details.VotesKnown = true
	}
}

func applyProductMakers(details *ProductDetails, makers []any) {
	for _, makerValue := range makers {
		makerMap, ok := makerValue.(map[string]any)
		if !ok {
			continue
		}
		maker := ProductMaker{Name: stringValue(makerMap["name"]), Username: stringValue(makerMap["username"]), URL: stringValue(makerMap["url"])}
		addProductMaker(details, maker)
	}
}

func applyProductTopics(details *ProductDetails, topics map[string]any) {
	edges, ok := topics["edges"].([]any)
	if !ok {
		return
	}
	for _, edgeValue := range edges {
		edge, ok := edgeValue.(map[string]any)
		if !ok {
			continue
		}
		node, ok := edge["node"].(map[string]any)
		if !ok {
			continue
		}
		addProductTopic(details, ProductTopic{Name: stringValue(node["name"]), Slug: stringValue(node["slug"])})
	}
}

func applyProductMediaList(details *ProductDetails, media []any) {
	for _, mediaValue := range media {
		mediaMap, ok := mediaValue.(map[string]any)
		if !ok {
			continue
		}
		urlValue := firstString(mediaMap, "url", "imageUrl", "videoUrl")
		kind := firstString(mediaMap, "type", "kind")
		addProductMedia(details, urlValue, kind)
	}
}

func finalizeProductDetails(details *ProductDetails, base string) {
	if details.ProductURL != "" {
		details.ProductURL = absoluteProductURL(details.ProductURL, base)
	} else if details.Slug != "" {
		details.ProductURL = base + "/products/" + details.Slug
	}
	if details.WebsiteURL != "" {
		details.WebsiteURL = html.UnescapeString(details.WebsiteURL)
	}
	if details.CleanDomain == "" && details.WebsiteURL != "" {
		details.CleanDomain = cleanDomain(details.WebsiteURL)
	}
	if details.Source == "" {
		details.Source = productHuntPageSource
	}
}

func flattenProductJSONLD(value any) []any {
	switch typed := value.(type) {
	case []any:
		var flattened []any
		for _, item := range typed {
			flattened = append(flattened, flattenProductJSONLD(item)...)
		}
		return flattened
	case map[string]any:
		if graph, ok := typed["@graph"]; ok {
			return flattenProductJSONLD(graph)
		}
		return []any{typed}
	default:
		return nil
	}
}

func (images *productLDImages) UnmarshalJSON(data []byte) error {
	var single string
	if json.Unmarshal(data, &single) == nil {
		*images = []string{single}
		return nil
	}
	var many []string
	if json.Unmarshal(data, &many) == nil {
		*images = many
		return nil
	}
	return nil
}

func (people *productLDPeople) UnmarshalJSON(data []byte) error {
	var single ProductMaker
	if json.Unmarshal(data, &single) == nil && (single.Name != "" || single.URL != "") {
		*people = []ProductMaker{single}
		return nil
	}
	var many []ProductMaker
	if json.Unmarshal(data, &many) == nil {
		*people = many
		return nil
	}
	return nil
}

func productDataObjectStarts(content string) []int {
	matches := productDataKeyPattern.FindAllStringIndex(content, -1)
	starts := make([]int, 0, len(matches))
	seen := make(map[int]bool, len(matches))
	for _, match := range matches {
		start := strings.LastIndex(content[:match[0]], "{")
		if start == -1 || seen[start] {
			continue
		}
		seen[start] = true
		starts = append(starts, start)
	}
	return starts
}

func balancedJSONObject(content string, start int) (string, bool) {
	if start < 0 || start >= len(content) || content[start] != '{' {
		return "", false
	}
	inString := false
	escaped := false
	depth := 0
	for index := start; index < len(content); index++ {
		char := content[index]
		if inString {
			if escaped {
				escaped = false
				continue
			}
			if char == '\\' {
				escaped = true
				continue
			}
			if char == '"' {
				inString = false
			}
			continue
		}
		if char == '"' {
			inString = true
			continue
		}
		if char == '{' {
			depth++
			continue
		}
		if char == '}' {
			depth--
			if depth == 0 {
				return content[start : index+1], true
			}
		}
	}
	return "", false
}

func parseProductMeta(content string) productMeta {
	meta := productMeta{}
	if match := productTitleRegex.FindStringSubmatch(content); len(match) != 0 {
		meta.Title = plainProductHTML(match[1])
	}
	for _, match := range productMetaRegex.FindAllStringSubmatch(content, -1) {
		attrs := parseHTMLAttrs(match[1])
		name := strings.ToLower(firstAttr(attrs, "name", "property"))
		value := firstAttr(attrs, "content")
		switch name {
		case "description", "og:description", "twitter:description":
			fillString(&meta.Description, value)
		case "og:title", "twitter:title":
			fillString(&meta.Title, value)
		case "og:image", "twitter:image":
			fillString(&meta.Image, value)
		case "og:url":
			fillString(&meta.Canonical, value)
		}
	}
	for _, match := range productLinkTagRegex.FindAllStringSubmatch(content, -1) {
		attrs := parseHTMLAttrs(match[1])
		if strings.ToLower(attrs["rel"]) == "canonical" {
			fillString(&meta.Canonical, attrs["href"])
		}
	}
	return meta
}

func productNameFromTitle(title string) string {
	name := strings.TrimSpace(title)
	for _, separator := range []string{" | Product Hunt", " - Product Hunt", " | ProductHunt", " - ProductHunt"} {
		name = strings.TrimSpace(strings.TrimSuffix(name, separator))
	}
	return name
}

func addProductMaker(details *ProductDetails, maker ProductMaker) {
	maker.Name = strings.TrimSpace(maker.Name)
	maker.Username = strings.TrimSpace(maker.Username)
	maker.URL = strings.TrimSpace(maker.URL)
	if maker.Name == "" && maker.Username == "" && maker.URL == "" {
		return
	}
	key := maker.Name + "\x00" + maker.Username + "\x00" + maker.URL
	for _, existing := range details.Makers {
		if existing.Name+"\x00"+existing.Username+"\x00"+existing.URL == key {
			return
		}
	}
	details.Makers = append(details.Makers, maker)
}

func addProductTopic(details *ProductDetails, topic ProductTopic) {
	topic.Name = strings.TrimSpace(topic.Name)
	topic.Slug = strings.TrimSpace(topic.Slug)
	if topic.Name == "" && topic.Slug == "" {
		return
	}
	for _, existing := range details.Topics {
		if existing.Name == topic.Name && existing.Slug == topic.Slug {
			return
		}
	}
	details.Topics = append(details.Topics, topic)
}

func addProductMedia(details *ProductDetails, urlValue string, kind string) {
	mediaURL := strings.TrimSpace(html.UnescapeString(urlValue))
	if mediaURL == "" {
		return
	}
	media := ProductMedia{URL: mediaURL, Type: strings.TrimSpace(kind)}
	for _, existing := range details.Media {
		if existing.URL == media.URL {
			return
		}
	}
	details.Media = append(details.Media, media)
}

func fillString(target *string, value string) {
	trimmed := strings.TrimSpace(html.UnescapeString(value))
	if *target == "" && trimmed != "" {
		*target = trimmed
	}
}

func fillInt(target *int, value any) {
	if *target != 0 {
		return
	}
	parsed := 0
	if intValue(value, &parsed) {
		*target = parsed
	}
}

func fillFloat(target *float64, value any) {
	if *target != 0 {
		return
	}
	if parsed := floatValue(value); parsed != 0 {
		*target = parsed
	}
}

func firstString(values map[string]any, keys ...string) string {
	for _, key := range keys {
		if value := stringValue(values[key]); value != "" {
			return value
		}
	}
	return ""
}

func stringValue(value any) string {
	switch typed := value.(type) {
	case string:
		return strings.TrimSpace(typed)
	case json.Number:
		return typed.String()
	case float64:
		return strconv.FormatFloat(typed, 'f', -1, 64)
	default:
		return ""
	}
}

func boolValue(value any) bool {
	typed, ok := value.(bool)
	return ok && typed
}

func intValue(value any, target *int) bool {
	switch typed := value.(type) {
	case float64:
		*target = int(typed)
		return true
	case json.Number:
		parsed, err := typed.Int64()
		if err != nil {
			return false
		}
		*target = int(parsed)
		return true
	case string:
		parsed, err := strconv.Atoi(strings.TrimSpace(strings.ReplaceAll(typed, ",", "")))
		if err != nil {
			return false
		}
		*target = parsed
		return true
	default:
		return false
	}
}

func firstInt(values ...string) int {
	for _, value := range values {
		parsed := 0
		if intValue(value, &parsed) {
			return parsed
		}
	}
	return 0
}

func floatValue(value any) float64 {
	switch typed := value.(type) {
	case float64:
		return typed
	case json.Number:
		parsed, err := typed.Float64()
		if err != nil {
			return 0
		}
		return parsed
	case string:
		return parseFloatString(typed)
	default:
		return 0
	}
}

func parseFloatString(value string) float64 {
	parsed, err := strconv.ParseFloat(strings.TrimSpace(value), 64)
	if err != nil {
		return 0
	}
	return parsed
}

func absoluteProductURL(value string, base string) string {
	parsed, err := url.Parse(strings.TrimSpace(value))
	if err != nil || parsed.String() == "" {
		return value
	}
	if parsed.IsAbs() {
		return stripQuery(parsed.String())
	}
	baseURL, err := url.Parse(base)
	if err != nil {
		return value
	}
	return stripQuery(baseURL.ResolveReference(parsed).String())
}

func cleanDomain(value string) string {
	parsed, err := url.Parse(value)
	if err != nil {
		return ""
	}
	return strings.TrimPrefix(parsed.Hostname(), "www.")
}

func pathSegments(path string) []string {
	parts := strings.Split(strings.Trim(path, "/"), "/")
	segments := make([]string, 0, len(parts))
	for _, part := range parts {
		if part != "" {
			segments = append(segments, part)
		}
	}
	return segments
}

func productPublishedWithinPast(published string, past int, now time.Time) bool {
	if past == 0 {
		return true
	}
	publishedAt, err := time.Parse(time.RFC3339, strings.TrimSpace(published))
	if err != nil {
		return true
	}
	start := now.AddDate(0, 0, -past)
	end := now.AddDate(0, 0, 1)
	return !publishedAt.Before(start) && publishedAt.Before(end)
}

func productAlternateURL(links []productAtomLink) string {
	for _, link := range links {
		if strings.TrimSpace(link.Rel) == "alternate" && strings.TrimSpace(link.Href) != "" {
			return strings.TrimSpace(link.Href)
		}
	}
	for _, link := range links {
		if strings.TrimSpace(link.Href) != "" {
			return strings.TrimSpace(link.Href)
		}
	}
	return ""
}

func firstProductParagraph(content string) string {
	decoded := html.UnescapeString(content)
	match := productParagraphRegex.FindStringSubmatch(decoded)
	if len(match) != 0 {
		return plainProductHTML(match[1])
	}
	match = productParagraphRegex.FindStringSubmatch(content)
	if len(match) == 0 {
		return plainProductHTML(content)
	}
	return plainProductHTML(match[1])
}

func productWebsiteURL(content string) string {
	decoded := html.UnescapeString(content)
	for _, match := range productLinkRegex.FindAllStringSubmatch(decoded, -1) {
		if len(match) < 3 {
			continue
		}
		label := strings.ToLower(plainProductHTML(match[2]))
		if label != "link" && label != "website" {
			continue
		}
		attrs := parseHTMLAttrs(match[1])
		if href := strings.TrimSpace(attrs["href"]); href != "" {
			return href
		}
	}
	return ""
}

func plainProductHTML(content string) string {
	unescaped := html.UnescapeString(content)
	withoutTags := productHTMLTagPattern.ReplaceAllString(unescaped, " ")
	return strings.Join(strings.Fields(html.UnescapeString(withoutTags)), " ")
}

func stripQuery(value string) string {
	parsed, err := url.Parse(value)
	if err != nil {
		return strings.Split(value, "?")[0]
	}
	parsed.RawQuery = ""
	parsed.Fragment = ""
	return parsed.String()
}

func (client Client) productUserAgent() string {
	return valueOrDefault(client.ProductUserAgent, defaultProductUserAgent)
}
