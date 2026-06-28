package output

import (
	"fmt"
	"io"

	"github.com/bvgroup-co/hacker-feeds-go-cli/internal/feeds"
	"github.com/bvgroup-co/hacker-feeds-go-cli/internal/i18n"
)

func GitHub(writer io.Writer, labels i18n.Labels, since string, repos []feeds.GitHubRepo) {
	header(writer, labels.GitHub.Header)
	for _, repo := range repos {
		fmt.Fprintf(writer, "%s: %s | %s: %s | %s: %d | %s: %d\n", labels.GitHub.Repo, repo.Repo, labels.GitHub.Language, repo.Language, labels.GitHub.Stars, repo.Stars, labels.GitHub.StarsForSince(since), repo.AddedStars)
		if repo.Desc != "" {
			fmt.Fprintf(writer, "%s: %s\n", labels.GitHub.Desc, repo.Desc)
		}
		fmt.Fprintf(writer, "%s: %s\n", labels.GitHub.Author, repo.Author)
		fmt.Fprintf(writer, "%s: %s\n", labels.GitHub.Link, repo.Link)
		separator(writer)
	}
}

func News(writer io.Writer, labels i18n.Labels, items []feeds.NewsItem) {
	header(writer, labels.News.Header)
	for _, item := range items {
		fmt.Fprintf(writer, "%s: %s\n", labels.News.Title, item.Title)
		fmt.Fprintf(writer, "ID: %d | Author: %s | Score: %d | Comments: %d\n", item.ID, item.Author, item.Score, item.Descendants)
		if item.URL != "" {
			fmt.Fprintf(writer, "%s: %s\n", labels.News.URL, item.URL)
		}
		separator(writer)
	}
}

func NewsDiscussion(writer io.Writer, labels i18n.Labels, discussion feeds.NewsDiscussion) {
	header(writer, labels.News.DiscussionHeader)
	item := discussion.Item
	fmt.Fprintf(writer, "ID: %d | %s: %s | Author: %s | Score: %d | Comments: %d\n", item.ID, labels.News.Title, item.Title, item.Author, item.Score, item.Descendants)
	if item.URL != "" {
		fmt.Fprintf(writer, "%s: %s\n", labels.News.URL, item.URL)
	}
	separator(writer)
	for _, comment := range discussion.Comments {
		writeNewsComment(writer, comment)
	}
}

func writeNewsComment(writer io.Writer, comment feeds.NewsComment) {
	indent := ""
	for range comment.Depth {
		indent += "  "
	}
	fmt.Fprintf(writer, "%sAuthor: %s | ID: %d\n", indent, comment.Author, comment.ID)
	text := comment.Text
	if comment.Deleted {
		text = "[deleted]"
	}
	if comment.Dead {
		text = "[dead]"
	}
	fmt.Fprintf(writer, "%sComment: %s\n", indent, text)
	for _, child := range comment.Children {
		writeNewsComment(writer, child)
	}
	if comment.Depth == 0 {
		separator(writer)
	}
}

func Product(writer io.Writer, labels i18n.Labels, products []feeds.Product) {
	header(writer, labels.Product.Header)
	for _, product := range products {
		fmt.Fprintf(writer, "%s: %s | Source: %s\n", labels.Product.Name, product.Name, product.Source)
		if product.VotesKnown {
			fmt.Fprintf(writer, "%s: %d\n", labels.Product.Votes, product.Votes)
		} else {
			fmt.Fprintf(writer, "%s: unavailable from public feed\n", labels.Product.Votes)
		}
		fmt.Fprintf(writer, "%s: %s\n", labels.Product.Description, product.Description)
		fmt.Fprintf(writer, "%s: %s\n", labels.Product.ProductURL, product.URL)
		fmt.Fprintf(writer, "%s: %s\n", labels.Product.Website, product.Website)
		separator(writer)
	}
}

func ProductDetails(writer io.Writer, labels i18n.Labels, details feeds.ProductDetails) {
	header(writer, labels.Product.DetailsHeader)
	if details.ProductID != "" {
		fmt.Fprintf(writer, "Product ID: %s\n", details.ProductID)
	}
	if details.PostID != "" {
		fmt.Fprintf(writer, "Post ID: %s\n", details.PostID)
	}
	if details.PostSlug != "" {
		fmt.Fprintf(writer, "Launch slug: %s\n", details.PostSlug)
	}
	if details.Name != "" {
		fmt.Fprintf(writer, "%s: %s\n", labels.Product.Name, details.Name)
	}
	if details.LaunchName != "" {
		fmt.Fprintf(writer, "Launch name: %s\n", details.LaunchName)
	}
	if details.LaunchState != "" {
		fmt.Fprintf(writer, "Launch state: %s\n", details.LaunchState)
	}
	if details.LaunchNumber != 0 {
		fmt.Fprintf(writer, "Launch number: %d\n", details.LaunchNumber)
	}
	if details.DailyRank != 0 {
		fmt.Fprintf(writer, "Daily rank: %d\n", details.DailyRank)
	}
	if details.WeeklyRank != 0 {
		fmt.Fprintf(writer, "Weekly rank: %d\n", details.WeeklyRank)
	}
	if details.MonthlyRank != 0 {
		fmt.Fprintf(writer, "Monthly rank: %d\n", details.MonthlyRank)
	}
	if details.Tagline != "" {
		fmt.Fprintf(writer, "Tagline: %s\n", details.Tagline)
	}
	if details.Description != "" {
		fmt.Fprintf(writer, "%s: %s\n", labels.Product.Description, details.Description)
	}
	if details.ProductURL != "" {
		fmt.Fprintf(writer, "%s: %s\n", labels.Product.ProductURL, details.ProductURL)
	}
	if details.WebsiteURL != "" {
		fmt.Fprintf(writer, "%s: %s\n", labels.Product.Website, details.WebsiteURL)
	}
	if details.CleanDomain != "" {
		fmt.Fprintf(writer, "Clean domain: %s\n", details.CleanDomain)
	}
	if details.LogoURL != "" {
		fmt.Fprintf(writer, "Logo URL: %s\n", details.LogoURL)
	}
	if details.ThumbnailURL != "" {
		fmt.Fprintf(writer, "Thumbnail URL: %s\n", details.ThumbnailURL)
	}
	if details.VotesKnown {
		fmt.Fprintf(writer, "%s: %d\n", labels.Product.Votes, details.Votes)
	} else {
		fmt.Fprintf(writer, "%s: unavailable from public page\n", labels.Product.Votes)
	}
	if details.CommentsCount != 0 {
		fmt.Fprintf(writer, "Comments count: %d\n", details.CommentsCount)
	}
	if details.PostsCount != 0 {
		fmt.Fprintf(writer, "Posts count: %d\n", details.PostsCount)
	}
	if details.ReviewsCount != 0 {
		fmt.Fprintf(writer, "Reviews count: %d\n", details.ReviewsCount)
	}
	if details.ReviewsRating != 0 {
		fmt.Fprintf(writer, "Reviews rating: %.1f\n", details.ReviewsRating)
	}
	if details.FollowersCount != 0 {
		fmt.Fprintf(writer, "Followers count: %d\n", details.FollowersCount)
	}
	if details.CreatedAt != "" {
		fmt.Fprintf(writer, "Created at: %s\n", details.CreatedAt)
	}
	if details.PublishedAt != "" {
		fmt.Fprintf(writer, "Featured at: %s\n", details.PublishedAt)
	}
	if details.ScheduledAt != "" {
		fmt.Fprintf(writer, "Scheduled at: %s\n", details.ScheduledAt)
	}
	if details.UpdatedAt != "" {
		fmt.Fprintf(writer, "Updated at: %s\n", details.UpdatedAt)
	}
	writeProductMakers(writer, details.Makers)
	writeProductTopics(writer, details.Topics)
	writeProductMedia(writer, details.Media)
	fmt.Fprintf(writer, "Source: %s\n", details.Source)
	separator(writer)
}

func ProductComments(writer io.Writer, labels i18n.Labels, comments feeds.ProductComments) {
	header(writer, "Product Hunt Comments")
	if comments.ProductName != "" {
		fmt.Fprintf(writer, "%s: %s\n", labels.Product.Name, comments.ProductName)
	}
	if comments.ProductURL != "" {
		fmt.Fprintf(writer, "%s: %s\n", labels.Product.ProductURL, comments.ProductURL)
	}
	fmt.Fprintf(writer, "Comments count: %d\n", comments.CommentsCount)
	fmt.Fprintf(writer, "Shown comments: %d\n", comments.ShownComments)
	complete := "no"
	if comments.Complete {
		complete = "yes"
	}
	fmt.Fprintf(writer, "Complete: %s\n", complete)
	fmt.Fprintln(writer, "Note: Product Hunt public HTML embeds only the initial comment page; more comments may exist.")
	separator(writer)
	for _, comment := range comments.Comments {
		writeProductComment(writer, comment, comments.IncludeHTML)
	}
}

func writeProductComment(writer io.Writer, comment feeds.ProductComment, includeHTML bool) {
	indent := ""
	for range comment.Depth {
		indent += "  "
	}
	author := comment.AuthorName
	if comment.Username != "" {
		author += " (@" + comment.Username + ")"
	}
	if author == "" {
		author = "unknown"
	}
	status := ""
	if comment.Hidden {
		status += " · hidden"
	}
	if comment.Deleted {
		status += " · deleted"
	}
	fmt.Fprintf(writer, "%s[%s] %s · %s · %d votes%s\n", indent, comment.ID, author, comment.CreatedAt, comment.Votes, status)
	if comment.BodyText != "" {
		fmt.Fprintf(writer, "%s%s\n", indent, comment.BodyText)
	}
	if includeHTML && comment.BodyHTML != "" {
		fmt.Fprintf(writer, "%sHTML: %s\n", indent, comment.BodyHTML)
	}
	fmt.Fprintln(writer)
	for _, reply := range comment.Replies {
		writeProductComment(writer, reply, includeHTML)
	}
}

func writeProductMakers(writer io.Writer, makers []feeds.ProductMaker) {
	if len(makers) == 0 {
		return
	}
	fmt.Fprintln(writer, "Makers:")
	for _, maker := range makers {
		name := maker.Name
		if maker.Username != "" {
			name += " (@" + maker.Username + ")"
		}
		if maker.URL != "" {
			fmt.Fprintf(writer, "- %s: %s\n", name, maker.URL)
		} else {
			fmt.Fprintf(writer, "- %s\n", name)
		}
		if maker.Headline != "" {
			fmt.Fprintf(writer, "  Headline: %s\n", maker.Headline)
		}
		if maker.ImageURL != "" {
			fmt.Fprintf(writer, "  Image URL: %s\n", maker.ImageURL)
		}
	}
}

func writeProductTopics(writer io.Writer, topics []feeds.ProductTopic) {
	if len(topics) == 0 {
		return
	}
	fmt.Fprintln(writer, "Topics:")
	for _, topic := range topics {
		if topic.Slug != "" {
			fmt.Fprintf(writer, "- %s (%s)\n", topic.Name, topic.Slug)
			continue
		}
		fmt.Fprintf(writer, "- %s\n", topic.Name)
	}
}

func writeProductMedia(writer io.Writer, media []feeds.ProductMedia) {
	if len(media) == 0 {
		return
	}
	fmt.Fprintln(writer, "Media URLs:")
	for _, item := range media {
		if item.Type != "" {
			fmt.Fprintf(writer, "- %s: %s\n", item.Type, item.URL)
			continue
		}
		fmt.Fprintf(writer, "- %s\n", item.URL)
	}
}

func Reddit(writer io.Writer, labels i18n.Labels, posts []feeds.RedditPost) {
	header(writer, labels.Reddit.Header)
	for _, post := range posts {
		fmt.Fprintf(writer, "%s: %s\n", labels.Reddit.Title, post.Title)
		fmt.Fprintf(writer, "ID: %s | Source: %s | %s: %s | Author: %s\n", post.ID, post.Source, labels.Reddit.Topic, post.Subreddit, post.Author)
		fmt.Fprintf(writer, "%s: %d | %s: %d | Score: %d\n", labels.Reddit.Comment, post.NumComments, labels.Reddit.Votes, post.Ups, post.Score)
		if post.URL != "" {
			fmt.Fprintf(writer, "URL: %s\n", post.URL)
		}
		fmt.Fprintf(writer, "%s: %s\n", labels.Reddit.Link, post.Permalink)
		if post.Domain != "" {
			fmt.Fprintf(writer, "Domain: %s\n", post.Domain)
		}
		if post.Content != "" {
			fmt.Fprintf(writer, "%s: %s\n", labels.Reddit.Content, post.Content)
		}
		separator(writer)
	}
}

func RedditDiscussion(writer io.Writer, labels i18n.Labels, discussion feeds.RedditDiscussion) {
	header(writer, labels.Reddit.Header)
	post := discussion.Post
	if post.ID != "" || post.Subreddit != "" || post.Title != "" {
		fmt.Fprintf(writer, "ID: %s | Source: %s | %s: %s\n", post.ID, post.Source, labels.Reddit.Topic, post.Subreddit)
	}
	if post.Title != "" {
		fmt.Fprintf(writer, "%s: %s\n", labels.Reddit.Title, post.Title)
		fmt.Fprintf(writer, "%s: %d | %s: %d | Score: %d | %s: %s | Author: %s\n", labels.Reddit.Comment, post.NumComments, labels.Reddit.Votes, post.Ups, post.Score, labels.Reddit.Topic, post.Subreddit, post.Author)
		if post.Permalink != "" {
			fmt.Fprintf(writer, "%s: %s\n", labels.Reddit.Link, post.Permalink)
		}
		if post.Content != "" {
			fmt.Fprintf(writer, "%s: %s\n", labels.Reddit.Content, post.Content)
		}
		separator(writer)
	} else if post.ID != "" || post.Subreddit != "" || post.Source != "" {
		separator(writer)
	}
	for _, comment := range discussion.Comments {
		writeRedditComment(writer, comment, 0)
	}
}

func writeRedditComment(writer io.Writer, comment feeds.RedditComment, depth int) {
	indent := ""
	for range depth {
		indent += "  "
	}
	if comment.More {
		fmt.Fprintf(writer, "%sMore comments: %d\n", indent, comment.Count)
		separator(writer)
		return
	}
	fmt.Fprintf(writer, "%sAuthor: %s | Score: %d | Source: %s\n", indent, comment.Author, comment.Score, comment.Source)
	if comment.Body != "" {
		fmt.Fprintf(writer, "%sComment: %s\n", indent, comment.Body)
	}
	if comment.Permalink != "" {
		fmt.Fprintf(writer, "%sLink: %s\n", indent, comment.Permalink)
	}
	for _, reply := range comment.Replies {
		writeRedditComment(writer, reply, depth+1)
	}
	if depth == 0 {
		separator(writer)
	}
}

func V2EX(writer io.Writer, labels i18n.Labels, topics []feeds.V2EXTopic) {
	header(writer, labels.V2EX.Header)
	for _, topic := range topics {
		fmt.Fprintf(writer, "%s: %s\n", labels.V2EX.Title, topic.Title)
		fmt.Fprintf(writer, "%s: %d | %s: %d | %s: %s\n", labels.V2EX.Comment, topic.Comment, labels.V2EX.Votes, topic.Votes, labels.V2EX.Topic, topic.Node)
		fmt.Fprintf(writer, "%s: %s\n", labels.V2EX.Link, topic.Link)
		if topic.Content != "" {
			fmt.Fprintf(writer, "%s: %s\n", labels.V2EX.Content, topic.Content)
		}
		separator(writer)
	}
}

func NoItems(writer io.Writer, labels i18n.Labels) {
	fmt.Fprintln(writer, labels.Shared.NoItems)
}

func header(writer io.Writer, title string) {
	fmt.Fprintln(writer, "-----------------------------------------")
	fmt.Fprintln(writer, title)
	fmt.Fprintln(writer, "-----------------------------------------")
}

func separator(writer io.Writer) {
	fmt.Fprintln(writer, "-----------------------------------------")
}
