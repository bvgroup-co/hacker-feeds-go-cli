package i18n

type Labels struct {
	GitHub  GitHubLabels
	News    NewsLabels
	Product ProductLabels
	Reddit  ForumLabels
	V2EX    ForumLabels
	Shared  SharedLabels
	Config  ConfigLabels
}

type GitHubLabels struct {
	Header         string
	Repo           string
	Desc           string
	Link           string
	Language       string
	Stars          string
	Author         string
	StarsThisWeek  string
	StarsThisDay   string
	StarsThisMonth string
}

type NewsLabels struct {
	Header           string
	DiscussionHeader string
	Title            string
	URL              string
}

type ProductLabels struct {
	Header      string
	Name        string
	Description string
	ProductURL  string
	Website     string
	Votes       string
}

type ForumLabels struct {
	Header  string
	Title   string
	Content string
	Comment string
	Link    string
	Votes   string
	Topic   string
}

type SharedLabels struct {
	Fetching string
	Failure  string
	NoItems  string
}

type ConfigLabels struct {
	Saved  string
	Failed string
}

func For(lang string) Labels {
	if lang == "zh" {
		return zh
	}
	return en
}

func ValidLang(lang string) bool {
	return lang == "en" || lang == "zh"
}

func (labels GitHubLabels) StarsForSince(since string) string {
	switch since {
	case "daily":
		return labels.StarsThisDay
	case "weekly":
		return labels.StarsThisWeek
	case "monthly":
		return labels.StarsThisMonth
	default:
		panic("invalid github since: " + since)
	}
}

var en = Labels{
	GitHub: GitHubLabels{
		Header:         "GitHub Trending List",
		Repo:           "Repo",
		Desc:           "Desc",
		Link:           "Link",
		Language:       "Language",
		Stars:          "Stars",
		Author:         "Author",
		StarsThisWeek:  "Stars this week",
		StarsThisDay:   "Stars this day",
		StarsThisMonth: "Stars this month",
	},
	News: NewsLabels{
		Header:           "Hacker News List",
		DiscussionHeader: "Hacker News Discussion",
		Title:            "Title",
		URL:              "URL",
	},
	Product: ProductLabels{
		Header:      "Product Hunt List",
		Name:        "Name",
		Description: "Description",
		ProductURL:  "Product URL",
		Website:     "Website",
		Votes:       "Votes",
	},
	Reddit: ForumLabels{
		Header:  "Reddit List",
		Title:   "Title",
		Content: "Content",
		Comment: "Comment",
		Link:    "Link",
		Votes:   "Votes",
		Topic:   "Topic",
	},
	V2EX: ForumLabels{
		Header:  "V2EX Feeds List",
		Title:   "Title",
		Content: "Content",
		Comment: "Comment",
		Link:    "Link",
		Votes:   "Votes",
		Topic:   "Node",
	},
	Shared: SharedLabels{
		Fetching: "Fetching feeds...",
		Failure:  "Something error, You can contact the developer. Mail to <phillzou@gmail.com>",
		NoItems:  "The ranking has not yet been updated, you can check the past data.",
	},
	Config: ConfigLabels{
		Saved:  "Config Saved",
		Failed: "Config Failed",
	},
}

var zh = Labels{
	GitHub: GitHubLabels{
		Header:         "GitHub 榜单",
		Repo:           "仓库",
		Desc:           "描述",
		Link:           "链接",
		Language:       "语言",
		Stars:          "星标",
		Author:         "作者",
		StarsThisWeek:  "本周新增星标",
		StarsThisDay:   "本日新增星标",
		StarsThisMonth: "本月新增星标",
	},
	News: NewsLabels{
		Header:           "Hacker News 新闻",
		DiscussionHeader: "Hacker News 讨论",
		Title:            "标题",
		URL:              "链接",
	},
	Product: ProductLabels{
		Header:      "Product Hunt 榜单",
		Name:        "名称",
		Description: "描述",
		ProductURL:  "产品介绍",
		Website:     "产品网址",
		Votes:       "投票",
	},
	Reddit: ForumLabels{
		Header:  "Reddit 帖子",
		Title:   "标题",
		Content: "详情",
		Comment: "评论",
		Link:    "链接",
		Votes:   "投票",
		Topic:   "话题",
	},
	V2EX: ForumLabels{
		Header:  "V2EX 帖子",
		Title:   "标题",
		Content: "详情",
		Comment: "评论",
		Link:    "链接",
		Votes:   "Votes",
		Topic:   "节点",
	},
	Shared: SharedLabels{
		Fetching: "数据拉取中...",
		Failure:  "程序错误, 你可以发送邮件至 <phillzou@gmail.com> 联系开发者",
		NoItems:  "数据暂未更新，你可以使用 '-p' 参数查看往日榜单",
	},
	Config: ConfigLabels{
		Saved:  "配置成功",
		Failed: "配置失败",
	},
}
