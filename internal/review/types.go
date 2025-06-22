package review

type ReviewComment struct {
	Path string
	Line int
	Body string
	Side string
}

type ReviewResult struct {
	Summary  string
	Comments []ReviewComment
}

type PRSizeCheck struct {
	ShouldReview   bool
	WarningMessage string
	SkipMessage    string
}
