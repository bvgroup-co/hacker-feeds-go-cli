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
		fmt.Fprintf(writer, "%s: %s | %s: %d\n", labels.Product.Name, product.Name, labels.Product.Votes, product.Votes)
		fmt.Fprintf(writer, "%s: %s\n", labels.Product.Description, product.Description)
		fmt.Fprintf(writer, "%s: %s\n", labels.Product.ProductURL, product.URL)
		fmt.Fprintf(writer, "%s: %s\n", labels.Product.Website, product.Website)
		separator(writer)
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
