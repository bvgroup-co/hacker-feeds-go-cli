package output

import (
	"strings"
	"testing"

	"github.com/bvgroup-co/hacker-feeds-go-cli/internal/feeds"
	"github.com/bvgroup-co/hacker-feeds-go-cli/internal/i18n"
)

func TestNewsDiscussionSeparatesTopLevelSubtrees(t *testing.T) {
	discussion := feeds.NewsDiscussion{
		Item: feeds.NewsItem{ID: 1, Title: "Story", Author: "alice", Score: 5, Descendants: 2},
		Comments: []feeds.NewsComment{
			{
				ID:     2,
				Author: "bob",
				Text:   "parent",
				Depth:  0,
				Children: []feeds.NewsComment{
					{ID: 3, Author: "carol", Text: "child", Depth: 1},
				},
			},
		},
	}
	var builder strings.Builder
	NewsDiscussion(&builder, i18n.For("en"), discussion)
	output := builder.String()
	parent := strings.Index(output, "Comment: parent")
	child := strings.Index(output, "  Comment: child")
	separatorAfterChild := strings.Index(output[child:], "-----------------------------------------")
	separatorBetween := strings.Index(output[parent:child], "-----------------------------------------")
	if parent == -1 || child == -1 || separatorAfterChild == -1 || separatorBetween != -1 {
		t.Fatalf("output = %q", output)
	}
}

func TestProductShowsUnknownVotesExplicitly(t *testing.T) {
	var builder strings.Builder
	Product(&builder, i18n.For("en"), []feeds.Product{{Name: "Prod", Source: "producthunt-feed", VotesKnown: false}})
	output := builder.String()
	if strings.Contains(output, "Votes: 0") || !strings.Contains(output, "Votes: unavailable from public feed") || !strings.Contains(output, "Source: producthunt-feed") {
		t.Fatalf("output = %q", output)
	}
}
