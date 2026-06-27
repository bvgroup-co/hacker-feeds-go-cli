package feeds

import (
	"encoding/xml"
	"fmt"
	"html"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"
)

const (
	defaultProductBase      = "https://www.producthunt.com/feed"
	defaultProductUserAgent = "hacker-feeds-go-cli/dev (+https://github.com/bvgroup-co/hacker-feeds-go-cli)"
	productHuntFeedSource   = "producthunt-feed"
)

var (
	productHTMLTagPattern = regexp.MustCompile(`<[^>]*>`)
	productParagraphRegex = regexp.MustCompile(`(?is)<p\b[^>]*>(.*?)</p>`)
	productLinkRegex      = regexp.MustCompile(`(?is)<a\b([^>]*)>(.*?)</a>`)
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
