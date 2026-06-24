package api

import "context"

// Threads is the surface the CLI commands depend on. *Client implements it; tests inject fakes.
// Keeping commands behind this interface keeps them unit-testable without a live API.
type Threads interface {
	Profile(ctx context.Context, userID string) (*Profile, error)
	ListPosts(ctx context.Context, o PageOpts) ([]Post, string, error)
	GetPost(ctx context.Context, id string) (*Post, error)
	Publish(ctx context.Context, r PublishReq) (*Post, error)
	Repost(ctx context.Context, id string) (string, error)
	DeletePost(ctx context.Context, id string) (*DeleteResult, error)
	ListReplies(ctx context.Context, mediaID string, o PageOpts) ([]Reply, string, error)
	Conversation(ctx context.Context, mediaID string, o PageOpts) ([]Reply, string, error)
	ManageReply(ctx context.Context, replyID string, hide bool) (*Reply, error)
	Mentions(ctx context.Context, o PageOpts) ([]Post, string, error)
	Search(ctx context.Context, o SearchOpts) ([]Post, string, string, error)
	PostInsights(ctx context.Context, mediaID string) (map[string]any, error)
	AccountInsights(ctx context.Context, o AccountInsightsOpts) (map[string]any, error)
	PublishingLimit(ctx context.Context) (*Quota, error)
}

// compile-time assertion that *Client satisfies the interface.
var _ Threads = (*Client)(nil)
