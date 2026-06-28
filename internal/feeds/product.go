package feeds

import (
	"encoding/json"
	"encoding/xml"
	"fmt"
	"html"
	"io"
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
	defaultProductPageAgent = "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/126.0.0.0 Safari/537.36"
	productHuntFeedSource   = "producthunt-feed"
	productHuntPageSource   = "producthunt-public-page"
	productCommentsPageCap  = 5
)

var (
	productHTMLTagPattern   = regexp.MustCompile(`<[^>]*>`)
	productParagraphRegex   = regexp.MustCompile(`(?is)<p\b[^>]*>(.*?)</p>`)
	productLinkRegex        = regexp.MustCompile(`(?is)<a\b([^>]*)>(.*?)</a>`)
	productJSONLDRegex      = regexp.MustCompile(`(?is)<script\b[^>]*type\s*=\s*["']application/ld\+json["'][^>]*>(.*?)</script>`)
	productTitleRegex       = regexp.MustCompile(`(?is)<title\b[^>]*>(.*?)</title>`)
	productMetaRegex        = regexp.MustCompile(`(?is)<meta\b([^>]*)>`)
	productLinkTagRegex     = regexp.MustCompile(`(?is)<link\b([^>]*)>`)
	productURLSlugPattern   = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9-]*$`)
	productDataKeyPattern   = regexp.MustCompile(`"(product|launch|makers|topics|media|latestScore|launchDayScore|hideVotesCount|commentsCount|followersCount|reviewsCount|reviewsRating|scheduledAt|featuredAt|__typename)"\s*:`)
	productApolloPushRegex  = regexp.MustCompile(`window\[Symbol\.for\(["']ApolloSSRDataTransport["']\)\]\.push\(`)
	productUnquotedKeyRegex = regexp.MustCompile(`([\{\[,\s])([A-Za-z_$][A-Za-z0-9_$]*)(\s*:)`)
	productUndefinedRegex   = regexp.MustCompile(`\bundefined\b`)
	productUUIDRegex        = regexp.MustCompile(`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$`)
	productCommentLinkRegex = regexp.MustCompile(`(?is)<a\b([^>]*)>`)
	productCloudflareRegex  = regexp.MustCompile(`(?is)(cf-browser-verification|cf-chl-|cf-turnstile|cf-ray|cdn-cgi/challenge-platform/h/|Just a moment\.\.\.|Attention Required! \| Cloudflare|Enable JavaScript and cookies to continue)`)
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

type productPage struct {
	Body       []byte
	Header     http.Header
	StatusCode int
	Base       string
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
	page, err := client.fetchProductPage(path)
	if err != nil {
		return ProductDetails{}, err
	}
	if productPageBlocked(page) {
		return ProductDetails{}, fmt.Errorf("Product Hunt public page fetch was blocked by Cloudflare")
	}
	details := parseProductDetailsPage(page.Body, page.Base, slug)
	if details.ProductURL == "" {
		details.ProductURL = page.Base + path
	}
	if details.Slug == "" {
		details.Slug = slug
	}
	return details, nil
}

func (client Client) FetchProductComments(input ProductCommentsInput) (ProductComments, error) {
	if input.Limit <= 0 {
		return ProductComments{}, fmt.Errorf("--limit must be a positive integer")
	}
	if input.Depth <= 0 {
		return ProductComments{}, fmt.Errorf("--depth must be a positive integer")
	}
	path, slug, err := productDetailsPath(ProductDetailsInput{URL: input.URL, Slug: input.Slug})
	if err != nil {
		return ProductComments{}, err
	}
	page, err := client.fetchProductPage(path)
	if err != nil {
		return ProductComments{}, err
	}
	if productPageBlocked(page) {
		return ProductComments{}, fmt.Errorf("Product Hunt public page fetch was blocked by Cloudflare")
	}
	comments, err := client.fetchProductCommentsPages(page, path, slug, input.Limit, input.Depth)
	if err != nil {
		return ProductComments{}, err
	}
	comments.IncludeHTML = input.IncludeHTML
	if comments.ProductURL == "" {
		comments.ProductURL = page.Base + path
	}
	return comments, nil
}

func (client Client) fetchProductCommentsPages(firstPage productPage, path string, slug string, limit int, depth int) (ProductComments, error) {
	collector := newProductCommentCollector()
	comments := parseProductCommentsPage(firstPage.Body, firstPage.Base, slug, limit, depth, collector)
	visited := map[string]bool{path: true}
	for _, nextPath := range productCommentPageLinks(firstPage.Body, path, slug) {
		if len(visited) >= productCommentsPageCap {
			break
		}
		if visited[nextPath] {
			continue
		}
		visited[nextPath] = true
		page, err := client.fetchProductPage(nextPath)
		if err != nil {
			return ProductComments{}, err
		}
		if productPageBlocked(page) {
			break
		}
		comments = parseProductCommentsPage(page.Body, page.Base, slug, limit, depth, collector)
	}
	return comments, nil
}

func (client Client) fetchProductPage(path string) (productPage, error) {
	base := strings.TrimRight(valueOrDefault(client.ProductWebBase, defaultProductWebBase), "/")
	fetchPath := strings.Replace(path, "#comments", "", 1)
	req, err := http.NewRequest(http.MethodGet, base+fetchPath, nil)
	if err != nil {
		return productPage{}, err
	}
	client.setProductPageHeaders(req)
	page, err := client.doProductPage(req, base)
	if err != nil {
		return productPage{}, err
	}
	return page, nil
}

func (client Client) doProductPage(req *http.Request, base string) (productPage, error) {
	httpClient := client.HTTP
	if httpClient == nil {
		httpClient = http.DefaultClient
	}
	res, err := httpClient.Do(req)
	if err != nil {
		return productPage{}, err
	}
	defer res.Body.Close()
	body, err := io.ReadAll(res.Body)
	if err != nil {
		return productPage{}, err
	}
	page := productPage{Body: body, Header: res.Header, StatusCode: res.StatusCode, Base: base}
	if res.StatusCode < 200 || res.StatusCode > 299 {
		if productPageBlocked(page) {
			return page, nil
		}
		return productPage{}, requestError{StatusCode: res.StatusCode, Body: responseSnippet(body), RetryAfter: res.Header.Get("Retry-After")}
	}
	return page, nil
}

func (client Client) setProductPageHeaders(req *http.Request) {
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,image/avif,image/webp,*/*;q=0.8")
	req.Header.Set("Accept-Language", "en-US,en;q=0.9")
	req.Header.Set("Cache-Control", "no-cache")
	req.Header.Set("Pragma", "no-cache")
	req.Header.Set("Sec-Fetch-Dest", "document")
	req.Header.Set("Sec-Fetch-Mode", "navigate")
	req.Header.Set("Sec-Fetch-Site", "none")
	req.Header.Set("Sec-Fetch-User", "?1")
	req.Header.Set("Upgrade-Insecure-Requests", "1")
	req.Header.Set("User-Agent", client.productPageUserAgent())
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

func parseProductCommentsPage(body []byte, base string, requestedSlug string, limit int, depth int, collector *productCommentCollector) ProductComments {
	content := string(body)
	details := ProductDetails{Slug: requestedSlug, Source: productHuntPageSource}
	applyProductJSONLD(&details, content)
	applyProductEmbeddedData(&details, content)
	applyProductMeta(&details, content)
	finalizeProductDetails(&details, base)
	if collector == nil {
		collector = newProductCommentCollector()
	}
	collector.hasNextPage = false
	for _, payload := range productEmbeddedPayloads(content) {
		collector.walk(payload)
	}
	comments := ProductComments{
		ProductName:   details.Name,
		ProductURL:    details.ProductURL,
		CommentsCount: details.CommentsCount,
		Source:        productHuntPageSource,
	}
	comments.Comments = limitProductCommentsDepth(limitProductComments(collector.roots, limit), depth)
	comments.ShownComments = countProductComments(comments.Comments)
	comments.Complete = comments.CommentsCount != 0 && comments.ShownComments >= comments.CommentsCount && !collector.hasNextPage
	return comments
}

func newProductCommentCollector() *productCommentCollector {
	return &productCommentCollector{seen: make(map[string]ProductComment)}
}

type productCommentCollector struct {
	roots       []ProductComment
	seen        map[string]ProductComment
	hasNextPage bool
}

func (collector *productCommentCollector) walk(value any) {
	switch typed := value.(type) {
	case map[string]any:
		if stringValue(typed["__typename"]) == "Comment" {
			comment := collector.commentFromMap(typed, "", 0)
			collector.addRoot(comment)
			return
		}
		if value, ok := typed["hasNextPage"].(bool); ok {
			collector.hasNextPage = value
		}
		for _, nested := range typed {
			collector.walk(nested)
		}
	case []any:
		for _, nested := range typed {
			collector.walk(nested)
		}
	}
}

func (collector *productCommentCollector) addRoot(comment ProductComment) {
	if !productCommentHasContent(comment) {
		return
	}
	existing, ok := collector.seen[comment.ID]
	if ok && !productCommentRicher(comment, existing) {
		return
	}
	collector.seen[comment.ID] = comment
	if ok {
		replaceProductComment(collector.roots, comment)
		return
	}
	collector.roots = append(collector.roots, comment)
}

func (collector *productCommentCollector) addReply(parent *ProductComment, reply ProductComment) {
	if !productCommentHasContent(reply) {
		return
	}
	existing, ok := collector.seen[reply.ID]
	if ok && !productCommentRicher(reply, existing) {
		return
	}
	collector.seen[reply.ID] = reply
	if ok && replaceProductComment(collector.roots, reply) {
		return
	}
	parent.Replies = append(parent.Replies, reply)
}

func (collector *productCommentCollector) commentFromMap(data map[string]any, parentID string, depth int) ProductComment {
	id := firstString(data, "id", "databaseId", "legacyId")
	comment := ProductComment{
		ID:        id,
		ParentID:  firstString(data, "parentId", "parentID"),
		BodyHTML:  stringValue(data["bodyHtml"]),
		BodyText:  stringValue(data["bodyText"]),
		CreatedAt: firstString(data, "createdAt", "insertedAt", "postedAt"),
		Hidden:    boolValue(data["hidden"]) || boolValue(data["isHidden"]),
		Deleted:   boolValue(data["deleted"]) || boolValue(data["isDeleted"]),
		Depth:     depth,
	}
	if comment.ParentID == "" {
		comment.ParentID = parentID
	}
	if comment.BodyText == "" {
		comment.BodyText = plainProductHTML(comment.BodyHTML)
	}
	fillInt(&comment.Votes, data["votesCount"])
	fillInt(&comment.Votes, data["votes"])
	fillInt(&comment.Votes, data["score"])
	comment.AuthorName, comment.Username = productCommentAuthor(data)
	for _, replyValue := range productReplyNodes(data["replies"]) {
		replyMap, ok := replyValue.(map[string]any)
		if !ok || stringValue(replyMap["__typename"]) != "Comment" {
			continue
		}
		reply := collector.commentFromMap(replyMap, id, depth+1)
		collector.addReply(&comment, reply)
	}
	return comment
}

func productCommentHasContent(comment ProductComment) bool {
	return comment.ID != "" && (comment.BodyText != "" || comment.BodyHTML != "" || comment.AuthorName != "" || comment.Username != "" || comment.CreatedAt != "" || comment.Votes != 0 || comment.Hidden || comment.Deleted)
}

func productCommentRicher(candidate ProductComment, existing ProductComment) bool {
	return productCommentScore(candidate) > productCommentScore(existing)
}

func productCommentScore(comment ProductComment) int {
	score := len(comment.BodyText) + len(comment.BodyHTML) + len(comment.AuthorName) + len(comment.Username) + len(comment.CreatedAt)
	if comment.Votes != 0 {
		score += 4
	}
	if comment.Hidden || comment.Deleted {
		score += 2
	}
	score += countProductComments(comment.Replies) * 8
	return score
}

func replaceProductComment(comments []ProductComment, replacement ProductComment) bool {
	for index := range comments {
		if comments[index].ID == replacement.ID {
			comments[index] = replacement
			return true
		}
		if replaceProductComment(comments[index].Replies, replacement) {
			return true
		}
	}
	return false
}

func productCommentAuthor(data map[string]any) (string, string) {
	for _, key := range []string{"user", "author", "maker"} {
		user, ok := data[key].(map[string]any)
		if !ok {
			continue
		}
		return firstString(user, "name"), firstString(user, "username", "slug")
	}
	return "", ""
}

func productReplyNodes(value any) []any {
	replies, ok := value.(map[string]any)
	if !ok {
		return nil
	}
	edges, ok := replies["edges"].([]any)
	if !ok {
		return nil
	}
	nodes := make([]any, 0, len(edges))
	for _, edgeValue := range edges {
		edge, ok := edgeValue.(map[string]any)
		if !ok {
			continue
		}
		if node, ok := edge["node"]; ok {
			nodes = append(nodes, node)
		}
	}
	return nodes
}

func productCommentPageLinks(body []byte, currentPath string, slug string) []string {
	content := string(body)
	links := make([]string, 0)
	seen := make(map[string]bool)
	for _, match := range productCommentLinkRegex.FindAllStringSubmatch(content, -1) {
		attrs := parseHTMLAttrs(match[1])
		href := strings.TrimSpace(attrs["href"])
		if href == "" {
			continue
		}
		path, ok := normalizeProductCommentPagePath(href, slug)
		if !ok || path == currentPath || seen[path] {
			continue
		}
		seen[path] = true
		links = append(links, path)
	}
	return links
}

func normalizeProductCommentPagePath(href string, slug string) (string, bool) {
	parsed, err := url.Parse(html.UnescapeString(strings.TrimSpace(href)))
	if err != nil {
		return "", false
	}
	if parsed.IsAbs() {
		host := strings.ToLower(parsed.Hostname())
		if host != "producthunt.com" && host != "www.producthunt.com" {
			return "", false
		}
	}
	segments := pathSegments(parsed.Path)
	if len(segments) != 2 || segments[0] != "products" || segments[1] != slug {
		return "", false
	}
	if parsed.Query().Get("page") == "" || parsed.Fragment != "comments" {
		return "", false
	}
	path := parsed.EscapedPath()
	if parsed.RawQuery != "" {
		path += "?" + parsed.RawQuery
	}
	path += "#" + parsed.Fragment
	return path, true
}

func limitProductComments(comments []ProductComment, limit int) []ProductComment {
	remaining := limit
	return limitProductCommentsWithRemaining(comments, &remaining)
}

func limitProductCommentsWithRemaining(comments []ProductComment, remaining *int) []ProductComment {
	limited := make([]ProductComment, 0, len(comments))
	for _, comment := range comments {
		if *remaining == 0 {
			break
		}
		*remaining -= 1
		comment.Replies = limitProductCommentsWithRemaining(comment.Replies, remaining)
		limited = append(limited, comment)
	}
	return limited
}

func limitProductCommentsDepth(comments []ProductComment, maxDepth int) []ProductComment {
	limited := make([]ProductComment, 0, len(comments))
	for _, comment := range comments {
		if comment.Depth+1 >= maxDepth {
			comment.Replies = nil
		} else {
			comment.Replies = limitProductCommentsDepth(comment.Replies, maxDepth)
		}
		limited = append(limited, comment)
	}
	return limited
}

func countProductComments(comments []ProductComment) int {
	count := 0
	for _, comment := range comments {
		count++
		count += countProductComments(comment.Replies)
	}
	return count
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
	for _, raw := range productEmbeddedPayloads(content) {
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
	if stringValue(data["__typename"]) == "Product" && productMapMatchesDetails(details, data) {
		applyProductObject(details, data)
	}
	if stringValue(data["__typename"]) == "Post" || stringValue(data["__typename"]) == "Launch" {
		applyProductLaunch(details, data)
	}
	if product, ok := data["product"].(map[string]any); ok && productMapMatchesDetails(details, product) {
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
	fillInt(&details.PostsCount, data["postsCount"])
	fillInt(&details.ReviewsCount, data["reviewsCount"])
	fillFloat(&details.ReviewsRating, data["reviewsRating"])
	fillString(&details.CreatedAt, stringValue(data["createdAt"]))
	fillString(&details.PublishedAt, stringValue(data["featuredAt"]))
	fillString(&details.PublishedAt, stringValue(data["scheduledAt"]))
	fillString(&details.ScheduledAt, stringValue(data["scheduledAt"]))
	fillString(&details.UpdatedAt, stringValue(data["updatedAt"]))
	if boolValue(data["hideVotesCount"]) {
		markProductVotesHidden(details)
		return
	}
	if details.VotesHidden {
		return
	}
	votes := 0
	if intValue(data["latestScore"], &votes) || intValue(data["launchDayScore"], &votes) {
		details.Votes = votes
		details.VotesKnown = true
	}
}

func applyProductObject(details *ProductDetails, product map[string]any) {
	fillString(&details.ProductID, firstString(product, "id", "databaseId", "legacyId"))
	fillString(&details.Name, stringValue(product["name"]))
	fillString(&details.Slug, stringValue(product["slug"]))
	fillString(&details.WebsiteURL, stringValue(product["websiteUrl"]))
	fillString(&details.CleanDomain, productCleanDomain(stringValue(product["cleanUrl"])))
	fillString(&details.CleanDomain, productCleanDomain(stringValue(product["cleanDomain"])))
	fillString(&details.LogoURL, productImageURL(firstString(product, "logoUuid", "logoUUID")))
	fillString(&details.ThumbnailURL, productImageURL(firstString(product, "thumbnailUuid", "thumbnailUUID")))
	fillInt(&details.PostsCount, product["postsCount"])
}

func productMapMatchesDetails(details *ProductDetails, product map[string]any) bool {
	slug := stringValue(product["slug"])
	return details.Slug == "" || slug == "" || slug == details.Slug
}

func applyProductLaunch(details *ProductDetails, launch map[string]any) {
	fillString(&details.PostID, firstString(launch, "id", "databaseId", "legacyId"))
	fillString(&details.PostSlug, stringValue(launch["slug"]))
	fillString(&details.LaunchName, stringValue(launch["name"]))
	fillString(&details.LaunchState, stringValue(launch["state"]))
	fillString(&details.Tagline, stringValue(launch["tagline"]))
	if description := stringValue(launch["description"]); description != "" {
		details.Description = description
	}
	fillInt(&details.LaunchNumber, launch["launchNumber"])
	fillInt(&details.DailyRank, launch["dailyRank"])
	fillInt(&details.WeeklyRank, launch["weeklyRank"])
	fillInt(&details.MonthlyRank, launch["monthlyRank"])
	fillString(&details.CreatedAt, stringValue(launch["createdAt"]))
	fillString(&details.PublishedAt, stringValue(launch["featuredAt"]))
	fillString(&details.PublishedAt, stringValue(launch["scheduledAt"]))
	fillString(&details.ScheduledAt, stringValue(launch["scheduledAt"]))
	fillString(&details.UpdatedAt, stringValue(launch["updatedAt"]))
	fillInt(&details.CommentsCount, launch["commentsCount"])
	fillInt(&details.ReviewsCount, launch["reviewsCount"])
	fillFloat(&details.ReviewsRating, launch["reviewsRating"])
	if boolValue(launch["hideVotesCount"]) {
		markProductVotesHidden(details)
		return
	}
	if details.VotesHidden {
		return
	}
	votes := 0
	if intValue(launch["latestScore"], &votes) || intValue(launch["launchDayScore"], &votes) {
		details.Votes = votes
		details.VotesKnown = true
	}
}

func markProductVotesHidden(details *ProductDetails) {
	details.VotesHidden = true
	details.VotesKnown = false
	details.Votes = 0
}

func applyProductMakers(details *ProductDetails, makers []any) {
	for _, makerValue := range makers {
		makerMap, ok := makerValue.(map[string]any)
		if !ok {
			continue
		}
		maker := ProductMaker{Name: stringValue(makerMap["name"]), Username: stringValue(makerMap["username"]), URL: stringValue(makerMap["url"]), Headline: stringValue(makerMap["headline"]), ImageURL: firstString(makerMap, "imageUrl", "avatarUrl")}
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
		if urlValue == "" {
			urlValue = productImageURL(firstString(mediaMap, "uuid", "imageUuid", "imageUUID"))
		}
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

func productEmbeddedPayloads(content string) []any {
	var payloads []any
	for _, object := range productApolloPushObjects(content) {
		payload, ok := parseProductEmbeddedLiteral(object)
		if ok {
			payloads = append(payloads, payload)
		}
	}
	for _, start := range productDataObjectStarts(content) {
		object, ok := balancedJSONObject(content, start)
		if !ok {
			continue
		}
		payload, ok := parseProductEmbeddedLiteral(object)
		if ok {
			payloads = append(payloads, payload)
		}
	}
	return payloads
}

func productApolloPushObjects(content string) []string {
	matches := productApolloPushRegex.FindAllStringIndex(content, -1)
	objects := make([]string, 0, len(matches))
	for _, match := range matches {
		object, ok := balancedJSValue(content, match[1])
		if ok {
			objects = append(objects, object)
		}
	}
	return objects
}

func parseProductEmbeddedLiteral(value string) (any, bool) {
	normalized := productUndefinedRegex.ReplaceAllString(value, "null")
	normalized = productUnquotedKeyRegex.ReplaceAllString(normalized, `$1"$2"$3`)
	var raw any
	decoder := json.NewDecoder(strings.NewReader(normalized))
	decoder.UseNumber()
	if decoder.Decode(&raw) != nil {
		return nil, false
	}
	return raw, true
}

func balancedJSValue(content string, start int) (string, bool) {
	if start < 0 || start >= len(content) {
		return "", false
	}
	for start < len(content) && (content[start] == ' ' || content[start] == '\n' || content[start] == '\t' || content[start] == '\r') {
		start++
	}
	if start >= len(content) || (content[start] != '{' && content[start] != '[') {
		return "", false
	}
	open := content[start]
	close := byte('}')
	if open == '[' {
		close = ']'
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
		if char == open {
			depth++
			continue
		}
		if char == close {
			depth--
			if depth == 0 {
				return content[start : index+1], true
			}
		}
	}
	return "", false
}

func productPageBlocked(page productPage) bool {
	server := strings.ToLower(page.Header.Get("Server"))
	cfRay := page.Header.Get("CF-Ray") != ""
	cfMitigated := strings.ToLower(page.Header.Get("CF-Mitigated"))
	statusBlocked := page.StatusCode == http.StatusForbidden || page.StatusCode == http.StatusTooManyRequests || page.StatusCode == http.StatusServiceUnavailable
	headersBlocked := strings.Contains(server, "cloudflare") && statusBlocked || cfRay && statusBlocked || cfMitigated == "challenge"
	bodyBlocked := productCloudflareRegex.Match(page.Body)
	return headersBlocked || statusBlocked && bodyBlocked
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
	maker.Headline = strings.TrimSpace(maker.Headline)
	maker.ImageURL = strings.TrimSpace(maker.ImageURL)
	if maker.Name == "" && maker.Username == "" && maker.URL == "" && maker.Headline == "" && maker.ImageURL == "" {
		return
	}
	key := maker.Name + "\x00" + maker.Username + "\x00" + maker.URL + "\x00" + maker.Headline + "\x00" + maker.ImageURL
	for _, existing := range details.Makers {
		if existing.Name+"\x00"+existing.Username+"\x00"+existing.URL+"\x00"+existing.Headline+"\x00"+existing.ImageURL == key {
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
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return ""
	}
	parsed, err := url.Parse(trimmed)
	if err != nil {
		return ""
	}
	return strings.TrimPrefix(parsed.Hostname(), "www.")
}

func productCleanDomain(value string) string {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return ""
	}
	if strings.Contains(trimmed, "://") {
		return cleanDomain(trimmed)
	}
	return cleanDomain("https://" + trimmed)
}

func productImageURL(uuid string) string {
	trimmed := strings.TrimSpace(uuid)
	if trimmed == "" {
		return ""
	}
	if strings.Contains(trimmed, "://") {
		return trimmed
	}
	if !productUUIDRegex.MatchString(trimmed) {
		return ""
	}
	return "https://ph-files.imgix.net/" + trimmed + "?auto=format"
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

func (client Client) productPageUserAgent() string {
	return valueOrDefault(client.ProductUserAgent, defaultProductPageAgent)
}
