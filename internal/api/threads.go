package api

import (
	"context"
	"net/url"
	"strconv"
)

// listEnvelope is the Graph collection shape: {data:[...], paging:{...}}.
type mediaList struct {
	Data []rawMedia `json:"data"`
	rawPaging
}

func setPaging(q url.Values, o PageOpts) {
	if o.Limit > 0 {
		q.Set("limit", strconv.Itoa(o.Limit))
	}
	if o.Cursor != "" {
		q.Set("after", o.Cursor)
	}
	if o.Since != "" {
		q.Set("since", o.Since)
	}
	if o.Until != "" {
		q.Set("until", o.Until)
	}
}

// Profile fetches a profile. userID "" or "me" → the authenticated user.
func (c *Client) Profile(ctx context.Context, userID string) (*Profile, error) {
	if userID == "" {
		userID = c.userID
	}
	q := url.Values{"fields": {profileFields}}
	var rp rawProfile
	if err := c.get(ctx, userID, q, &rp); err != nil {
		return nil, err
	}
	p := rp.toProfile()
	return &p, nil
}

// ListPosts returns the authenticated user's posts (newest first) + the next cursor.
func (c *Client) ListPosts(ctx context.Context, o PageOpts) ([]Post, string, error) {
	q := url.Values{"fields": {postFields}}
	setPaging(q, o)
	var ml mediaList
	if err := c.get(ctx, c.userID+"/threads", q, &ml); err != nil {
		return nil, "", err
	}
	return mapPosts(ml.Data), ml.nextCursor(), nil
}

// GetPost fetches one post by media id.
func (c *Client) GetPost(ctx context.Context, id string) (*Post, error) {
	q := url.Values{"fields": {postFields}}
	var rm rawMedia
	if err := c.get(ctx, id, q, &rm); err != nil {
		return nil, err
	}
	p := rm.toPost()
	return &p, nil
}

// ListReplies returns top-level replies to a post.
func (c *Client) ListReplies(ctx context.Context, mediaID string, o PageOpts) ([]Reply, string, error) {
	return c.replies(ctx, mediaID+"/replies", o, nil)
}

// Conversation returns the full reply tree for a post (reverse-chronological).
func (c *Client) Conversation(ctx context.Context, mediaID string, o PageOpts) ([]Reply, string, error) {
	return c.replies(ctx, mediaID+"/conversation", o, url.Values{"reverse": {"true"}})
}

func (c *Client) replies(ctx context.Context, path string, o PageOpts, extra url.Values) ([]Reply, string, error) {
	q := url.Values{"fields": {replyFields}}
	for k, v := range extra {
		q[k] = v
	}
	setPaging(q, o)
	var ml mediaList
	if err := c.get(ctx, path, q, &ml); err != nil {
		return nil, "", err
	}
	out := make([]Reply, 0, len(ml.Data))
	for i := range ml.Data {
		out = append(out, ml.Data[i].toReply())
	}
	return out, ml.nextCursor(), nil
}

// Mentions returns public posts mentioning the authenticated user.
func (c *Client) Mentions(ctx context.Context, o PageOpts) ([]Post, string, error) {
	q := url.Values{"fields": {postFields}}
	setPaging(q, o)
	var ml mediaList
	if err := c.get(ctx, c.userID+"/mentions", q, &ml); err != nil {
		return nil, "", err
	}
	return mapPosts(ml.Data), ml.nextCursor(), nil
}

// Search runs a keyword or topic-tag search. It returns the results, the next cursor, and a
// best-effort scope ("public" if any result is authored by someone other than the authed user,
// else "self") so an agent knows whether advanced access was actually in effect.
func (c *Client) Search(ctx context.Context, o SearchOpts) ([]Post, string, string, error) {
	q := url.Values{"fields": {postFields}, "q": {o.Query}, "search_type": {"RECENT"}}
	if o.Tag {
		q.Set("search_mode", "TAG")
	}
	if mt := apiMediaType(o.MediaType); mt != "" {
		q.Set("media_type", mt)
	}
	setPaging(q, o.PageOpts)
	var ml mediaList
	if err := c.get(ctx, "keyword_search", q, &ml); err != nil {
		return nil, "", "", err
	}
	posts := mapPosts(ml.Data)
	scope := c.scopeOf(ctx, posts)
	return posts, ml.nextCursor(), scope, nil
}

// scopeOf reports "public" if any post is authored by someone other than the authed user.
func (c *Client) scopeOf(ctx context.Context, posts []Post) string {
	if len(posts) == 0 {
		return "self" // inconclusive; treat as self (advanced access may simply have no results)
	}
	self, err := c.selfUsername(ctx)
	if err != nil || self == "" {
		return "self"
	}
	for i := range posts {
		if posts[i].Username != "" && posts[i].Username != self {
			return "public"
		}
	}
	return "self"
}

var selfUsernameCache string

func (c *Client) selfUsername(ctx context.Context) (string, error) {
	if selfUsernameCache != "" {
		return selfUsernameCache, nil
	}
	p, err := c.Profile(ctx, "me")
	if err != nil {
		return "", err
	}
	selfUsernameCache = p.Username
	return selfUsernameCache, nil
}

func mapPosts(raw []rawMedia) []Post {
	out := make([]Post, 0, len(raw))
	for i := range raw {
		out = append(out, raw[i].toPost())
	}
	return out
}

func apiMediaType(s string) string {
	switch s {
	case "text":
		return "TEXT_POST"
	case "image":
		return "IMAGE"
	case "video":
		return "VIDEO"
	default:
		return ""
	}
}

// --- insights ---------------------------------------------------------------

type insightsResp struct {
	Data []struct {
		Name       string `json:"name"`
		TotalValue struct {
			Value      *int64 `json:"value"`
			Breakdowns []any  `json:"breakdowns"`
		} `json:"total_value"`
		Values []struct {
			Value int64 `json:"value"`
		} `json:"values"`
	} `json:"data"`
}

func (r *insightsResp) collapse() map[string]any {
	out := map[string]any{}
	for _, d := range r.Data {
		if d.Name == "follower_demographics" {
			out["followerDemographics"] = d.TotalValue.Breakdowns
			continue
		}
		key := insightKey(d.Name)
		switch {
		case d.TotalValue.Value != nil:
			out[key] = *d.TotalValue.Value
		case len(d.Values) > 0:
			out[key] = d.Values[len(d.Values)-1].Value
		default:
			out[key] = nil
		}
	}
	return out
}

func insightKey(name string) string {
	switch name {
	case "followers_count":
		return "followersCount"
	default:
		return name // views, likes, replies, reposts, quotes, shares
	}
}

// PostInsights returns per-post metrics.
func (c *Client) PostInsights(ctx context.Context, mediaID string) (map[string]any, error) {
	q := url.Values{"metric": {"views,likes,replies,reposts,quotes,shares"}}
	var ir insightsResp
	if err := c.get(ctx, mediaID+"/insights", q, &ir); err != nil {
		return nil, err
	}
	return withDefaults(ir.collapse(), "views", "likes", "replies", "reposts", "quotes", "shares"), nil
}

// AccountInsights returns account-level metrics + demographics.
func (c *Client) AccountInsights(ctx context.Context, o AccountInsightsOpts) (map[string]any, error) {
	metrics := o.Metrics
	if len(metrics) == 0 {
		metrics = []string{"views", "likes", "replies", "reposts", "quotes", "followers_count"}
	}
	q := url.Values{"metric": {joinComma(metrics)}}
	if o.Since != "" {
		q.Set("since", o.Since)
	}
	if o.Until != "" {
		q.Set("until", o.Until)
	}
	var ir insightsResp
	if err := c.get(ctx, c.userID+"/threads_insights", q, &ir); err != nil {
		return nil, err
	}
	return ir.collapse(), nil
}

func withDefaults(m map[string]any, keys ...string) map[string]any {
	for _, k := range keys {
		if _, ok := m[k]; !ok {
			m[k] = nil
		}
	}
	return m
}

func joinComma(s []string) string {
	out := ""
	for i, v := range s {
		if i > 0 {
			out += ","
		}
		out += v
	}
	return out
}

// --- publishing quota -------------------------------------------------------

// PublishingLimit returns the remaining 24h post-publish quota.
func (c *Client) PublishingLimit(ctx context.Context) (*Quota, error) {
	q := url.Values{"fields": {"quota_usage,config"}}
	var resp struct {
		Data []struct {
			QuotaUsage int `json:"quota_usage"`
			Config     struct {
				QuotaTotal int `json:"quota_total"`
			} `json:"config"`
		} `json:"data"`
	}
	if err := c.get(ctx, c.userID+"/threads_publishing_limit", q, &resp); err != nil {
		return nil, err
	}
	if len(resp.Data) == 0 {
		return &Quota{}, nil
	}
	d := resp.Data[0]
	return &Quota{Used: d.QuotaUsage, Total: d.Config.QuotaTotal, Remaining: d.Config.QuotaTotal - d.QuotaUsage}, nil
}
